package database

import (
	"time"

	meshviewer "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	"github.com/genofire/meshviewer-collector/runtime"
)

// Connection interface to use for implementation in e.g. influxdb
type Connection interface {
	// InsertNode stores statistics per node
	InsertNode(node *meshviewer.Node)

	// InsertLink stores statistics per link
	InsertLink(*meshviewer.Link, time.Time)

	// InsertGlobals stores global statistics
	InsertGlobals(*runtime.GlobalStats, time.Time, string)

	// PruneNodes prunes historical per-node data
	PruneNodes(deleteAfter time.Duration)

	// Close closes the database connection
	Close()
}

// Connect function with config to get DB connection interface
type Connect func(config map[string]interface{}) (Connection, error)

// Adapters is the list of registered database adapters
var Adapters = map[string]Connect{}

func RegisterAdapter(name string, n Connect) {
	Adapters[name] = n
}
