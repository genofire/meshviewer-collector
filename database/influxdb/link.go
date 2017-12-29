package influxdb

import (
	"time"

	meshviewer "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	models "github.com/influxdata/influxdb/models"
)

// InsertLink adds a link data point
func (conn *Connection) InsertLink(link *meshviewer.Link, t time.Time) {
	tags := models.Tags{}
	tags.SetString("source.id", link.Source)
	tags.SetString("source.mac", link.SourceMAC)
	tags.SetString("target.id", link.Target)
	tags.SetString("target.mac", link.TargetMAC)

	conn.addPoint(MeasurementLink, tags, models.Fields{"tq": link.SourceTQ * 100}, t)

	tags.SetString("target.id", link.Source)
	tags.SetString("target.mac", link.SourceMAC)
	tags.SetString("source.id", link.Target)
	tags.SetString("source.mac", link.TargetMAC)

	conn.addPoint(MeasurementLink, tags, models.Fields{"tq": link.TargetTQ * 100}, t)
}
