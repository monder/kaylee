package lib

import (
	"encoding/json"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
)

type Unit struct {
	Name string `json:"name"`
	Spec struct {
		Replicas           int      `json:"replicas"`
		MaxReplicasPerHost int      `json:"maxReplicasPerHost" yaml:"maxReplicasPerHost"`
		Image              string   `json:"image"`
		Cmd                string   `json:"cmd,omitempty"`
		EnvFiles           []string `json:"envFiles,omitempty" yaml:"envFiles,omitempty"`
		Env                []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"env,omitempty"`
		Labels []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"labels,omitempty"`
		Machine    []string `json:"machine,omitempty"`
		MachineID  string   `json:"machineId,omitempty" yaml:"machineId,omitempty"`
		DockerArgs []string `json:"dockerArgs,omitempty" yaml:"dockerArgs,omitempty"`
		Global     bool     `json:"global,omitempty"`
	} `json:"spec"`
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

func (u *Units) ReloadAll(cb func(*Unit)) {
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
		cb(&unit)
	}
}

func (u *Units) WatchForChanges(isMaster *bool, cb func(*Unit)) {
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
		Assert(err)
		if *isMaster == false {
			fmt.Println("Not watching anymore")
			return
		}
		if change.Node != nil && (change.PrevNode == nil || change.Node.Value != change.PrevNode.Value) {
			var unit Unit
			err = json.Unmarshal([]byte(change.Node.Value), &unit)
			if err != nil {
				fmt.Printf("Unable to parse unit %s. Err: %s\n", change.Node.Key, err)
			}
			cb(&unit)
		}
	}
}
