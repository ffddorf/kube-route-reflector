package reflector

import (
	"context"

	bnet "github.com/bio-routing/bio-rd/net"
	bio "github.com/bio-routing/bio-rd/protocols/bgp/server"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type Server struct {
	bgp        bio.BGPServer
	kubeClient *kubernetes.Clientset

	conf          BGPConfig
	staticPeerSet map[bnet.IP]struct{}
}

func NewServer(bgp bio.BGPServer, kubeClient *kubernetes.Clientset, bgpConf BGPConfig) *Server {
	return &Server{
		bgp:           bgp,
		kubeClient:    kubeClient,
		conf:          bgpConf,
		staticPeerSet: make(map[bnet.IP]struct{}),
	}
}

func (s *Server) Start(log logrus.FieldLogger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	log.Info("starting Kubernetes node watcher")

	factory := informers.NewSharedInformerFactory(s.kubeClient, defaultResync)
	informer := factory.Core().V1().Nodes().Informer()

	watch := newWatcher(s, log)
	informer.AddEventHandler(watch)

	informer.Run(ctx.Done())
	return nil
}
