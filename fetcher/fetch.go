package fetcher

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FreifunkBremen/yanic/jsontime"
	meshviewerFFRGB "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	log "github.com/Sirupsen/logrus"
	lib "github.com/genofire/golang-lib/file"
	"github.com/genofire/meshviewer-collector/database"
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
	output := &meshviewerFFRGB.Meshviewer{
		Timestamp: now,
	}

	wgFetch := sync.WaitGroup{}
	count := 0

	var reading []*meshviewerFFRGB.Meshviewer

	f.ConfigMutex.Lock()
	for _, dp := range f.DataPaths {
		wgFetch.Add(1)
		go func(url string) {
			defer wgFetch.Done()
			var meshviewerFile meshviewerFFRGB.Meshviewer

			err := runtime.JSONRequest(url, &meshviewerFile)
			if err != nil {
				log.Errorf("fetch url %s failed: %s", url, err)
				return
			}

			log.Infof("fetch url %s", url)
			reading = append(reading, &meshviewerFile)
			count++
		}(dp)
	}
	f.ConfigMutex.Unlock()

	wgFetch.Wait()
	log.Infof("%d community fetched", count)

	nodes := make(map[string]*meshviewerFFRGB.Node)
	links := make(map[string]*meshviewerFFRGB.Link)

	for _, mv := range reading {
		nodesExists := make(map[string]bool)
		if mv.Timestamp.Before(ignoreMeshviewer) {
			continue
		}
		if mv.Timestamp.Before(output.Timestamp) {
			output.Timestamp = mv.Timestamp
		}
		for _, node := range mv.Nodes {
			if node.Lastseen.After(ignoreNode) {

				if oldnode, ok := nodes[node.NodeID]; ok {
					if oldnode.Lastseen.Before(node.Lastseen) {
						nodes[node.NodeID] = node
						nodesExists[node.NodeID] = true
					}
				} else {
					nodes[node.NodeID] = node
					nodesExists[node.NodeID] = true
				}
			}
		}
		for _, link := range mv.Links {
			var key string
			if strings.Compare(link.SourceMAC, link.TargetMAC) > 0 {
				key = fmt.Sprintf("%s-%s", link.SourceMAC, link.TargetMAC)
			} else {
				key = fmt.Sprintf("%s-%s", link.TargetMAC, link.SourceMAC)
			}
			if nodesExists[link.Source] && nodesExists[link.Target] {
				links[key] = link
			}
		}
	}

	nodeUpdate := make(map[string]bool)
	for _, node := range nodes {
		output.Nodes = append(output.Nodes, node)
		if date, ok := f.currentNode[node.NodeID]; !ok || date.Before(node.Lastseen.GetTime()) {
			nodeUpdate[node.NodeID] = true
			f.currentNode[node.NodeID] = node.Lastseen.GetTime()
			f.database.InsertNode(node)
		}
	}
	for _, link := range links {
		output.Links = append(output.Links, link)
		if nodeUpdate[link.Source] || nodeUpdate[link.Target] {
			if f.currentNode[link.Source].Before(f.currentNode[link.Target]) {
				f.database.InsertLink(link, f.currentNode[link.Source])
			} else {
				f.database.InsertLink(link, f.currentNode[link.Target])
			}
		}
	}

	for site, stats := range runtime.NewGlobalStats(output) {
		f.database.InsertGlobals(stats, now.GetTime(), site)
	}

	log.Infof("%d nodes readed", len(output.Nodes))

	err := lib.SaveJSON(f.Output, output)
	if err != nil {
		log.Errorf("save output failed: %s", err)
	}
}
