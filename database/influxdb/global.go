package influxdb

import (
	"time"

	runtimeYanic "github.com/FreifunkBremen/yanic/runtime"
	runtime "github.com/genofire/meshviewer-collector/runtime"
	"github.com/influxdata/influxdb/models"
)

// InsertGlobals implementation of database
func (conn *Connection) InsertGlobals(stats *runtime.GlobalStats, time time.Time, site string) {
	var tags models.Tags

	measurementGlobal := MeasurementGlobal
	counterMeasurementModel := CounterMeasurementModel
	counterMeasurementFirmware := CounterMeasurementFirmware
	counterMeasurementAutoupdater := CounterMeasurementAutoupdater

	if site != runtimeYanic.GLOBAL_SITE {
		tags = models.Tags{
			models.Tag{Key: []byte("site"), Value: []byte(site)},
		}

		measurementGlobal += "_site"
		counterMeasurementModel += "_site"
		counterMeasurementFirmware += "_site"
		counterMeasurementAutoupdater += "_site"
	}

	conn.addPoint(measurementGlobal, tags, GlobalStatsFields(stats), time)
	conn.addCounterMap(counterMeasurementModel, stats.Models, time, site)
	conn.addCounterMap(counterMeasurementFirmware, stats.Firmwares, time, site)
}

// GlobalStatsFields returns fields for InfluxDB
func GlobalStatsFields(stats *runtime.GlobalStats) map[string]interface{} {
	return map[string]interface{}{
		"nodes":          stats.Nodes,
		"gateways":       stats.Gateways,
		"clients.total":  stats.Clients,
		"clients.other":  stats.ClientsOther,
		"clients.wifi24": stats.ClientsWifi24,
		"clients.wifi5":  stats.ClientsWifi5,
		"autoupdater":    stats.Autoupdater,
	}
}

// Saves the values of a CounterMap in the database.
// The key are used as 'value' tag.
// The value is used as 'counter' field.
func (conn *Connection) addCounterMap(name string, m runtimeYanic.CounterMap, t time.Time, site string) {
	for key, count := range m {
		conn.addPoint(
			name,
			models.Tags{
				models.Tag{Key: []byte("value"), Value: []byte(key)},
				models.Tag{Key: []byte("site"), Value: []byte(site)},
			},
			models.Fields{"count": count},
			t,
		)
	}
}
