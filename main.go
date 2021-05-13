package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	bnet "github.com/bio-routing/bio-rd/net"
	bgpapi "github.com/bio-routing/bio-rd/protocols/bgp/api"
	bgp "github.com/bio-routing/bio-rd/protocols/bgp/server"
	"github.com/ffddorf/kube-route-reflector/reflector"
	"github.com/ffddorf/kube-route-reflector/watcher"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type APIConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
}

type Config struct {
	Clusters []watcher.KubernetesConfig `yaml:"clusters"`
	BGP      reflector.BGPConfig        `yaml:"bgp"`
	API      APIConfig                  `yaml:"api"`
}

func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	conf := new(Config)
	if err := yaml.NewDecoder(f).Decode(&conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func main() {
	log := logrus.New()

	configFileFlag := flag.String("config", "", "yaml config file to use")
	debugFlag := flag.Bool("debug", false, "turn on debug logging")
	flag.Parse()

	if *debugFlag {
		log.SetLevel(logrus.DebugLevel)
		logrus.SetLevel(logrus.DebugLevel) // for bio
	}

	conf, err := loadConfig(*configFileFlag)
	if err != nil {
		log.WithError(err).Fatal("failed to read config")
	}

	// configure BGP server
	rID, err := bnet.IPFromString(conf.BGP.RouterID)
	if err != nil {
		log.WithError(err).Fatal("failed to parse router id")
	}
	bgpServer := bgp.NewBGPServer(rID.ToUint32(), []string{
		"[::]:179",
		"0.0.0.0:179",
	})
	if conf.API.Enabled {
		if err := startBGPAPI(bgpServer, conf.API.Address); err != nil {
			log.WithError(err).Fatal("failed to start api server")
		}
	}

	server := reflector.NewServer(log.WithField("component", "reflector"), bgpServer, conf.BGP)
	if err := server.Start(log); err != nil {
		log.WithError(err).Fatal("bgp server failed to start")
	}

	// shutdown logic
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	watcher.WatchClusters(ctx, log, conf.Clusters, server)
}

func startBGPAPI(server bgp.BGPServer, address string) error {
	grpcServer := grpc.NewServer()

	api := bgp.NewBGPAPIServer(server)
	bgpapi.RegisterBgpServiceServer(grpcServer, api)

	if address == "" {
		address = "localhost:5566"
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	go grpcServer.Serve(lis)
	return nil
}
