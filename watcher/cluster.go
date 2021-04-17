package watcher

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const defaultResync = 1 * time.Minute

type KubernetesConfig struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Token     string `yaml:"token"`
	TokenFile string `yaml:"token_file"`
}

func (k *KubernetesConfig) ForClientSet() *rest.Config {
	return &rest.Config{
		Host:            k.Host,
		BearerToken:     k.Token,
		BearerTokenFile: k.TokenFile,
	}
}

func WatchClusters(ctx context.Context, log logrus.FieldLogger, configs []KubernetesConfig, bgp BGPServer) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // for proper cleanup on panics

	for _, conf := range configs {
		clog := log.WithField("cluster_host", conf.Host)

		// configure kubernetes client
		k8sConf := conf.ForClientSet()
		client, err := kubernetes.NewForConfig(k8sConf)
		if err != nil {
			clog.WithError(err).Error("failed to create kubernetes client")
			continue
		}

		if conf.Name == "" {
			clog.Error("Cluster name missing in config")
			continue
		}
		clog = clog.WithField("cluster_name", conf.Name)

		// setup watching
		factory := informers.NewSharedInformerFactory(client, defaultResync)
		informer := factory.Core().V1().Nodes().Informer()
		watch := newWatcher(bgp, clog, conf.Name)
		informer.AddEventHandler(watch)
		go informer.Run(ctx.Done())
		clog.Info("Node watcher started")
	}

	<-ctx.Done()
}
