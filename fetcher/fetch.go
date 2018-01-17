package fetcher

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FreifunkBremen/yanic/lib/jsontime"
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

	reading := make(map[string]*meshviewerFFRGB.Meshviewer)
	readingMX := sync.Mutex{}

	// fetch jsons
	f.ConfigMutex.Lock()
	for _, dp := range f.DataPaths {
		wgFetch.Add(1)
		go func(url string) {
			defer wgFetch.Done()
			var meshviewerFile meshviewerFFRGB.Meshviewer

			err := runtime.JSONRequest(url, &meshviewerFile)
			if err != nil {
				log.WithField("url", url).Errorf("fetch meshviewer.json failed: %s", err)
				return
			}

			log.WithFields(log.Fields{
				"url":   url,
				"nodes": len(meshviewerFile.Nodes),
				"links": len(meshviewerFile.Links),
			}).Info("fetch")
			readingMX.Lock()
			reading[url] = &meshviewerFile
			readingMX.Unlock()
		}(dp)
	}
	f.ConfigMutex.Unlock()

	wgFetch.Wait()

	nodes := make(map[string]*meshviewerFFRGB.Node)
	links := make(map[string]*meshviewerFFRGB.Link)

	countMerge := 0

	// merge jsons
	for url, mv := range reading {
		nodesExists := make(map[string]bool)
		if mv.Timestamp.Before(ignoreMeshviewer) {
			log.WithField("url", url).Errorf("drop meshviewer.json %s is older then %s", mv.Timestamp.GetTime().String(), ignoreMeshviewer.GetTime().String())
			continue
		}
		if mv.Timestamp.Before(output.Timestamp) {
			output.Timestamp = mv.Timestamp
		}
		countNodes := 0
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
					countNodes++
				}
			}
		}

		countLinks := 0
		for _, link := range mv.Links {
			var key string
			if strings.Compare(link.SourceMAC, link.TargetMAC) > 0 {
				key = fmt.Sprintf("%s-%s", link.SourceMAC, link.TargetMAC)
			} else {
				key = fmt.Sprintf("%s-%s", link.TargetMAC, link.SourceMAC)
			}
			if nodesExists[link.Source] && nodesExists[link.Target] {
				links[key] = link
				countLinks++
			}
		}
		countMerge++
		log.WithFields(log.Fields{
			"url":   url,
			"nodes": countNodes,
			"links": countLinks,
		}).Info("read")
	}

	// export (to database)
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
	log.WithFields(log.Fields{
		"nodes":               len(output.Nodes),
		"links":               len(output.Links),
		"communities":         countMerge,
		"communities_fetched": len(reading),
		"communities_config":  len(f.DataPaths),
	}).Info("export")

	for site, stats := range runtime.NewGlobalStats(output) {
		f.database.InsertGlobals(stats, now.GetTime(), site)
	}

	err := lib.SaveJSON(f.Output, output)
	if err != nil {
		log.Errorf("save output failed: %s", err)
	}
}
