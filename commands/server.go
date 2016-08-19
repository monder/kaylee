package commands

import (
	"encoding/json"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	fleetClient "github.com/coreos/fleet/client"
	"github.com/coreos/fleet/registry"
	"github.com/monder/kaylee/fleet"
	"github.com/monder/kaylee/spec"
	"github.com/niniwzw/etcdlock"
	"github.com/satori/go.uuid"
	"gopkg.in/urfave/cli.v1"
	"os"
	"strings"
	"time"

	"os/signal"
	"syscall"
)

var Server cli.Command

func init() {
	Server = cli.Command{
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
		Action: startServer,
	}
}

func getFleet(c *cli.Context) (*fleet.Fleet, error) {
	etcdAPI := GetEtcdKeysAPI(c)
	reg := registry.NewEtcdRegistry(etcdAPI, c.String("fleet-prefix"), 3.0*time.Second)
	fleet := fleet.Fleet{
		API: &fleetClient.RegistryClient{
			Registry: reg,
		},
		Prefix: c.String("unit-prefix"),
	}
	return &fleet, nil
}

func cleanup(c *cli.Context, id string) {
	etcdClient, _ := etcd.New(etcd.Config{
		Endpoints: strings.Split(c.GlobalString("etcd-endpoints"), ","),
	})
	etcdAPI := etcd.NewKeysAPI(etcdClient)
	etcdAPI.Delete(context.Background(), fmt.Sprintf("%s/master", c.GlobalString("etcd-prefix")), &etcd.DeleteOptions{
		PrevValue: id,
	})
}

func monitorExit() <-chan os.Signal {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	return sigs
}

func monitorMasterStatus(c *cli.Context, id string) (<-chan etcdlock.MasterEvent, error) {
	lock, err := etcdlock.NewMaster(
		etcdlock.NewEtcdRegistry(strings.Split(c.GlobalString("etcd-endpoints"), ",")),
		fmt.Sprintf("%s/master", c.GlobalString("etcd-prefix")),
		id,
		600,
	)
	if err != nil {
		return nil, err
	}
	lock.Start()
	return lock.EventsChan(), nil
}

type UnitEvent struct {
	Error  error
	Action string
	Unit   *spec.Spec
}

func monitorUnitSpecs(c *cli.Context) <-chan UnitEvent {
	etcdAPI := GetEtcdKeysAPI(c)
	// Create a dir if not exist
	etcdAPI.Set(context.Background(), fmt.Sprintf("%s/units", c.GlobalString("etcd-prefix")), "", &etcd.SetOptions{Dir: true})
	unitWatcher := etcdAPI.Watcher(
		fmt.Sprintf("%s/units", c.GlobalString("etcd-prefix")),
		&etcd.WatcherOptions{
			AfterIndex: 0,
			Recursive:  true,
		},
	)
	unitEvents := make(chan UnitEvent, 10)
	go func() {
		for {
			change, err := unitWatcher.Next(context.Background())
			if err != nil {
				unitEvents <- UnitEvent{Error: err}
			} else if change.Action == "delete" {
				var unit spec.Spec
				err = json.Unmarshal([]byte(change.PrevNode.Value), &unit)
				if err != nil {
					fmt.Printf("Unable to parse spec %s. Err: %s\n", change.Node.Key, err)
					continue
				}
				unitEvents <- UnitEvent{Action: "remove", Unit: &unit}
			} else {
				var unit spec.Spec
				err = json.Unmarshal([]byte(change.Node.Value), &unit)
				if err != nil {
					fmt.Printf("Unable to parse spec %s. Err: %s\n", change.Node.Key, err)
					continue
				}
				unitEvents <- UnitEvent{Action: "add", Unit: &unit}
			}
		}
	}()
	return unitEvents
}

func reloadAll(c *cli.Context, fleet *fleet.Fleet) error {
	etcdAPI := GetEtcdKeysAPI(c)
	resp, err := etcdAPI.Get(
		context.Background(),
		fmt.Sprintf("%s/units", c.GlobalString("etcd-prefix")),
		&etcd.GetOptions{Recursive: true},
	)
	if err != nil {
		return err
	}
	for _, node := range resp.Node.Nodes {
		var unit spec.Spec
		err = json.Unmarshal([]byte(node.Value), &unit)
		if err != nil {
			fmt.Printf("Unable to parse unit %s. Err: %s\n", node.Key, err)
		}
		fleet.ScheduleUnit(&unit, false)
	}
	return nil
}

func startServer(c *cli.Context) error {
	id := uuid.NewV4().String()
	fmt.Printf("Launching new node with id: %s\n", id)

	fleet, err := getFleet(c)
	if err != nil {
		return cli.NewExitError(err.Error(), 2)
	}

	// Exit signal
	sigsChan := monitorExit()

	// Master change signal
	masterChan, err := monitorMasterStatus(c, id)
	if err != nil {
		return cli.NewExitError(err.Error(), 2)
	}

	// Unit spec monitoring
	unitsChan := monitorUnitSpecs(c)

	isMaster := false
	fmt.Println("Waiting to become master")
	for {
		select {
		case e := <-masterChan:
			if e.Type == etcdlock.MasterAdded {
				isMaster = true
				fmt.Println("Master status acquired")
				reloadAll(c, fleet) // TODO handle error?
			} else if e.Type == etcdlock.MasterDeleted {
				isMaster = false
				fmt.Println("Master status lost")
			}
		case ue := <-unitsChan:
			if isMaster {
				if ue.Error != nil {
					fmt.Println("Error in etcd. Exiting...")
					cleanup(c, id)
					return cli.NewExitError(ue.Error.Error(), 2)
				} else if ue.Action == "add" {
					fmt.Printf("Updating unit %s\n", ue.Unit.Name)
					fleet.ScheduleUnit(ue.Unit, true)
				} else if ue.Action == "remove" {
					fmt.Printf("Should remove unit %s [NOT IMPLEMENTED]\n", ue.Unit.Name) //TODO
				}
			}
		case <-sigsChan:
			fmt.Println("Cleaning up...")
			cleanup(c, id)
			return nil
		}
	}
}
