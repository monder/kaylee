package lib

import (
	"encoding/json"
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/niniwzw/etcdlock"
	"github.com/satori/go.uuid"
	"log"
)

type ClusterInstance struct {
	isMaster   bool
	watcher    etcd.Watcher
	fleet      *Fleet
	etcdAPI    etcd.KeysAPI
	etcdPrefix string
	lock       etcdlock.MasterInterface
}

func ConnectToCluster(etcdServerList []string, etcdPrefix string, fleet *Fleet) *ClusterInstance {
	etcdConf, err := etcd.New(etcd.Config{Endpoints: etcdServerList})
	if err != nil {
		log.Fatal(err)
	}
	etcdAPI := etcd.NewKeysAPI(etcdConf)

	lock, err := etcdlock.NewMaster(
		etcdlock.NewEtcdRegistry(etcdServerList),
		fmt.Sprintf("%s/master", etcdPrefix),
		uuid.NewV4().String(),
		30,
	)
	if err != nil {
		log.Fatal(err)
	}

	instance := &ClusterInstance{
		lock:       lock,
		etcdAPI:    etcdAPI,
		etcdPrefix: etcdPrefix,
		isMaster:   false,
		fleet:      fleet,
		watcher: etcdAPI.Watcher(
			fmt.Sprintf("%s/instances", etcdPrefix),
			&etcd.WatcherOptions{
				AfterIndex: 0,
				Recursive:  true,
			},
		),
	}

	return instance
}

func (instance *ClusterInstance) Start() {
	instance.lock.Start()
	instance.monitorMasterStatus(instance.lock.EventsChan())
}

func (instance *ClusterInstance) monitorMasterStatus(eventsCh <-chan etcdlock.MasterEvent) {
	for {
		select {
		case e := <-eventsCh:
			if e.Type == etcdlock.MasterAdded {
				log.Println("->Acquired the lock.")
				instance.isMaster = true
				instance.reloadUnits()
				go instance.watchForUnits()
			} else if e.Type == etcdlock.MasterDeleted {
				instance.isMaster = false
				log.Println("->Lost the lock.")
			} else {
				log.Println("->Lock ownership changed.")
			}
		}

	}
}

func (instance *ClusterInstance) reloadUnits() {
	instance.etcdAPI.Set(
		context.Background(),
		fmt.Sprintf("%s/instances", instance.etcdPrefix),
		"",
		&etcd.SetOptions{Dir: true},
	)
	resp, err := instance.etcdAPI.Get(
		context.Background(),
		fmt.Sprintf("%s/instances", instance.etcdPrefix),
		&etcd.GetOptions{
			Recursive: true,
		},
	)
	for _, node := range resp.Node.Nodes {
		var unit Unit
		err = json.Unmarshal([]byte(node.Value), &unit)
		if err != nil {
			log.Printf("Unable to parse unit %s. Err: %s\n", node.Key, err)
		}
		instance.fleet.ScheduleUnit(unit)
	}
}

func (instance *ClusterInstance) watchForUnits() {
	for {
		if instance.isMaster == false {
			return
		}
		change, err := instance.watcher.Next(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		if change.Node != nil && (change.PrevNode == nil || change.Node.Value != change.PrevNode.Value) {
			var unit Unit
			err = json.Unmarshal([]byte(change.Node.Value), &unit)
			if err != nil {
				log.Printf("Unable to parse unit %s. Err: %s\n", change.Node.Key, err)
			}
			instance.fleet.ScheduleUnit(unit)
		}
	}
}
