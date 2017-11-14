package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	meshviewerFFRGB "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	log "github.com/Sirupsen/logrus"
	lib "github.com/genofire/golang-lib/file"
)

type Fetcher struct {
	*Config
	closed  bool
}

func NewFetcher(config *Config) *Fetcher {
	return &Fetcher{
		temp:   t,
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
	pruneMeshviewer := now.Add(-time.Hour * 6)
	// do not merge a node older then 3d
	pruneNode := now.Add(-time.Hour * 24 * 3)
	output := &meshviewerFFRGB.Meshviewer{}
	
	wgFetch := sync.WaitGroup{}
	count := 0
	
	wgRead := sync.WaitGroup{}
	reading := make(chan *meshviewerFFRGB.Meshviewer, len(f.Meshviewer.DataPaths))

	for _, dp := range f.Meshviewer.DataPaths {
		wgFetch.Add(1)
		go func(dp string) {
			defer wgFetch.Done()
			var meshviewerFile meshviewerFFRGB.Meshviewer

			err := JSONRequest(fmt.Sprintf("%s/meshviewer.json", dp), &meshviewerFile)
			if err != nil {
				log.Errorf("fetch url %s failed: %s", dp, err)
				return
			}

			log.Infof("fetch url %s", dp)
			reading <- &meshviewerFile
			wgRead.Add(1)
			count++
		}(dp)
	}
	go func() {
		for mv := range reading {
			if mv.Timestamp.Before(pruneMeshviewer) {
				wgRead.Done()
				continue
			}
			if mv.Timestamp.Before(output.Timestamp) {
				output.Timestamp  = mv.Timestamp
			}
			for _, node := range mv.Nodes {
				if node.Lastseen.After(pruneNode) {
					output.Nodes = append(output.Nodes, node)
				}
			}
			for _, links := range mv.Links {
				output.Links = append(output.Links, links)
			}
			wgRead.Done()
		}
	} ()
	
	wgFetch.Wait()
	log.Infof("%d community fetched", count)
	
	wgRead.Wait()
	log.Infof("%d nodes readed", len(output.List)
	close(reading)
		
	err = lib.SaveJSON(f.Meshviewer.Output, output)
	if err != nil {
		log.Errorf("save output failed: %s", err)
		return
	}
}
