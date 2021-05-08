package reflector

import (
	bnet "github.com/bio-routing/bio-rd/net"
	bio "github.com/bio-routing/bio-rd/protocols/bgp/server"
	"github.com/bio-routing/bio-rd/routingtable/vrf"
)

func (s *Server) EnsurePeer(ip *bnet.IP, name string) error {
	log := s.log.WithField("node", name).WithField("addr", ip)
	log.Debug("updating peer")

	if _, ok := s.staticPeerSet[*ip]; ok {
		log.Info("peer in static peers, not using dynamic config")
		return nil
	}

	if conf := s.bgp.GetPeerConfig(ip); conf != nil {
		if conf.Description == name {
			log.Debug("already peering")
			return nil
		}
		// cycle peer if name changed
		s.bgp.DisposePeer(ip)
	}

	peer := s.buildPeerConfig(ip, name)
	return s.bgp.AddPeer(peer)
}

func (s *Server) RemovePeer(ip *bnet.IP) error {
	s.log.WithField("addr", ip).Debug("removing peer")
	s.bgp.DisposePeer(ip)
	return nil
}

func (s *Server) buildPeerConfig(addr *bnet.IP, name string) bio.PeerConfig {
	primaryVRF := vrf.GetGlobalRegistry().CreateVRFIfNotExists("primary", 0)
	return bio.PeerConfig{
		AuthenticationKey:          "",
		AdminEnabled:               true,
		ReconnectInterval:          DefaultReconnectInterval,
		HoldTime:                   DefaultHoldTime,
		KeepAlive:                  DefaultKeepAlive,
		LocalAddress:               s.conf.localAddressIP,
		PeerAddress:                addr,
		TTL:                        0, // todo: route?
		LocalAS:                    s.conf.LocalAS,
		PeerAS:                     s.conf.LocalAS,
		Passive:                    true,
		RouterID:                   addr.ToUint32(),
		RouteServerClient:          false,
		RouteReflectorClient:       false,
		RouteReflectorClusterID:    0,
		AdvertiseIPv4MultiProtocol: s.conf.IPv4MultiProtocol,
		IPv4: &bio.AddressFamilyConfig{
			ImportFilterChain: allFilterChain,
			ExportFilterChain: noneFilterChain,
		},
		IPv6:        &bio.AddressFamilyConfig{},
		VRF:         primaryVRF,
		Description: name,
	}
}
