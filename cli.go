package main

import (
	"encoding/json"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/monder/kaylee/lib"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.Println("hello")

	var unit lib.Unit
	filename, _ := filepath.Abs(os.Args[1])
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(yamlFile, &unit)
	if err != nil {
		panic(err)
	}
	a, err := json.Marshal(unit)
	if err != nil {
		panic(err)
	}
	log.Printf("Value: %#v\n", string(a))

	c, err := etcd.New(etcd.Config{
		Endpoints: []string{"http://52.17.158.201:4001"},
	})
	if err != nil {
		panic(err)
	}
	log.Println(c)
	etcdAPI := etcd.NewKeysAPI(c)
	// Create dir if doesn't exist
	aaa, err := etcdAPI.Set(
		context.Background(),
		fmt.Sprintf("/sched/units/%s", unit.Name),
		string(a),
		&etcd.SetOptions{},
	)
	if err != nil {
		panic(err)
	}
	log.Println(aaa)
}
