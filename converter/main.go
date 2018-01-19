package converter

import (
	"strings"
	"time"

	yanicData "github.com/FreifunkBremen/yanic/data"
	meshviewerFFRGB "github.com/FreifunkBremen/yanic/output/meshviewer-ffrgb"
	yanicRuntime "github.com/FreifunkBremen/yanic/runtime"
)

type Converter struct {
	Meshviewer  *meshviewerFFRGB.Meshviewer
	nodeIDToMac map[string][]string
	macToLink   map[string][]*meshviewerFFRGB.Link
	macToTQ     map[string]float32
	macTunnel   map[string]struct{}
	macWifi     map[string]struct{}
}

func NewConverter(meshviewer *meshviewerFFRGB.Meshviewer) (conv *Converter) {
	conv = &Converter{
		Meshviewer:  meshviewer,
		nodeIDToMac: make(map[string][]string),
		macToLink:   make(map[string][]*meshviewerFFRGB.Link),
		macToTQ:     make(map[string]float32),
		macTunnel:   make(map[string]struct{}),
		macWifi:     make(map[string]struct{}),
	}

	for _, link := range meshviewer.Links {
		if link.SourceMAC == "" || link.Source == "" || link.TargetMAC == "" || link.Target == "" {
			continue
		}
		if _, ok := conv.macToLink[link.SourceMAC]; !ok {
			conv.macToLink[link.SourceMAC] = []*meshviewerFFRGB.Link{link}
		} else {
			conv.macToLink[link.SourceMAC] = append(conv.macToLink[link.SourceMAC], link)
		}
		if _, ok := conv.macToLink[link.TargetMAC]; !ok {
			conv.macToLink[link.TargetMAC] = []*meshviewerFFRGB.Link{link}
		} else {
			conv.macToLink[link.TargetMAC] = append(conv.macToLink[link.TargetMAC], link)
		}

		conv.macToTQ[link.SourceMAC+"-"+link.TargetMAC] = link.SourceTQ
		conv.macToTQ[link.TargetMAC+"-"+link.SourceMAC] = link.TargetTQ

		if _, ok := conv.nodeIDToMac[link.Source]; !ok {
			conv.nodeIDToMac[link.Source] = []string{link.SourceMAC}
		} else {
			conv.nodeIDToMac[link.Source] = append(conv.nodeIDToMac[link.Source], link.SourceMAC)
		}

		if _, ok := conv.nodeIDToMac[link.Target]; !ok {
			conv.nodeIDToMac[link.Target] = []string{link.TargetMAC}
		} else {
			conv.nodeIDToMac[link.Target] = append(conv.nodeIDToMac[link.Target], link.TargetMAC)
		}

		if strings.Contains(link.Type, "vpn") {
			conv.macTunnel[link.SourceMAC] = struct{}{}
			conv.macTunnel[link.TargetMAC] = struct{}{}
		}
		if strings.Contains(link.Type, "wifi") {
			conv.macWifi[link.SourceMAC] = struct{}{}
			conv.macWifi[link.TargetMAC] = struct{}{}
		}
	}

	return
}

func (conv *Converter) Node(node *meshviewerFFRGB.Node) (*yanicRuntime.Node, int) {
	now := time.Now()
	newNode := &yanicRuntime.Node{
		Nodeinfo: &yanicData.NodeInfo{
			NodeID:   node.NodeID,
			Hostname: node.Hostname,
			Hardware: yanicData.Hardware{
				Model: node.Model,
				Nproc: node.Nproc,
			},
			Network: yanicData.Network{
				Mac:       node.MAC,
				Addresses: node.Addresses,
				Mesh:      make(map[string]*yanicData.BatInterface),
			},
			VPN: node.VPN,
			System: yanicData.System{
				SiteCode: node.SiteCode,
			},
			Software: yanicData.Software{
				Firmware: node.Firmware,
				Autoupdater: struct {
					Enabled bool   `json:"enabled,omitempty"`
					Branch  string `json:"branch,omitempty"`
				}{
					Enabled: node.Autoupdater.Enabled,
					Branch:  node.Autoupdater.Branch,
				},
			},
		},
		Statistics: &yanicData.Statistics{
			NodeID: node.NodeID,
			Uptime: float64(node.Uptime.Unix() - now.Unix()),
			Clients: yanicData.Clients{
				Total:  node.Clients,
				Wifi:   node.Clients - node.ClientsOthers,
				Wifi24: node.ClientsWifi24,
				Wifi5:  node.ClientsWifi5,
			},
			RootFsUsage:    node.RootFSUsage,
			LoadAverage:    node.LoadAverage,
			GatewayNexthop: node.GatewayNexthop,
			GatewayIPv4:    node.GatewayIPv4,
			GatewayIPv6:    node.GatewayIPv6,
			Memory: yanicData.Memory{
				Free:  100 - uint32(*node.MemoryUsage*100.0),
				Total: 100,
			},
		},
		Neighbours: &yanicData.Neighbours{
			NodeID: node.NodeID,
			Batadv: make(map[string]yanicData.BatadvNeighbours),
		},
	}

	if node.Location != nil {
		newNode.Nodeinfo.Location = &yanicData.Location{
			Latitude:  node.Location.Latitude,
			Longitude: node.Location.Longitude,
		}
	}

	if node.Owner != "" {
		newNode.Nodeinfo.Owner = &yanicData.Owner{
			Contact: node.Owner,
		}
	}

	neighbours := 0
	ifaceWireless := []string{}
	ifaceOther := []string{}
	ifaceTunnel := []string{}
	for _, mac := range conv.nodeIDToMac[node.NodeID] {
		if _, ok := conv.macTunnel[mac]; ok {
			ifaceTunnel = append(ifaceTunnel, mac)
		} else if _, ok := conv.macWifi[mac]; ok {
			ifaceWireless = append(ifaceWireless, mac)
		} else {
			ifaceOther = append(ifaceOther, mac)
		}

		ifname := yanicData.BatadvNeighbours{
			Neighbours: make(map[string]yanicData.BatmanLink),
		}
		for _, link := range conv.macToLink[mac] {
			neighMAC := link.SourceMAC
			if neighMAC == mac {
				neighMAC = link.TargetMAC
			}
			ifname.Neighbours[neighMAC] = yanicData.BatmanLink{
				Tq:       int(conv.macToTQ[mac+"-"+neighMAC] * 255.0),
				Lastseen: 1,
			}

		}
		newNode.Neighbours.Batadv[mac] = ifname
		neighbours++
	}
	newNode.Nodeinfo.Network.Mesh["bat0"] = &yanicData.BatInterface{
		Interfaces: struct {
			Wireless []string `json:"wireless,omitempty"`
			Other    []string `json:"other,omitempty"`
			Tunnel   []string `json:"tunnel,omitempty"`
		}{
			Wireless: ifaceWireless,
			Other:    ifaceOther,
			Tunnel:   ifaceTunnel,
		},
	}

	return newNode, neighbours
}
