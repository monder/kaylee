package main

import (
	"flag"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	fleetClient "github.com/coreos/fleet/client"
	"github.com/coreos/fleet/registry"
	"github.com/monder/kaylee/lib"
	"log"
	"os"
	"strings"
	"time"
)

var opts struct {
	etcdEndpoints string
	etcdPrefix    string
	fleetPrefix   string
	unitPrefix    string
	help          bool
}

func init() {
	flag.StringVar(&opts.etcdEndpoints, "etcd-endpoints", "http://127.0.0.1:4001,http://127.0.0.1:2379", "a comma-delimited list of etcd endpoints")
	flag.StringVar(&opts.etcdPrefix, "etcd-prefix", "/kaylee", "etcd prefix to store the job specs")
	flag.StringVar(&opts.fleetPrefix, "fleet-prefix", "/_coreos.com/fleet/", "Keyspace for fleet data in etcd")
	flag.StringVar(&opts.unitPrefix, "unit-prefix", "kaylee", "Prefix for units in fleet")
	flag.BoolVar(&opts.help, "help", false, "Print this message")
}

func parseFlags() {
	flag.Set("logtostderr", "true")

	flag.Parse()

	if flag.NArg() > 0 || opts.help {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}
}

func getFleet() (*lib.Fleet, error) {
	etcdConf, err := etcd.New(etcd.Config{
		Endpoints: strings.Split(opts.etcdEndpoints, ","),
	})
	if err != nil {
		return nil, err
	}
	etcdKeys := etcd.NewKeysAPI(etcdConf)
	reg := registry.NewEtcdRegistry(etcdKeys, opts.fleetPrefix, 3.0*time.Second)
	fleet := lib.Fleet{
		API: &fleetClient.RegistryClient{
			Registry: reg,
		},
		Prefix: opts.unitPrefix,
	}
	return &fleet, nil
}

func main() {
	parseFlags()

	fleet, err := getFleet()
	if err != nil {
		log.Fatal(err)
	}

	cluster := lib.ConnectToCluster(
		strings.Split(opts.etcdEndpoints, ","),
		opts.etcdPrefix,
		fleet,
	)
	cluster.Start()
}
