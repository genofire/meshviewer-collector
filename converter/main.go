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
		if link.SourceAddress == "" || link.Source == "" || link.TargetAddress == "" || link.Target == "" {
			continue
		}
		if _, ok := conv.macToLink[link.SourceAddress]; !ok {
			conv.macToLink[link.SourceAddress] = []*meshviewerFFRGB.Link{link}
		} else {
			conv.macToLink[link.SourceAddress] = append(conv.macToLink[link.SourceAddress], link)
		}
		if _, ok := conv.macToLink[link.TargetAddress]; !ok {
			conv.macToLink[link.TargetAddress] = []*meshviewerFFRGB.Link{link}
		} else {
			conv.macToLink[link.TargetAddress] = append(conv.macToLink[link.TargetAddress], link)
		}

		conv.macToTQ[link.SourceAddress+"-"+link.TargetAddress] = link.SourceTQ
		conv.macToTQ[link.TargetAddress+"-"+link.SourceAddress] = link.TargetTQ

		if _, ok := conv.nodeIDToMac[link.Source]; !ok {
			conv.nodeIDToMac[link.Source] = []string{link.SourceAddress}
		} else {
			conv.nodeIDToMac[link.Source] = append(conv.nodeIDToMac[link.Source], link.SourceAddress)
		}

		if _, ok := conv.nodeIDToMac[link.Target]; !ok {
			conv.nodeIDToMac[link.Target] = []string{link.TargetAddress}
		} else {
			conv.nodeIDToMac[link.Target] = append(conv.nodeIDToMac[link.Target], link.TargetAddress)
		}

		if strings.Contains(link.Type, "vpn") {
			conv.macTunnel[link.SourceAddress] = struct{}{}
			conv.macTunnel[link.TargetAddress] = struct{}{}
		}
		if strings.Contains(link.Type, "wifi") {
			conv.macWifi[link.SourceAddress] = struct{}{}
			conv.macWifi[link.TargetAddress] = struct{}{}
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
				Mesh:      make(map[string]*yanicData.NetworkInterface),
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
		},
		Neighbours: &yanicData.Neighbours{
			NodeID: node.NodeID,
			Batadv: make(map[string]yanicData.BatadvNeighbours),
		},
	}

	if node.MemoryUsage != nil {
		newNode.Statistics.Memory = yanicData.Memory{
			Free:  100 - int64(*node.MemoryUsage*100.0),
			Total: 100,
		}
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
			neighMAC := link.SourceAddress
			if neighMAC == mac {
				neighMAC = link.TargetAddress
			}
			ifname.Neighbours[neighMAC] = yanicData.BatmanLink{
				Tq:       int(conv.macToTQ[mac+"-"+neighMAC] * 255.0),
				Lastseen: 1,
			}

		}
		newNode.Neighbours.Batadv[mac] = ifname
		neighbours++
	}
	newNode.Nodeinfo.Network.Mesh["bat0"] = &yanicData.NetworkInterface{
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
