package main

import (
	"encoding/json"
	"flag"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
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
	flag.StringVar(&opts.unitPrefix, "unit-prefix", "kaylee", "prefix for units in fleet")
	flag.BoolVar(&opts.help, "help", false, "print this message")
}

func main() {
	flag.Set("logtostderr", "true")

	flag.Parse()

	if flag.NArg() > 0 || opts.help {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	etcdConf, err := etcd.New(etcd.Config{
		Endpoints: strings.Split(opts.etcdEndpoints, ","),
	})
	if err != nil {
		log.Fatal(err)
	}
	etcdKeys := etcd.NewKeysAPI(etcdConf)

	reg := registry.NewEtcdRegistry(etcdKeys, opts.fleetPrefix, 3.0*time.Second)
	fleet := lib.Fleet{
		API: &fleetClient.RegistryClient{
			Registry: reg,
		},
		Prefix: opts.unitPrefix,
	}

	resp, err := etcdKeys.Get(context.Background(), opts.etcdPrefix, &etcd.GetOptions{
		Recursive: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, node := range resp.Node.Nodes {
		var unit lib.Unit
		err = json.Unmarshal([]byte(node.Value), &unit)
		if err != nil {
			log.Printf("Unable to parse unit %s. Err: %s\n", node.Key, err)
		}
		fleet.ScheduleUnit(unit)
	}

	watcher := etcdKeys.Watcher(opts.etcdPrefix, &etcd.WatcherOptions{
		AfterIndex: 0,
		Recursive:  true,
	})

	for {
		resp, err := watcher.Next(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		if (resp.Action == "set") && (resp.Node != nil) && (resp.PrevNode != nil) && (resp.Node.Value != resp.PrevNode.Value) {
			var unit lib.Unit
			err = json.Unmarshal([]byte(resp.Node.Value), &unit)
			if err != nil {
				log.Printf("Unable to parse unit %s. Err: %s\n", resp.Node.Key, err)
			}
			fleet.ScheduleUnit(unit)
		}
	}
}
