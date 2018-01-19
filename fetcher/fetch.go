package fetcher

import (
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

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

	// fetch jsons
	f.ConfigMutex.Lock()
	for _, dp := range f.DataPaths {
		wgFetch.Add(1)
		go func(url string) {
			defer wgFetch.Done()
			logger := log.WithField("url", url)
			var mv meshviewerFFRGB.Meshviewer

			err := runtime.JSONRequest(url, &mv)
			if err != nil {
				log.WithField("url", url).Errorf("fetch meshviewer.json failed: %s", err)
				return
			}

			if mv.Timestamp.Before(ignoreMeshviewer) {
				logger.Errorf("drop meshviewer.json %s is older then %s", mv.Timestamp.GetTime().String(), ignoreMeshviewer.GetTime().String())
				return
			}

			conv := converter.NewConverter(&mv)

			count := 0
			neighbours := 0
			for _, node := range mv.Nodes {
				if node.Lastseen.Before(ignoreNode) {
					continue
				}
				originNode, n := conv.Node(node)
				f.database.InsertNode(originNode)
				neighbours += n
				count++
			}

			all := len(mv.Nodes)
			logger.WithFields(log.Fields{
				"send_nodes":       count,
				"skip_nodes":       all - count,
				"meshviewer_nodes": all,
				"send_neighbours":  neighbours,
				"meshviewer_links": len(mv.Links),
			}).Info("send to yanic")

		}(dp)
	}
	f.ConfigMutex.Unlock()

	wgFetch.Wait()
	log.Info("compelete of fetching")
}
