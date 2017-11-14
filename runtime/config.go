package runtime

import (
	yanicRuntime "github.com/FreifunkBremen/yanic/runtime"
)

type Config struct {
	RunEvery   yanicRuntime.Duration `toml:"run_every"`
	Meshviewer struct {
		Output     string    `toml:"output"`
		DataPaths  []string  `toml:"dataPaths"`
	} `toml:"meshviewer"`
}
