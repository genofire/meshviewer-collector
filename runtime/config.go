package runtime

import (
	yanicRuntime "github.com/FreifunkBremen/yanic/runtime"
)

type Config struct {
	RunEvery         yanicRuntime.Duration `toml:"run_every"`
	Output           string                `toml:"output"`
	IgnoreMeshviewer yanicRuntime.Duration `toml:"ignore_meshviewer"`
	IgnoreNode       yanicRuntime.Duration `toml:"ignore_node"`
	DataPaths        []string              `toml:"dataPaths"`
}
