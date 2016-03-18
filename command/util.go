package command

import (
	"fmt"
	"github.com/codegangsta/cli"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"os"
	"strings"
)

func GetEtcdKeysAPI(c *cli.Context) etcd.KeysAPI {
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints: strings.Split(c.GlobalString("etcd-endpoints"), ","),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	etcdAPI := etcd.NewKeysAPI(etcdClient)
	return etcdAPI
}
