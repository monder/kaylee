package commands

import (
	"fmt"
	"github.com/codegangsta/cli"
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

func getFleet(c *cli.Context) (*lib.Fleet, error) {
	etcdAPI := GetEtcdKeysAPI(c)
	reg := registry.NewEtcdRegistry(etcdAPI, c.String("fleet-prefix"), 3.0*time.Second)
	fleet := lib.Fleet{
		API: &fleetClient.RegistryClient{
			Registry: reg,
		},
		Prefix: c.String("unit-prefix"),
	}
	return &fleet, nil
}

func cleanup(c *cli.Context, isMaster *bool) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	if *isMaster == true {
		etcdClient, _ := etcd.New(etcd.Config{
			Endpoints: strings.Split(c.GlobalString("etcd-endpoints"), ","),
		})
		etcdAPI := etcd.NewKeysAPI(etcdClient)
		fmt.Println("Deleting master")
		etcdAPI.Delete(context.Background(), fmt.Sprintf("%s/master", c.GlobalString("etcd-prefix")), &etcd.DeleteOptions{})
	}

	os.Exit(0)
}

func NewServerCommand() cli.Command {
	return cli.Command{
		Name:      "server",
		Usage:     "runs the scheduling server",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "fleet-prefix",
				Value: "/_coreos.com/fleet/",
				Usage: "keyspace for fleet data in etcd",
			},
			cli.StringFlag{
				Name:  "unit-prefix",
				Value: "k2",
				Usage: "prefix for units in fleet",
			},
		},
		Action: func(c *cli.Context) error {
			var isMaster bool
			go cleanup(c, &isMaster)

			id := uuid.NewV4().String()

			cluster := &lib.Cluster{
				EtcdEndpoints: strings.Split(c.GlobalString("etcd-endpoints"), ","),
				EtcdKey:       fmt.Sprintf("%s/master", c.GlobalString("etcd-prefix")),
				ID:            id,
			}
			units := &lib.Units{
				EtcdEndpoints: strings.Split(c.GlobalString("etcd-endpoints"), ","),
				EtcdKey:       fmt.Sprintf("%s/units", c.GlobalString("etcd-prefix")),
			}

			fleet, err := getFleet(c)
			lib.Assert(err)

			cluster.MonitorMasterState(func(master bool) {
				fmt.Println("Master: ", master)
				isMaster = master
				if isMaster {
					units.ReloadAll(fleet.ScheduleUnit)
					go units.WatchForChanges(&isMaster, fleet.ScheduleUnit)
				}
			})
			return nil
		},
	}
}
