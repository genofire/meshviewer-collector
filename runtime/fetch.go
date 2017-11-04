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
	temp    *template.Template
	founded []string
	closed  bool
}

func NewFetcher(config *Config) *Fetcher {
	t, err := template.ParseFiles(config.Meshviewer.ConfigTemplate)
	if err != nil {
		log.Error(err)
		return nil
	}
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

	wgDirectory := sync.WaitGroup{}
	count := 0

	for _, dp := range f.Meshviewer.DataPath {
		wgDirectory.Add(1)
		go func(dp *DataPath) {
			defer wgDirectory.Done()
			var meshviewerFile meshviewerFFRGB.Meshviewer

			err := JSONRequest(fmt.Sprintf("%s/meshviewer.json", dp.URL), &meshviewerFile)
			if err != nil {
				log.Errorf("fetch url %s failed: %s", dp.URL, err)
				return
			}

			log.Infof("fetch url %s", dp.URL)
			filename := fmt.Sprintf("%s-meshviewer.json", dp.Filename)
			err = lib.SaveJSON(filepath.Join(f.Folder, filename), meshviewerFile)
			if err != nil {
				log.Errorf("save url %s failed: %s", dp.URL, err)
				return
			}
			f.founded = append(f.founded, fmt.Sprintf(f.Meshviewer.DataPathURL, dp.Filename))

			count++
		}(dp)
	}
	wgDirectory.Wait()
	log.Infof("%d community readed", count)

	siteNames, _ := json.Marshal(f.Meshviewer.SiteNames)
	dataPath, _ := json.Marshal(f.founded)
	file, err := os.OpenFile(f.Meshviewer.ConfigOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Error(err)
	}
	err = f.temp.Execute(file, map[string]interface{}{
		"SiteNames": string(siteNames),
		"DataPath":  string(dataPath),
	})
	if err != nil {
		log.Error(err)
	}
	file.Close()
	f.founded = []string{}
}
