package reflector

import (
	bnet "github.com/bio-routing/bio-rd/net"
	bgp "github.com/bio-routing/bio-rd/protocols/bgp/server"
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

func (c *BGPConfig) Peers() ([]bgp.PeerConfig, error) {
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
			LocalAS:                    c.LocalAS,
			PeerAS:                     p.AS,
			AuthenticationKey:          p.AuthKey,
			LocalAddress:               c.localAddressIP,
			PeerAddress:                &addr,
			Passive:                    p.Passive,
			RouterID:                   routerID.ToUint32(),
			RouteReflectorClient:       p.RouteReflectorClient,
			RouteReflectorClusterID:    p.RouteReflectorClusterID,
			AdvertiseIPv4MultiProtocol: true,
		})
	}
	return list, nil
}
