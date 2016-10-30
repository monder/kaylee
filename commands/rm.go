package commands

import (
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"gopkg.in/urfave/cli.v1"
)

var Remove cli.Command

func init() {
	Restart = cli.Command{
		Name:      "rm",
		Usage:     "destroys the specified unit",
		ArgsUsage: "<unit name>",
		Action:    remove,
	}
}

func remove(c *cli.Context) error {
	if len(c.Args()) != 1 {
		cli.ShowCommandHelp(c, "rm")
		return nil
	}

	unitName := c.Args().First()
	etcdAPI := GetEtcdKeysAPI(c)
	resp, err := etcdAPI.Delete(
		context.Background(),
		fmt.Sprintf("%s/units/%s", c.GlobalString("etcd-prefix"), unitName),
		&etcd.DeleteOptions{},
	)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)
	return err
}
