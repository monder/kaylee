package commands

import (
	"encoding/json"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	fleetClient "github.com/coreos/fleet/client"
	"github.com/monder/kaylee/lib"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/urfave/cli.v1"
	"os"

	"github.com/coreos/fleet/registry"
	"strings"
	"time"
)

func NewLsCommand() cli.Command {
	return cli.Command{
		Name:      "ls",
		Usage:     "list units",
		ArgsUsage: " ",
		Action: func(c *cli.Context) error {
			etcdAPI := GetEtcdKeysAPI(c)
			res, err := etcdAPI.Get(
				context.Background(),
				fmt.Sprintf("%s/units/", c.GlobalString("etcd-prefix")),
				&etcd.GetOptions{Sort: true},
			)
			if err != nil {
				fmt.Println(err)
				return err
			}
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Unit name", "Fleet instance name", "Status"})

			// TODO
			fll := fleetClient.RegistryClient{
				Registry: registry.NewEtcdRegistry(etcdAPI, "/_coreos.com/fleet/", 3.0*time.Second),
			}
			fleetUnits, err := fll.UnitStates()

			for _, node := range res.Node.Nodes {
				var unit lib.Unit
				err = json.Unmarshal([]byte(node.Value), &unit)
				if err != nil {
					fmt.Printf("Unable to parse unit %s. Err: %s\n", node.Key, err)
					continue
				}
				firstLine := true
				for _, unitState := range fleetUnits {
					if strings.HasPrefix(unitState.Name, fmt.Sprintf("%s:%s:", c.GlobalString("etcd-prefix"), unit.Name)) {
						line := []string{unit.Name, unitState.Name, unitState.SystemdSubState}
						if !firstLine {
							line[0] = ""
						}
						firstLine = false
						table.Append(line)
					}
				}
			}

			table.Render()
			return nil
		},
	}
}
