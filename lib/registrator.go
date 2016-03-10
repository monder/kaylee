package lib

import (
	"fmt"
	etcd "github.com/coreos/fleet/Godeps/_workspace/src/github.com/coreos/etcd/client"
	"github.com/coreos/fleet/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/fsouza/go-dockerclient"
	"os"
)

type Registrator struct {
	EtcdEndpoints []string
	EtdcKey       string
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

func (r *Registrator) RunDockerLoop() {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		os.Setenv("DOCKER_HOST", "unix:///tmp/docker.sock")
	}
	dockerClient, err := docker.NewClientFromEnv()
	Assert(err)
	_, err = dockerClient.Version()
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
		etcdKey:      r.EtdcKey,
		id:           r.ID,
	}
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
