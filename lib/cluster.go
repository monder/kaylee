package lib

import (
	"github.com/niniwzw/etcdlock"
)

type Cluster struct {
	EtcdEndpoints []string
	EtcdKey       string
	ID            string
}

func (c *Cluster) MonitorMasterState(cb func(isMaster bool)) {
	lock, err := etcdlock.NewMaster(
		etcdlock.NewEtcdRegistry(c.EtcdEndpoints),
		c.EtcdKey,
		c.ID,
		30,
	)
	Assert(err)

	lock.Start()
	for {
		select {
		case e := <-lock.EventsChan():
			cb(e.Master == c.ID)
		}
	}
}
