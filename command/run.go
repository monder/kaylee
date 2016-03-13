package command

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/monder/kaylee/lib"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

var Run = cli.Command{
	Name:      "run",
	Usage:     "runs a new unit or update an existing one",
	ArgsUsage: "<file>",
	Action: func(c *cli.Context) {
		fileName := c.Args().First()
		if len(fileName) == 0 {
			cli.ShowCommandHelp(c, "run")
			return
		}
		yamlFile, err := ioutil.ReadFile(fileName)

		fmt.Println(err)
		var unit lib.Unit
		err = yaml.Unmarshal(yamlFile, &unit)
		fmt.Println(err)
		unitJson, err := json.Marshal(unit)

		etcdClient, err := etcd.New(etcd.Config{
			Endpoints: strings.Split(c.GlobalString("etcd-endpoints"), ","),
		})
		etcdAPI := etcd.NewKeysAPI(etcdClient)
		aaa, err := etcdAPI.Set(
			context.Background(),
			fmt.Sprintf("%s/units/%s", c.GlobalString("etcd-prefix"), unit.Name),
			string(unitJson),
			&etcd.SetOptions{},
		)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(aaa)
	},
}
