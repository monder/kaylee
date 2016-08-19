package lib

import (
	"encoding/json"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/monder/kaylee/spec"
)

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

func (u *Units) ReloadAll(schedule func(*spec.Spec, bool)) {
	etcdAPI := u.getEtcdAPI()
	resp, err := etcdAPI.Get(
		context.Background(),
		u.EtcdKey,
		&etcd.GetOptions{Recursive: true},
	)
	Assert(err)
	for _, node := range resp.Node.Nodes {
		var unit spec.Spec
		err = json.Unmarshal([]byte(node.Value), &unit)
		if err != nil {
			fmt.Printf("Unable to parse unit %s. Err: %s\n", node.Key, err)
		}
		schedule(&unit, false)
	}
}
