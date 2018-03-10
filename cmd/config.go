package cmd

import (
	"dev.sum7.eu/genofire/golang-lib/file"

	"github.com/genofire/meshviewer-collector/runtime"
)

var configPath string

func loadConfig() (runtime.Config, error) {
	config := runtime.Config{}
	err := file.ReadTOML(configPath, &config)
	return config, err
}
