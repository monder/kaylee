package main

import (
	"flag"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	fleetClient "github.com/coreos/fleet/client"
	"github.com/coreos/fleet/registry"
	"github.com/monder/kaylee/lib"
	"github.com/satori/go.uuid"
	"os"
	"strings"
	"time"

	"os/signal"
	"syscall"
)

var opts struct {
	etcdEndpoints string

	etcdPrefix  string
	fleetPrefix string
	unitPrefix  string

	help bool
}

var state struct {
	isMaster bool
}

func init() {
	flag.StringVar(&opts.etcdEndpoints, "etcd-endpoints", "http://127.0.0.1:4001,http://127.0.0.1:2379", "a comma-delimited list of etcd endpoints")

	flag.StringVar(&opts.etcdPrefix, "etcd-prefix", "/kaylee", "Keyspace for our data in etcd")
	flag.StringVar(&opts.fleetPrefix, "fleet-prefix", "/_coreos.com/fleet/", "Keyspace for fleet data in etcd")
	flag.StringVar(&opts.unitPrefix, "unit-prefix", "k", "Prefix for units in fleet")

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

func cleanup() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	if state.isMaster == true {
		c, _ := etcd.New(etcd.Config{
			Endpoints: strings.Split(opts.etcdEndpoints, ","),
		})
		etcdAPI := etcd.NewKeysAPI(c)
		fmt.Println("Deleting master")
		etcdAPI.Delete(context.Background(), fmt.Sprintf("%s/master", opts.etcdPrefix), &etcd.DeleteOptions{})
	}

	os.Exit(0)
}

func main() {
	parseFlags()

	go cleanup()

	id := uuid.NewV4().String()

	cluster := &lib.Cluster{
		EtcdEndpoints: strings.Split(opts.etcdEndpoints, ","),
		EtcdKey:       fmt.Sprintf("%s/master", opts.etcdPrefix),
		ID:            id,
	}
	units := &lib.Units{
		EtcdEndpoints: strings.Split(opts.etcdEndpoints, ","),
		EtcdKey:       fmt.Sprintf("%s/units", opts.etcdPrefix),
	}

	fleet, err := getFleet()
	lib.Assert(err)

	registrator := &lib.Registrator{
		EtcdEndpoints: strings.Split(opts.etcdEndpoints, ","),
		EtcdKey:       fmt.Sprintf("%s/instances", opts.etcdPrefix),
		ID:            id,
	}

	go cluster.MonitorMasterState(func(isMaster bool) {
		fmt.Println("Master: ", isMaster)
		state.isMaster = isMaster
		if isMaster {
			go registrator.ReloadAllInstances()
			units.ReloadAll(fleet.ScheduleUnit)
			go units.WatchForChanges(&state.isMaster, fleet.ScheduleUnit)
		}
	})

	registrator.RunDockerLoop()
}
