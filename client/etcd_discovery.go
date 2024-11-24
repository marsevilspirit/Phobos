package client

import (
	"strings"
	"sync"
	"time"

	"github.com/marsevilspirit/m_RPC/log"
	"github.com/rpcxio/libkv"
	"github.com/rpcxio/libkv/store"
	estore "github.com/rpcxio/rpcx-etcd/store"
	"github.com/rpcxio/rpcx-etcd/store/etcdv3"
)

func init() {
	libkv.AddStore(estore.ETCDV3, etcdv3.New)
}

type EtcdDiscovery struct {
	basePath string
	kv       store.Store
	pairs    []*KVPair
	chans    []chan []*KVPair
	mu       sync.Mutex

	RetriesAfterWatchFailed int

	stopCh chan struct{}
}

func NewEtcdDiscovery(basePath string, etcdAddr []string) ServiceDiscovery {
	kv, err := libkv.NewStore(estore.ETCDV3, etcdAddr, nil)
	if err != nil {
		log.Infof("cannot create store: %v", err)
		panic(err)
	}

	return NewEtcdDiscoveryStore(basePath, kv)
}

func NewEtcdDiscoveryStore(basePath string, kv store.Store) ServiceDiscovery {
	if len(basePath) > 1 && strings.HasSuffix(basePath, "/") {
		basePath = basePath[:len(basePath)-1]
	}

	// mrpc_example/HelloWorld
	d := &EtcdDiscovery{basePath: basePath, kv: kv}
	d.stopCh = make(chan struct{})

	ps, err := kv.List(basePath)
	if err != nil {
		log.Infof("cannot get services of from registry: %v", basePath, err)
		panic(err)
	}

	pairs := make([]*KVPair, 0, len(ps))
	var prefix string
	for _, p := range ps {
		if prefix == "" {
			if strings.HasPrefix(p.Key, "/") {
				if strings.HasPrefix(d.basePath, "/") {
					prefix = d.basePath + "/"
				} else {
					prefix = "/" + d.basePath + "/"
				}
			} else {
				if strings.HasPrefix(d.basePath, "/") {
					prefix = d.basePath[1:] + "/"
				} else {
					prefix = d.basePath + "/"
				}
			}
		}
		if p.Key == prefix[:len(prefix)-1] || !strings.HasPrefix(p.Key, prefix) {
			continue
		}
		k := strings.TrimPrefix(p.Key, prefix)
		pairs = append(pairs, &KVPair{Key: k, Value: string(p.Value)})
	}

	d.pairs = pairs
	d.RetriesAfterWatchFailed = -1

	go d.watch()

	return d
}

func NewEtcdDiscoveryTemplate(basePath string, etcdAddr []string, options *store.Config) ServiceDiscovery {
	if len(basePath) > 1 && strings.HasSuffix(basePath, "/") {
		basePath = basePath[:len(basePath)-1]
	}

	kv, err := libkv.NewStore(estore.ETCDV3, etcdAddr, options)
	if err != nil {
		log.Infof("cannot create store: %v", err)
		panic(err)
	}

	return &EtcdDiscovery{basePath: basePath, kv: kv}
}

func (d EtcdDiscovery) Clone(servicePath string) ServiceDiscovery {
	return NewEtcdDiscoveryStore(d.basePath+"/"+servicePath, d.kv)
}

func (d EtcdDiscovery) GetServices() []*KVPair {
	return d.pairs
}

func (d *EtcdDiscovery) WatchService() chan []*KVPair {
	ch := make(chan []*KVPair, 10)
	d.chans = append(d.chans, ch)
	return ch
}

func (d *EtcdDiscovery) RemoveWatcher(ch chan []*KVPair) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, c := range d.chans {
		if c == ch {
			d.chans = append(d.chans[:i], d.chans[i+1:]...)
			return
		}
	}
}

func (d *EtcdDiscovery) watch() {
	for {
		var err error
		var c <-chan []*store.KVPair
		var tempDelay time.Duration

		retry := d.RetriesAfterWatchFailed

		for d.RetriesAfterWatchFailed == -1 || retry > 0 {

			c, err = d.kv.WatchTree(d.basePath, nil)
			if err != nil {
				if d.RetriesAfterWatchFailed > 0 {
					retry--
				}

				if tempDelay == 0 {
					tempDelay = 1 * time.Second
				} else {
					tempDelay *= 2
				}

				if max := 30 * time.Second; tempDelay > max {
					tempDelay = max
				}

				log.Warnf("can not watchtree (with retry %d, sleep %v): %s: %v", retry, tempDelay, d.basePath, err)
				time.Sleep(tempDelay)

				continue
			}
			break
		}

		if err != nil {
			log.Warnf("can not watchtree (with retry %d): %s: %v", retry, d.basePath, err)
			return
		}

	readChanges:
		for {
			select {
			case <-d.stopCh:
				log.Info("watcher is stopped")
				return
			case ps := <-c:
				if ps == nil {
					log.Warn("watcher chan is closed and will rewatch")
					break readChanges
				}
				var pairs []*KVPair
				var prefix string
				for _, p := range ps {
					if prefix == "" {
						if strings.HasPrefix(p.Key, "/") {
							if strings.HasPrefix(d.basePath, "/") {
								prefix = d.basePath + "/"
							} else {
								prefix = "/" + d.basePath + "/"
							}
						} else {
							if strings.HasPrefix(d.basePath, "/") {
								prefix = d.basePath[1:] + "/"
							} else {
								prefix = d.basePath + "/"
							}
						}
					}
					if p.Key == prefix[:len(prefix)-1] || !strings.HasPrefix(p.Key, prefix) {
						continue
					}
					k := strings.TrimPrefix(p.Key, prefix)
					pairs = append(pairs, &KVPair{Key: k, Value: string(p.Value)})
				}
				d.pairs = pairs

				for _, ch := range d.chans {
					ch := ch
					go func() {
						defer func() {
							if r := recover(); r != nil {
								log.Warn("watcher chan is closed")
							}
						}()
						select {
						case ch <- pairs:
						case <-time.After(time.Minute):
							log.Warn("chan is full and new change has hen dropped")
						}
					}()
				}
			}

			log.Warn("chan is closed and will rewatch")
		}
	}
}

func (d *EtcdDiscovery) Close() {
	close(d.stopCh)
}
