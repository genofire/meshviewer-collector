package fetcher

import "github.com/FreifunkBremen/yanic/lib/jsontime"

type Status struct {
	URL             string        `json:"url"`
	Error           string        `json:"error,omitempty"`
	Timestamp       jsontime.Time `json:"timestemp"`
	NodesCount      int           `json:"nodes_count"`
	NodesSkipCount  int           `json:"nodes_skip_count"`
	NeighboursCount int           `json:"neighbours_count"`
	LinksCount      int           `json:"links_count"`
}
