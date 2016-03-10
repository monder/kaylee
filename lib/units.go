package lib

import (
	"encoding/json"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"log"
)

type Unit struct {
	Name string `json:"name"`
	Spec struct {
		Replicas           int    `json:"replicas"`
		MaxReplicasPerHost int    `json:"maxReplicasPerHost"`
		Image              string `json:"image"`
		Env                []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"env"`
		Labels []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"labels"`
		Machine    []string `json:"machine"`
		DockerArgs []string `json:"dockerArgs"`
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
			log.Printf("Unable to parse unit %s. Err: %s\n", node.Key, err)
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
			log.Println("Not watching anymore")
			return
		}
		if change.Node != nil && (change.PrevNode == nil || change.Node.Value != change.PrevNode.Value) {
			var unit Unit
			err = json.Unmarshal([]byte(change.Node.Value), &unit)
			if err != nil {
				log.Printf("Unable to parse unit %s. Err: %s\n", change.Node.Key, err)
			}
			cb(&unit)
		}
	}
}
