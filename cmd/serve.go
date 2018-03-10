package cmd

import (
	"os"
	"os/signal"
	"syscall"

	databaseYanic "github.com/FreifunkBremen/yanic/database/respondd"
	log "github.com/Sirupsen/logrus"
	"github.com/genofire/meshviewer-collector/fetcher"
	"github.com/spf13/cobra"
)

var collectCMD = &cobra.Command{
	Use:   "collect",
	Short: "start the collecting as deamon",

	Run: func(cmd *cobra.Command, args []string) {

		config, err := loadConfig()
		if err != nil {
			os.Exit(3)
		}
		connection, err := databaseYanic.Connect(config.YanicConnection)
		if err != nil {
			panic(err)
		}
		defer connection.Close()

		f := fetcher.NewFetcher(&config, connection)
		// Wait for INT/TERM
		go f.Start()
		log.Info("collecting started")

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1)
		for sig := range sigs {
			switch sig {
			case syscall.SIGTERM:
				log.Panic("terminated")
				f.Stop()
				os.Exit(0)

			case syscall.SIGQUIT:
				log.Info("stopped")
				f.Stop()

			case syscall.SIGHUP:
				log.Info("stopped")
				f.Stop()

			case syscall.SIGUSR1:
				log.Info("reloading")
				config, err = loadConfig()
				if err != nil {
					log.Errorf("error loading config on reload: %s", err)
				} else if len(config.DataPaths) > 0 {
					f.ConfigMutex.Lock()
					f.DataPaths = config.DataPaths
					f.ConfigMutex.Unlock()
					log.Info("reloaded")
				} else {
					log.Warn("try to reload with empty config")
				}
			}
		}
	},
}

func init() {
	collectCMD.Flags().StringVarP(&configPath, "config", "c", "config.toml", "Path to configuration file")
	RootCMD.AddCommand(collectCMD)
}
