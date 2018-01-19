package runtime

import (
	"github.com/FreifunkBremen/yanic/lib/duration"
)

type Config struct {
	RunEvery duration.Duration `toml:"run_every"`

	IgnoreMeshviewer duration.Duration `toml:"ignore_meshviewer"`
	IgnoreNode       duration.Duration `toml:"ignore_node"`
	DataPaths        []string          `toml:"dataPaths"`

	YanicConnection map[string]interface{} `toml:"yanic_connection"`
}
