package all

import (
	"log"
	"time"

	meshviewer "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	"github.com/genofire/meshviewer-collector/database"
	"github.com/genofire/meshviewer-collector/runtime"
)

type Connection struct {
	database.Connection
	list []database.Connection
}

func Connect(allConnection map[string]interface{}) (database.Connection, error) {
	var list []database.Connection
	for dbType, conn := range database.Adapters {
		configForType := allConnection[dbType]
		if configForType == nil {
			log.Printf("the output type '%s' has no configuration\n", dbType)
			continue
		}
		dbConfigs, ok := configForType.([]map[string]interface{})
		if !ok {
			log.Panicf("the output type '%s' has the wrong format\n", dbType)
		}

		for _, config := range dbConfigs {
			if c, ok := config["enable"].(bool); ok && !c {
				continue
			}
			connected, err := conn(config)
			if err != nil {
				return nil, err
			}
			if connected == nil {
				continue
			}
			list = append(list, connected)
		}
	}
	return &Connection{list: list}, nil
}

func (conn *Connection) InsertNode(node *meshviewer.Node) {
	for _, item := range conn.list {
		item.InsertNode(node)
	}
}

func (conn *Connection) InsertLink(link *meshviewer.Link, time time.Time) {
	for _, item := range conn.list {
		item.InsertLink(link, time)
	}
}

func (conn *Connection) InsertGlobals(stats *runtime.GlobalStats, time time.Time, site string) {
	for _, item := range conn.list {
		item.InsertGlobals(stats, time, site)
	}
}

func (conn *Connection) PruneNodes(deleteAfter time.Duration) {
	for _, item := range conn.list {
		item.PruneNodes(deleteAfter)
	}
}

func (conn *Connection) Close() {
	for _, item := range conn.list {
		item.Close()
	}
}
