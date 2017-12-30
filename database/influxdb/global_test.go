package influxdb

import (
	"sync"
	"testing"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/stretchr/testify/assert"

	"github.com/FreifunkBremen/yanic/jsontime"
	meshviewer "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	runtimeYanic "github.com/FreifunkBremen/yanic/runtime"
	"github.com/genofire/meshviewer-collector/runtime"
)

const TEST_SITE = "ffxx"

func TestGlobalStats(t *testing.T) {
	stats := runtime.NewGlobalStats(createTestNodes())

	assert := assert.New(t)

	// check SITE_GLOBAL fields
	fields := GlobalStatsFields(stats[runtimeYanic.GLOBAL_SITE])
	assert.EqualValues(3, fields["nodes"])
	assert.EqualValues(1, fields["autoupdater"])

	fields = GlobalStatsFields(stats[TEST_SITE])
	assert.EqualValues(1, fields["nodes"])
	assert.EqualValues(0, fields["autoupdater"])

	conn := &Connection{
		points: make(chan *client.Point),
	}

	global := 0
	globalSite := 0
	model := 0
	modelSite := 0
	firmware := 0
	firmwareSite := 0
	wg := sync.WaitGroup{}
	wg.Add(6)
	go func() {
		for p := range conn.points {
			switch p.Name() {
			case MeasurementGlobal:
				global++
				break
			case "global_site":
				globalSite++
				break
			case CounterMeasurementModel:
				model++
				break
			case "model_site":
				modelSite++
				break
			case CounterMeasurementFirmware:
				firmware++
				break
			case "firmware_site":
				firmwareSite++
				break
			default:
				assert.Equal("invalid p.Name found", p.Name())
			}
			wg.Done()
		}
	}()
	for site, stat := range stats {
		conn.InsertGlobals(stat, time.Now(), site)
	}
	wg.Wait()
	assert.Equal(1, global)
	assert.Equal(1, globalSite)
	assert.Equal(2, model)
	assert.Equal(1, modelSite)
	assert.Equal(1, firmware)
	assert.Equal(0, firmwareSite)
}

func createTestNodes() *meshviewer.Meshviewer {
	return &meshviewer.Meshviewer{
		Timestamp: jsontime.Now(),
		Nodes: []*meshviewer.Node{
			&meshviewer.Node{
				IsOnline:    true,
				Hostname:    "blub2",
				Firmware:    meshviewer.Firmware{Base: "2016.1.4"},
				Autoupdater: meshviewer.Autoupdater{Enabled: true},
				Clients:     23,
				Model:       "TP-Link 841",
			},
			&meshviewer.Node{
				IsOnline:    true,
				Hostname:    "blub2",
				Firmware:    meshviewer.Firmware{Base: "2016.1.4"},
				Autoupdater: meshviewer.Autoupdater{Enabled: false},
				Model:       "TP-Link 841",
				Clients:     2,
			},
			&meshviewer.Node{
				IsOnline: true,
				Hostname: "blub3",
				VPN:      true,
				Model:    "Xeon Multi-Core",
				SiteCode: TEST_SITE,
			},
		},
	}
}
