package runtime

import (
	yanicRuntime "github.com/FreifunkBremen/yanic/runtime"
)

type Config struct {
	Folder     string                `toml:"folder"`
	RunEvery   yanicRuntime.Duration `toml:"run_every"`
	Meshviewer struct {
		ConfigTemplate string      `toml:"config_template"`
		ConfigOutput   string      `toml:"config_output"`
		DataPathURL    string      `toml:"data_path_url"`
		DataPath       []*DataPath `toml:"dataPath"`
		SiteNames      []struct {
			Site string `toml:"site" json:"site"`
			Name string `toml:"name" json:"name"`
		} `toml:"siteNames"`
	} `toml:"meshviewer"`
}

type DataPath struct {
	URL      string `toml:"url"`
	Filename string `toml:"filename"`
}
