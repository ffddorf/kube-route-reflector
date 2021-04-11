package reflector

import (
	"time"

	bnet "github.com/bio-routing/bio-rd/net"
	bio "github.com/bio-routing/bio-rd/protocols/bgp/server"
	"github.com/sirupsen/logrus"
	k8s "k8s.io/api/core/v1"
)

const defaultResync = time.Hour

type watcher struct {
	server *Server
	log    logrus.FieldLogger
	errCh  chan error
}

func newWatcher(server *Server, log logrus.FieldLogger) *watcher {
	return &watcher{
		server: server,
		log:    log,
	}
}

func (w *watcher) getPeerAddress(addr k8s.NodeAddress) *bnet.IP {
	if addr.Type != k8s.NodeExternalIP && addr.Type != k8s.NodeInternalIP {
		return nil
	}

	ip, err := bnet.IPFromString(addr.Address)
	if err != nil {
		w.log.WithError(err).Warn("invalid ip in peer")
		return nil
	}

	if _, ok := w.server.staticPeerSet[ip]; ok {
		w.log.Info("skipping - part of the static peers")
		return nil
	}

	return &ip
}

func (w *watcher) OnAdd(obj interface{}) {
	node := obj.(*k8s.Node)
	for _, addr := range node.Status.Addresses {
		ip := w.getPeerAddress(addr)
		if ip == nil {
			continue
		}
		err := w.server.addPeer(w.log, node, ip)
		if err != nil {
			w.log.WithError(err).Warn("failed to add node")
		}
	}
}

func (w *watcher) OnUpdate(oldObj, newObj interface{}) {
	oldNode := oldObj.(*k8s.Node)
	newNode := newObj.(*k8s.Node)

	oldAddrs := make(map[bnet.IP]struct{})

	// fill list of old addresses
	for _, addr := range oldNode.Status.Addresses {
		ip := w.getPeerAddress(addr)
		if ip == nil {
			continue
		}
		oldAddrs[*ip] = struct{}{}
	}

	// compare with new addresses
	for _, addr := range newNode.Status.Addresses {
		ip := w.getPeerAddress(addr)
		if ip == nil {
			continue
		}

		if _, exists := oldAddrs[*ip]; exists {
			// peer didn't change, remove from tracking list
			delete(oldAddrs, *ip)
		} else {
			// didn't peer yet, add peer
			w.server.addPeer(w.log, newNode, ip)
		}
	}

	// remove peers that are only in the old list
	for ip := range oldAddrs {
		log := w.log.WithField("node", oldNode.Name)
		w.server.removePeer(log, &ip)
	}
}

func (w *watcher) OnDelete(obj interface{}) {}

func (s *Server) addPeer(log logrus.FieldLogger, node *k8s.Node, ip *bnet.IP) error {
	log = log.WithField("node", node.Name).WithField("addr", ip)
	log.Debug("adding peer")

	if conf := s.bgp.GetPeerConfig(ip); conf != nil {
		log.Debug("already peering")
		return nil
	}

	peer := s.buildPeerConfig(node, ip)
	return s.bgp.AddPeer(peer)
}

func (s *Server) removePeer(log logrus.FieldLogger, ip *bnet.IP) {
	log.WithField("addr", ip).Debug("removing peer")
	s.bgp.DisposePeer(ip)
}

func (s *Server) buildPeerConfig(node *k8s.Node, addr *bnet.IP) bio.PeerConfig {
	return bio.PeerConfig{
		LocalAddress: s.conf.localAddressIP,
		PeerAddress:  addr,
		LocalAS:      s.conf.LocalAS,
		PeerAS:       s.conf.LocalAS, // IBGP only
		Passive:      true,           // allow nodes to be offline
		Description:  node.Name,
		RouterID:     addr.ToUint32(),

		RouteReflectorClient:       true,
		RouteReflectorClusterID:    0,
		AdvertiseIPv4MultiProtocol: s.conf.IPv4MultiProtocol,

		// todo: configure filters
		IPv4: &bio.AddressFamilyConfig{},
		IPv6: &bio.AddressFamilyConfig{},
	}
}
