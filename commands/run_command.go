package commands

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/ghodss/yaml"
	"github.com/monder/kaylee/lib"
	"io/ioutil"
)

func NewRunCommand() cli.Command {
	return cli.Command{
		Name:      "run",
		Usage:     "runs a new unit or update an existing one",
		ArgsUsage: "<file>",
		Action: func(c *cli.Context) error {
			fileName := c.Args().First()
			if len(fileName) == 0 {
				cli.ShowCommandHelp(c, "run")
				return nil
			}
			yamlFile, err := ioutil.ReadFile(fileName)

			fmt.Println(err)
			var unit lib.Unit
			err = yaml.Unmarshal(yamlFile, &unit)
			fmt.Println(err)
			unitJson, err := json.Marshal(unit)

			etcdAPI := GetEtcdKeysAPI(c)
			aaa, err := etcdAPI.Set(
				context.Background(),
				fmt.Sprintf("%s/units/%s", c.GlobalString("etcd-prefix"), unit.Name),
				string(unitJson),
				&etcd.SetOptions{},
			)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println(aaa)
			return nil
		},
	}
}
