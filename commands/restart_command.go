package commands

import (
	"fmt"
	"github.com/codegangsta/cli"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
)

func NewRestartCommand() cli.Command {
	return cli.Command{
		Name:      "restart",
		Usage:     "restarts specified unit",
		ArgsUsage: "<unit name>",
		Action:    restart,
	}
}

func restart(c *cli.Context) {
	if len(c.Args()) != 1 {
		cli.ShowCommandHelp(c, "restart")
		return
	}

	unitName := c.Args().First()
	etcdAPI := GetEtcdKeysAPI(c)
	resp, err := etcdAPI.Get(
		context.Background(),
		fmt.Sprintf("%s/units/%s", c.GlobalString("etcd-prefix"), unitName),
		&etcd.GetOptions{},
	)
	if err != nil {
		fmt.Println(err)
	}
	value := resp.Node.Value
	resp, err = etcdAPI.Set(
		context.Background(),
		fmt.Sprintf("%s/units/%s", c.GlobalString("etcd-prefix"), unitName),
		value,
		&etcd.SetOptions{PrevValue: value},
	)
	fmt.Println(resp)
	fmt.Println(err)
}
