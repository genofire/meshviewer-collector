package runtime

import (
	"github.com/FreifunkBremen/yanic/lib/duration"
)

type Config struct {
	RunEvery         duration.Duration `toml:"run_every"`
	Output           string            `toml:"output"`
	IgnoreMeshviewer duration.Duration `toml:"ignore_meshviewer"`
	IgnoreNode       duration.Duration `toml:"ignore_node"`
	DataPaths        []string          `toml:"dataPaths"`

	Database struct {
		DeleteInterval duration.Duration `toml:"delete_interval"` // Delete stats of nodes every n minutes
		DeleteAfter    duration.Duration `toml:"delete_after"`    // Delete stats of nodes till now-deletetill n minutes
		Connection     map[string]interface{}
	}
}
