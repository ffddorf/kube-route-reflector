package main

import (
	"flag"
	"os"

	bnet "github.com/bio-routing/bio-rd/net"
	bgp "github.com/bio-routing/bio-rd/protocols/bgp/server"
	"github.com/ffddorf/kube-route-reflector/reflector"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesConfig struct {
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

type Config struct {
	Kubernetes KubernetesConfig    `yaml:"kubernetes"`
	BGP        reflector.BGPConfig `yaml:"bgp"`
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

	// configure kubernetes client
	k8sConf := conf.Kubernetes.ForClientSet()
	clientset, err := kubernetes.NewForConfig(k8sConf)
	if err != nil {
		log.WithError(err).Fatal("failed to create kubernetes client")
	}

	server := reflector.NewServer(bgpServer, clientset, conf.BGP)
	if err := server.Start(log); err != nil {
		log.WithError(err).Fatal("server failed")
	}
}
