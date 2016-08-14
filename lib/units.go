package lib

import (
	"encoding/json"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
)

type Unit struct {
	Name               string
	Replicas           int
	MaxReplicasPerHost int `json:"maxReplicasPerHost,omitempty"`

	EnvFiles []string `json:"envFiles,omitempty"`
	Env      []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"env,omitempty"`

	Volumes []struct {
		ID      string `json:"id"`
		Driver  string `json:"driver"`
		Path    string `json:"path"`
		Options string `json:"options"`
	} `json:"volumes,omitempty"`

	Net string `json:"net,omitempty"`

	Apps []struct {
		Image string   `json:"image"`
		Args  []string `json:"args,omitempty"`

		Volumes []struct {
			ID      string `json:"id"`
			Driver  string `json:"driver"`
			Path    string `json:"path"`
			Options string `json:"options"`
		} `json:"volumes,omitempty"`
	}

	Args []string `json:"args,omitempty"`

	Machine   []string `json:"machine,omitempty"`
	MachineID string   `json:"machineId,omitempty"`
	Global    bool     `json:"global,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
}

type Units struct {
	EtcdEndpoints []string
	EtcdKey       string
}

func (u *Units) getEtcdAPI() etcd.KeysAPI {
	c, err := etcd.New(etcd.Config{Endpoints: u.EtcdEndpoints})
	Assert(err)
	etcdAPI := etcd.NewKeysAPI(c)
	// Create dir if doesn't exist
	etcdAPI.Set(
		context.Background(),
		u.EtcdKey,
		"",
		&etcd.SetOptions{Dir: true},
	)
	return etcdAPI
}

func (u *Units) ReloadAll(schedule func(*Unit, bool)) {
	etcdAPI := u.getEtcdAPI()
	resp, err := etcdAPI.Get(
		context.Background(),
		u.EtcdKey,
		&etcd.GetOptions{Recursive: true},
	)
	Assert(err)
	for _, node := range resp.Node.Nodes {
		var unit Unit
		err = json.Unmarshal([]byte(node.Value), &unit)
		if err != nil {
			fmt.Printf("Unable to parse unit %s. Err: %s\n", node.Key, err)
		}
		schedule(&unit, false)
	}
}

func (u *Units) WatchForChanges(isMaster *bool, schedule func(*Unit, bool)) {
	etcdAPI := u.getEtcdAPI()
	watcher := etcdAPI.Watcher(
		u.EtcdKey,
		&etcd.WatcherOptions{
			AfterIndex: 0,
			Recursive:  true,
		},
	)
	for {
		change, err := watcher.Next(context.Background())
		if *isMaster == false {
			fmt.Println("Not watching anymore")
			return
		}
		if err != nil { // e.g. outdated event
			u.ReloadAll(schedule)
			return // TODO
		}
		if change.Node != nil {
			var unit Unit
			err = json.Unmarshal([]byte(change.Node.Value), &unit)
			if err != nil {
				fmt.Printf("Unable to parse unit %s. Err: %s\n", change.Node.Key, err)
				continue
			}
			schedule(&unit, true)
		}
	}
}
