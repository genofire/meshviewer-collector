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

	Database struct {
		DeleteInterval yanicRuntime.Duration `toml:"delete_interval"` // Delete stats of nodes every n minutes
		DeleteAfter    yanicRuntime.Duration `toml:"delete_after"`    // Delete stats of nodes till now-deletetill n minutes
		Connection     map[string]interface{}
	}
}
