package influxdb

import (
	"fmt"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
	models "github.com/influxdata/influxdb/models"

	meshviewer "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	yanicRuntime "github.com/FreifunkBremen/yanic/runtime"
)

// PruneNodes prunes historical per-node data
func (conn *Connection) PruneNodes(deleteAfter time.Duration) {
	for _, measurement := range []string{MeasurementNode, MeasurementLink} {
		query := fmt.Sprintf("delete from %s where time < now() - %ds", measurement, deleteAfter/time.Second)
		conn.client.Query(client.NewQuery(query, conn.config.Database(), "m"))
	}

}

// InsertNode stores statistics and neighbours in the database
func (conn *Connection) InsertNode(node *meshviewer.Node) {
	time := node.Lastseen.GetTime()

	tags := models.Tags{}
	tags.SetString("nodeid", node.NodeID)
	tags.SetString("hostname", node.Hostname)
	tags.SetString("site", node.SiteCode)

	fields := models.Fields{
		"load":           node.LoadAverage,
		"time.up":        node.Uptime,
		"clients.total":  node.Clients,
		"clients.wifi24": node.ClientsWifi24,
		"clients.wifi5":  node.ClientsWifi5,
		"clients.other":  node.ClientsOthers,
	}
	if node.MemoryUsage != nil {
		fields["memory.buffers"] = node.MemoryUsage
	}

	// Hardware
	tags.SetString("model", node.Model)
	tags.SetString("firmware_base", node.Firmware.Base)
	if node.Autoupdater.Enabled {
		tags.SetString("autoupdater", node.Autoupdater.Branch)
	} else {
		tags.SetString("autoupdater", yanicRuntime.DISABLED_AUTOUPDATER)
	}

	conn.addPoint(MeasurementNode, tags, fields, time)

	return
}
