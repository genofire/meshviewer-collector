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

	wgRead := sync.WaitGroup{}
	reading := make(chan *meshviewerFFRGB.Meshviewer, len(f.DataPaths))

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
			reading <- &meshviewerFile
			wgRead.Add(1)
			count++
		}(dp)
	}
	go func() {
		for mv := range reading {
			nodeExists := make(map[string]bool)
			if mv.Timestamp.Before(ignoreMeshviewer) {
				wgRead.Done()
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
			wgRead.Done()
		}
	}()

	wgFetch.Wait()
	log.Infof("%d community fetched", count)

	wgRead.Wait()
	log.Infof("%d nodes readed", len(output.Nodes))
	close(reading)

	err := lib.SaveJSON(f.Output, output)
	if err != nil {
		log.Errorf("save output failed: %s", err)
	}
}
