package watcher

import (
	"fmt"

	bnet "github.com/bio-routing/bio-rd/net"
	"github.com/sirupsen/logrus"
	k8s "k8s.io/api/core/v1"
)

type BGPServer interface {
	EnsurePeer(addr *bnet.IP, name string) error
	RemovePeer(addr *bnet.IP) error
}

type watcher struct {
	server BGPServer
	log    logrus.FieldLogger

	namePrefix string
}

func newWatcher(server BGPServer, log logrus.FieldLogger, namePrefix string) *watcher {
	return &watcher{
		server:     server,
		log:        log,
		namePrefix: namePrefix,
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

	return &ip
}

func (w *watcher) name(node *k8s.Node) string {
	return fmt.Sprintf("%s_%s", w.namePrefix, node.Name)
}

func (w *watcher) OnAdd(obj interface{}) {
	node, ok := obj.(*k8s.Node)
	if !ok {
		w.log.Warnf("invalid object received: %T", obj)
		return
	}

	for _, addr := range node.Status.Addresses {
		ip := w.getPeerAddress(addr)
		if ip == nil {
			continue
		}
		err := w.server.EnsurePeer(ip, w.name(node))
		if err != nil {
			w.log.WithError(err).Warn("failed to add node")
		}
	}
}

func (w *watcher) OnUpdate(oldObj, newObj interface{}) {
	oldNode, oldOk := oldObj.(*k8s.Node)
	if !oldOk {
		w.log.Warnf("invalid object received: %T", oldObj)
		return
	}
	newNode, newOk := newObj.(*k8s.Node)
	if !newOk {
		w.log.Warnf("invalid object received: %T", newObj)
		return
	}

	newAddrs := make(map[bnet.IP]struct{})
	// update all new addresses and fill list
	for _, addr := range newNode.Status.Addresses {
		ip := w.getPeerAddress(addr)
		if ip == nil {
			continue
		}
		// track address
		newAddrs[*ip] = struct{}{}
		// update peering
		if err := w.server.EnsurePeer(ip, w.name(newNode)); err != nil {
			w.log.WithError(err).Warn("failed to update peer")
		}
	}

	// compare with previous addresses
	for _, addr := range oldNode.Status.Addresses {
		ip := w.getPeerAddress(addr)
		if ip == nil {
			continue
		}

		if _, exists := newAddrs[*ip]; !exists {
			// doesn't exist anymore, remove peer
			if err := w.server.RemovePeer(ip); err != nil {
				w.log.WithError(err).Warn("failed to remove peer")
			}
		}
	}
}

func (w *watcher) OnDelete(obj interface{}) {
	node, ok := obj.(*k8s.Node)
	if !ok {
		w.log.Warnf("invalid object received: %T", obj)
		return
	}

	for _, addr := range node.Status.Addresses {
		ip := w.getPeerAddress(addr)
		if ip == nil {
			continue
		}
		w.server.RemovePeer(ip)
	}
}
