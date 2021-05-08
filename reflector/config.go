package reflector

import (
	"time"

	bnet "github.com/bio-routing/bio-rd/net"
	bgp "github.com/bio-routing/bio-rd/protocols/bgp/server"
	"github.com/bio-routing/bio-rd/routingtable/filter"
	"github.com/bio-routing/bio-rd/routingtable/vrf"
)

type BGPPeer struct {
	RouterID                string `yaml:"router_id"`
	Address                 string `yaml:"peer_address"`
	AS                      uint32 `yaml:"peer_as"`
	AuthKey                 string `yaml:"auth_key"`
	Passive                 bool   `yaml:"passive"`
	RouteReflectorClient    bool   `yaml:"route_reflector_client"`
	RouteReflectorClusterID uint32 `yaml:"route_reflector_cluster_id"`
}

type BGPConfig struct {
	RouterID string `yaml:"router_id"`
	LocalAS  uint32 `yaml:"local_as"`

	LocalAddress   string   `yaml:"local_address"`
	localAddressIP *bnet.IP `yaml:"-"`

	IPv4MultiProtocol bool `yaml:"ipv4_multi_protocol"`

	StaticPeers []BGPPeer `yaml:"static_peers"`
}

// Useful filter chains
var (
	allFilterChain  = filter.NewAcceptAllFilterChain()
	noneFilterChain = filter.NewDrainFilterChain()
)

// Defaults for BGP timers
const (
	DefaultReconnectInterval = time.Second * 15
	DefaultHoldTime          = time.Second * 90
	DefaultKeepAlive         = time.Second * 30
)

func (c *BGPConfig) Peers() ([]bgp.PeerConfig, error) {
	primaryVRF := vrf.GetGlobalRegistry().CreateVRFIfNotExists("primary", 0)

	list := make([]bgp.PeerConfig, 0, len(c.StaticPeers))
	for _, p := range c.StaticPeers {
		routerID, err := bnet.IPFromString(p.RouterID)
		if err != nil {
			return nil, err
		}
		addr, err := bnet.IPFromString(p.Address)
		if err != nil {
			return nil, err
		}
		list = append(list, bgp.PeerConfig{
			AuthenticationKey:          p.AuthKey,
			AdminEnabled:               true,
			ReconnectInterval:          DefaultReconnectInterval,
			HoldTime:                   DefaultHoldTime,
			KeepAlive:                  DefaultKeepAlive,
			LocalAddress:               c.localAddressIP,
			PeerAddress:                &addr,
			TTL:                        0, // todo: allow routing?
			LocalAS:                    c.LocalAS,
			PeerAS:                     p.AS,
			Passive:                    p.Passive,
			RouterID:                   routerID.ToUint32(),
			RouteServerClient:          false,
			RouteReflectorClient:       p.RouteReflectorClient,
			RouteReflectorClusterID:    p.RouteReflectorClusterID,
			AdvertiseIPv4MultiProtocol: true,
			IPv4: &bgp.AddressFamilyConfig{
				ImportFilterChain: allFilterChain,
				ExportFilterChain: allFilterChain,
			},
			IPv6:        &bgp.AddressFamilyConfig{},
			VRF:         primaryVRF,
			Description: "",
		})
	}
	return list, nil
}
