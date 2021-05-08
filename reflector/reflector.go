package reflector

import (
	bnet "github.com/bio-routing/bio-rd/net"
	bio "github.com/bio-routing/bio-rd/protocols/bgp/server"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Server struct {
	bgp           bio.BGPServer
	conf          BGPConfig
	staticPeerSet map[bnet.IP]struct{}
	log           logrus.FieldLogger
}

func NewServer(log logrus.FieldLogger, bgp bio.BGPServer, bgpConf BGPConfig) *Server {
	return &Server{
		bgp:           bgp,
		conf:          bgpConf,
		staticPeerSet: make(map[bnet.IP]struct{}),
		log:           log,
	}
}

func (s *Server) Start(log logrus.FieldLogger) error {
	localAddr, err := bnet.IPFromString(s.conf.LocalAddress)
	if err != nil {
		return errors.Wrap(err, "failed to read local address")
	}
	s.conf.localAddressIP = &localAddr

	peers, err := s.conf.Peers()
	if err != nil {
		return errors.Wrap(err, "failed to read peers")
	}
	for _, peer := range peers {
		if err := s.bgp.AddPeer(peer); err != nil {
			log.WithError(err).WithField("addr", peer.PeerAddress).Warn("failed to add peer, continuing")
		}
		s.staticPeerSet[*peer.PeerAddress] = struct{}{}
	}

	log.Info("starting BGP server")
	if err := s.bgp.Start(); err != nil {
		return errors.Wrap(err, "failed to start BGP server")
	}

	return nil
}
