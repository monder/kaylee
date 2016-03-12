package lib

import (
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/fsouza/go-dockerclient"
	"os"
	"time"
)

type Registrator struct {
	EtcdEndpoints []string
	EtcdKey       string
	ID            string
}

type registratorInternal struct {
	dockerClient *docker.Client
	etcd         etcd.KeysAPI
	etcdKey      string
	id           string
}

func (r *registratorInternal) addContainer(ID string) {
	c, _ := r.dockerClient.InspectContainer(ID)
	name := c.Config.Labels["s7r.name"]
	if name == "" {
		name = "unknown"
	}
	fmt.Printf("add %s/%s/%s/%s\n", r.etcdKey, r.id, name, c.Name)
	r.etcd.Set(
		context.Background(),
		fmt.Sprintf("%s/%s/%s/%s", r.etcdKey, r.id, name, c.Name),
		c.NetworkSettings.IPAddress,
		&etcd.SetOptions{},
	)
}

func (r *registratorInternal) removeContainer(ID string) {
	c, _ := r.dockerClient.InspectContainer(ID)
	name := c.Config.Labels["s7r.name"]
	if name == "" {
		name = "unknown"
	}
	fmt.Printf("delete %s/%s/%s/%s\n", r.etcdKey, r.id, name, c.Name)
	r.etcd.Delete(
		context.Background(),
		fmt.Sprintf("%s/%s/%s/%s", r.etcdKey, r.id, name, c.Name),
		&etcd.DeleteOptions{},
	)
	r.etcd.Delete(
		context.Background(),
		fmt.Sprintf("%s/%s/%s", r.etcdKey, r.id, name),
		&etcd.DeleteOptions{Dir: true},
	)
}

func (r *registratorInternal) resyncAll() {
	fmt.Println("Resync ", fmt.Sprintf("%s/%s", r.etcdKey, r.id))
	// TODO Make a proper diff
	r.etcd.Delete(context.Background(), fmt.Sprintf("%s/%s", r.etcdKey, r.id), &etcd.DeleteOptions{Recursive: true})

	containers, err := r.dockerClient.ListContainers(docker.ListContainersOptions{})
	Assert(err)
	for _, c := range containers {
		r.addContainer(c.ID)
	}
	if len(containers) == 0 {
		r.etcd.Set(context.Background(), fmt.Sprintf("%s/%s", r.etcdKey, r.id), "", &etcd.SetOptions{Dir: true})
	}
}

func (r *registratorInternal) watchForReload() {
	watcher := r.etcd.Watcher(
		fmt.Sprintf("%s/%s", r.etcdKey, r.id),
		&etcd.WatcherOptions{AfterIndex: 0},
	)
	for {
		change, err := watcher.Next(context.Background())
		Assert(err)
		if change.Node.Expiration != nil {
			fmt.Println("Removing expiration for ", fmt.Sprintf("%s/%s", r.etcdKey, r.id))
			r.etcd.Set(context.Background(), fmt.Sprintf("%s/%s", r.etcdKey, r.id), "",
				&etcd.SetOptions{
					Dir:       true,
					PrevExist: etcd.PrevExist,
				},
			)
			fmt.Println("Node expiration changed")
			r.resyncAll()
		} else if change.Node == nil && change.Action == "delete" {
			fmt.Println("%#v", change)
			fmt.Println("My id: ", fmt.Sprintf("%s/%s", r.etcdKey, r.id))
			fmt.Println("Node deleted: ", change.PrevNode.Key)
			r.resyncAll()
		}
	}
}

func (r *Registrator) ReloadAllInstances() {
	c, err := etcd.New(etcd.Config{Endpoints: r.EtcdEndpoints})
	Assert(err)
	etcdAPI := etcd.NewKeysAPI(c)

	fmt.Println("removing everything")
	resp, err := etcdAPI.Get(context.Background(), fmt.Sprintf("%s/", r.EtcdKey), &etcd.GetOptions{})
	Assert(err)

	for _, n := range resp.Node.Nodes {
		etcdAPI.Set(context.Background(), n.Key, "",
			&etcd.SetOptions{
				Dir:       true,
				TTL:       5 * time.Second,
				PrevExist: etcd.PrevExist,
			},
		)
		time.Sleep(10 * time.Second)
	}
}

func (r *Registrator) RunDockerLoop() {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		os.Setenv("DOCKER_HOST", "unix:///tmp/docker.sock")
	}
	dockerClient, err := docker.NewClientFromEnv()
	Assert(err)
	err = dockerClient.Ping()
	Assert(err)
	dockerEvents := make(chan *docker.APIEvents)
	err = dockerClient.AddEventListener(dockerEvents)
	Assert(err)
	c, err := etcd.New(etcd.Config{Endpoints: r.EtcdEndpoints})
	Assert(err)
	etcdAPI := etcd.NewKeysAPI(c)
	ri := &registratorInternal{
		dockerClient: dockerClient,
		etcd:         etcdAPI,
		etcdKey:      r.EtcdKey,
		id:           r.ID,
	}
	go ri.watchForReload()
	ri.resyncAll()

	for {
		select {
		case event := <-dockerEvents:
			switch event.Status {
			case "start":
				go ri.addContainer(event.ID)
			case "die":
				go ri.removeContainer(event.ID)
			}
		}
	}
}
