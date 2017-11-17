package runtime

import (
	"sync"
	"time"

	"github.com/FreifunkBremen/yanic/jsontime"
	meshviewerFFRGB "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	log "github.com/Sirupsen/logrus"
	lib "github.com/genofire/golang-lib/file"
)

type Fetcher struct {
	*Config
	closed bool
}

func NewFetcher(config *Config) *Fetcher {
	return &Fetcher{
		Config: config,
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

	for _, dp := range f.DataPaths {
		wgFetch.Add(1)
		go func(url string) {
			defer wgFetch.Done()
			var meshviewerFile meshviewerFFRGB.Meshviewer

			err := JSONRequest(url, &meshviewerFile)
			if err != nil {
				log.Errorf("fetch url %s failed: %s", url, err)
				return
			}

			log.Infof("fetch url %s", url)
			reading = append(reading, &meshviewerFile)
			count++
		}(dp)
	}

	wgFetch.Wait()
	log.Infof("%d community fetched", count)

	for _, mv := range reading {
		nodeExists := make(map[string]bool)
		if mv.Timestamp.Before(ignoreMeshviewer) {
			continue
		}
		if mv.Timestamp.Before(output.Timestamp) {
			output.Timestamp = mv.Timestamp
		}
		for _, node := range mv.Nodes {
			if node.Lastseen.After(ignoreNode) {
				nodeExists[node.NodeID] = true
				output.Nodes = append(output.Nodes, node)
			}
		}
		for _, link := range mv.Links {
			if nodeExists[link.Source] && nodeExists[link.Target] {
				output.Links = append(output.Links, link)
			}
		}
	}

	log.Infof("%d nodes readed", len(output.Nodes))

	err := lib.SaveJSON(f.Output, output)
	if err != nil {
		log.Errorf("save output failed: %s", err)
	}
}
