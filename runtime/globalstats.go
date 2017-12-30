package runtime

import (
	meshviewer "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	runtimeYanic "github.com/FreifunkBremen/yanic/runtime"
)

// GlobalStats struct
type GlobalStats struct {
	Clients       uint32
	ClientsWifi24 uint32
	ClientsWifi5  uint32
	ClientsOther  uint32
	Gateways      uint32
	Nodes         uint32
	Autoupdater   uint32

	Firmwares runtimeYanic.CounterMap
	Models    runtimeYanic.CounterMap
}

func NewGlobalStats(meshviewer *meshviewer.Meshviewer) (result map[string]*GlobalStats) {
	result = make(map[string]*GlobalStats)

	result[runtimeYanic.GLOBAL_SITE] = &GlobalStats{
		Firmwares: make(runtimeYanic.CounterMap),
		Models:    make(runtimeYanic.CounterMap),
	}

	for _, node := range meshviewer.Nodes {
		if node.IsOnline {
			result[runtimeYanic.GLOBAL_SITE].Add(node)
			if node.SiteCode != "" {

				if _, ok := result[node.SiteCode]; !ok {
					result[node.SiteCode] = &GlobalStats{
						Firmwares: make(runtimeYanic.CounterMap),
						Models:    make(runtimeYanic.CounterMap),
					}
				}
				result[node.SiteCode].Add(node)
			}
		}
	}

	return result
}

func (s *GlobalStats) Add(node *meshviewer.Node) {
	s.Nodes++
	s.Clients += node.Clients
	s.ClientsWifi24 += node.ClientsWifi24
	s.ClientsWifi5 += node.ClientsWifi5
	s.ClientsOther += node.ClientsOthers
	if node.IsGateway {
		s.Gateways++
	}
	s.Models.Increment(node.Model)
	s.Firmwares.Increment(node.Firmware.Base)
	if node.Autoupdater.Enabled {
		s.Autoupdater++
	}
}
