package commands

import (
	"encoding/json"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/ghodss/yaml"
	"github.com/monder/kaylee/spec"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
)

func NewRunCommand() cli.Command {
	return cli.Command{
		Name:      "run",
		Usage:     "runs a new unit or updates the existing one",
		ArgsUsage: "<file>",
		Action: func(c *cli.Context) error {
			fileName := c.Args().First()
			if len(fileName) == 0 {
				cli.ShowCommandHelp(c, "run")
				return nil
			}
			yamlFile, err := ioutil.ReadFile(fileName)

			fmt.Println(err)
			var unit spec.Spec
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
