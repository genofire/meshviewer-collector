package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	lib "github.com/genofire/golang-lib/file"
	"github.com/genofire/meshviewer-collector/runtime"
)

var filePath string

func main() {
	flag.StringVar(&filePath, "config", "config.toml", "place to read")
	flag.Parse()
	var config runtime.Config
	err := lib.ReadTOML(filePath, &config)
	if err != nil {
		log.Panic(err)
	}
	f := runtime.NewFetcher(&config)
	// Wait for INT/TERM
	go f.Start()
	log.Info("meshviewer-collector started")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1)
	for sig := range sigs {
		switch sig {
		case syscall.SIGTERM:
			log.Panic("terminated")
			os.Exit(0)

		case syscall.SIGQUIT:
			log.Info("stopped")
			f.Stop()

		case syscall.SIGHUP:
			log.Info("stopped")
			f.Stop()

		case syscall.SIGUSR1:
			log.Info("reloading")
			err := lib.ReadTOML(filePath, &config)
			if err != nil {
				log.Error(err)
			} else {
				f.DataPaths = config.DataPaths
				log.Info("reloaded")
			}
		}
	}
}
