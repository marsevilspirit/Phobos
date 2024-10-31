package client

import (
	"time"

	"github.com/marsevilspirit/m_RPC/log"
)

// 多个server的服务发现
type MultipleServersDiscovery struct {
	pairs []*KVPair
	chans []chan []*KVPair
}

func NewMultipleServersDiscovery(pairs []*KVPair) ServiceDiscovery {
	return &MultipleServersDiscovery{
		pairs: pairs,
	}
}

func (d MultipleServersDiscovery) GetServices() []*KVPair {
	return d.pairs
}

func (d MultipleServersDiscovery) WatchService() chan []*KVPair {
	ch := make(chan []*KVPair, 10)
	d.chans = append(d.chans, ch)
	return ch
}

func (d MultipleServersDiscovery) Update(pairs []*KVPair) {
	for _, ch := range d.chans {
		ch := ch
		go func() {
			select {
			case ch <- pairs:
			case <-time.After(time.Minute):
				log.Warn("chan is full and new change has ben dropped")
			}
		}()
	}
}
