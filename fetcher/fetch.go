package fetcher

import (
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

	"dev.sum7.eu/genofire/golang-lib/file"
	"github.com/FreifunkBremen/yanic/database"
	"github.com/FreifunkBremen/yanic/lib/jsontime"
	meshviewerFFRGB "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"

	"github.com/genofire/meshviewer-collector/converter"
	"github.com/genofire/meshviewer-collector/runtime"
)

type Fetcher struct {
	*runtime.Config
	ConfigMutex sync.Mutex
	currentNode map[string]time.Time
	database    database.Connection
	closed      bool
}

func NewFetcher(config *runtime.Config, db database.Connection) *Fetcher {
	return &Fetcher{
		Config:      config,
		database:    db,
		currentNode: make(map[string]time.Time),
	}
}

func (f *Fetcher) Start() {
	f.fetch()
	timer := time.NewTimer(f.RunEvery.Duration)
	for !f.closed {
		select {
		case <-timer.C:
			f.fetch()
			timer.Reset(f.RunEvery.Duration)
		}
	}
	timer.Stop()
}

func (f *Fetcher) Stop() {
	f.closed = true
}

func (f *Fetcher) fetch() {
	now := jsontime.Now()
	// do not merge a meshviewer.json older then 6h
	ignoreMeshviewer := now.Add(-f.IgnoreMeshviewer.Duration)
	ignoreNode := now.Add(-f.IgnoreNode.Duration)

	wgFetch := sync.WaitGroup{}

	var status []*Status

	// fetch jsons
	f.ConfigMutex.Lock()
	for url, name := range f.DataPaths {
		wgFetch.Add(1)
		go func(url string, name string) {
			defer wgFetch.Done()
			logger := log.WithFields(map[string]interface{}{
				"url":  url,
				"name": name,
			})
			var mv meshviewerFFRGB.Meshviewer

			err := runtime.JSONRequest(url, &mv)
			if err != nil {
				status = append(status, &Status{
					URL:   url,
					Error: err.Error(),
				})
				logger.Errorf("fetch meshviewer.json failed: %s", err)
				return
			}

			allNodes := len(mv.Nodes)
			allLinks := len(mv.Links)

			if mv.Timestamp.Before(ignoreMeshviewer) {
				errStr := fmt.Sprintf("drop meshviewer.json %s is older then %s", mv.Timestamp.GetTime().String(), ignoreMeshviewer.GetTime().String())
				status = append(status, &Status{
					URL:            url,
					Error:          errStr,
					Timestamp:      mv.Timestamp,
					NodesSkipCount: allNodes,
					NodesCount:     allNodes,
					LinksCount:     allLinks,
				})
				logger.Errorf(errStr)
				return
			}

			conv := converter.NewConverter(&mv)

			count := 0
			neighbours := 0
			for _, node := range mv.Nodes {
				if node.Lastseen.Before(ignoreNode) {
					continue
				}
				originNode, n := conv.Node(node, name)
				f.database.InsertNode(originNode)
				neighbours += n
				count++
			}

			logger.WithFields(log.Fields{
				"send_nodes":       count,
				"skip_nodes":       allNodes - count,
				"meshviewer_nodes": allNodes,
				"send_neighbours":  neighbours,
				"meshviewer_links": allLinks,
			}).Info("send to yanic")

			status = append(status, &Status{
				URL:             url,
				Name:            name,
				Timestamp:       mv.Timestamp,
				NodesCount:      count,
				NodesSkipCount:  allNodes - count,
				NeighboursCount: neighbours,
				LinksCount:      allLinks,
			})

		}(url, name)
	}
	f.ConfigMutex.Unlock()

	wgFetch.Wait()
	if f.StatusJSON != "" {
		if err := file.SaveJSON(f.StatusJSON, status); err != nil {
			log.Warnf("complete of fetching with error save status %s", err.Error())
		} else {

			log.Info("complete of fetching")
		}
	} else {
		log.Info("complete of fetching")
	}
}
