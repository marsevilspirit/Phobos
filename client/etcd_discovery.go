package client

import (
	"strings"
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
}

func NewEtcdDiscovery(basePath string, etcdAddr []string) ServiceDiscovery {
	kv, err := libkv.NewStore(estore.ETCDV3, etcdAddr, nil)
	if err != nil {
		log.Infof("cannot create store: %v", err)
		panic(err)
	}

	// mrpc_example/HelloWorld
	d := &EtcdDiscovery{basePath: basePath, kv: kv}
	go d.watch()

	ps, err := kv.List(basePath)
	if err != nil {
		log.Infof("cannot get services of from registry: %v", basePath, err)
		panic(err)
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
	return d
}

func (d EtcdDiscovery) GetServices() []*KVPair {
	return d.pairs
}

func (d *EtcdDiscovery) WatchService() chan []*KVPair {
	ch := make(chan []*KVPair, 10)
	d.chans = append(d.chans, ch)
	return ch
}

func (d *EtcdDiscovery) watch() {
	c, err := d.kv.WatchTree(d.basePath, nil)
	if err != nil {
		log.Fatalf("can not watchtree: %s: %v", d.basePath, err)
		return
	}

	for ps := range c {
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
				select {
				case ch <- pairs:
				case <-time.After(time.Minute):
					log.Warn("chan is full and new change has hen dropped")
				}
			}()
		}
	}
}
