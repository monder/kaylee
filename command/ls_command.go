package command

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/monder/kaylee/lib"
)

var Ls = cli.Command{
	Name:      "ls",
	Usage:     "list units",
	ArgsUsage: " ",
	Action: func(c *cli.Context) {
		etcdAPI := GetEtcdKeysAPI(c)
		res, err := etcdAPI.Get(
			context.Background(),
			fmt.Sprintf("%s/units/", c.GlobalString("etcd-prefix")),
			&etcd.GetOptions{Sort: true},
		)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Units:")
		for _, node := range res.Node.Nodes {
			var unit lib.Unit
			err = json.Unmarshal([]byte(node.Value), &unit)
			if err != nil {
				fmt.Printf("Unable to parse unit %s. Err: %s\n", node.Key, err)
			}
			fmt.Printf(" %s\n", unit.Name)
		}
	},
}
