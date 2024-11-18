package client

import (
	"sync"
	"time"

	"github.com/marsevilspirit/m_RPC/log"
)

// 多个server的服务发现
type MultipleServersDiscovery struct {
	pairs []*KVPair
	chans []chan []*KVPair
	mu    sync.Mutex
}

func NewMultipleServersDiscovery(pairs []*KVPair) ServiceDiscovery {
	return &MultipleServersDiscovery{
		pairs: pairs,
	}
}

func (d MultipleServersDiscovery) Clone(servicePath string) ServiceDiscovery {
	return &d
}

func (d MultipleServersDiscovery) GetServices() []*KVPair {
	return d.pairs
}

func (d *MultipleServersDiscovery) WatchService() chan []*KVPair {
	ch := make(chan []*KVPair, 10)
	d.chans = append(d.chans, ch)
	return ch
}

func (d *MultipleServersDiscovery) RemoveWatcher(ch chan []*KVPair) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var chans []chan []*KVPair
	for _, c := range d.chans {
		if c != ch {
			chans = append(chans, c)
		}
	}
	d.chans = chans
}

func (d *MultipleServersDiscovery) Update(pairs []*KVPair) {
	for _, ch := range d.chans {
		ch := ch
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error("watcher chan closed")
				}
			}()
			select {
			case ch <- pairs:
			case <-time.After(time.Minute):
				log.Warn("chan is full and new change has ben dropped")
			}
		}()
	}
}
