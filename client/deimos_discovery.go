package client

import (
	"context"
	"strings"
	"sync"
	"time"

	deimos "github.com/marsevilspirit/deimos-client"

	"github.com/marsevilspirit/phobos/log"
)

// DeimosDiscovery implements service discovery using the deimos client.
type DeimosDiscovery struct {
	basePath string
	client   *deimos.Client
	pairs    []*KVPair
	chans    []chan []*KVPair
	mu       sync.Mutex
	stopCh   chan struct{}
}

// NewDeimosDiscovery creates a new DeimosDiscovery with a new client.
func NewDeimosDiscovery(basePath string, deimosAddr []string) ServiceDiscovery {
	client := deimos.NewClient(deimosAddr)
	return NewDeimosDiscoveryStore(basePath, client)
}

// NewDeimosDiscoveryStore creates a new DeimosDiscovery with a provided client.
func NewDeimosDiscoveryStore(basePath string, client *deimos.Client) ServiceDiscovery {
	if len(basePath) > 1 && strings.HasSuffix(basePath, "/") {
		basePath = basePath[:len(basePath)-1]
	}

	d := &DeimosDiscovery{
		basePath: basePath,
		client:   client,
		stopCh:   make(chan struct{}),
	}

	// Initial fetch of all services under the base path.
	resp, err := client.Get(context.Background(), basePath, deimos.WithRecursive())
	if err != nil {
		log.Infof("cannot get services from deimos registry: %s, error: %v", basePath, err)
		panic(err)
	}

	// The Get response for a directory contains nodes under it.
	if resp.Node != nil && resp.Node.Dir {
		prefix := d.basePath + "/"
		for _, node := range resp.Node.Nodes {
			if node.Dir { // We are interested in service nodes, not sub-directories.
				continue
			}
			key := strings.TrimPrefix(node.Key, prefix)
			d.pairs = append(d.pairs, &KVPair{Key: key, Value: node.Value})
		}
	}

	// Start the background watcher.
	go d.watch()

	return d
}

// Clone creates a new discovery for a sub-path.
func (d *DeimosDiscovery) Clone(servicePath string) ServiceDiscovery {
	return NewDeimosDiscoveryStore(d.basePath+"/"+servicePath, d.client)
}

// GetServices returns the current cached list of services.
func (d *DeimosDiscovery) GetServices() []*KVPair {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pairs
}

// WatchService registers a channel to receive service updates.
func (d *DeimosDiscovery) WatchService() chan []*KVPair {
	d.mu.Lock()
	defer d.mu.Unlock()
	ch := make(chan []*KVPair, 10)
	d.chans = append(d.chans, ch)
	return ch
}

// RemoveWatcher unregisters a watcher channel.
func (d *DeimosDiscovery) RemoveWatcher(ch chan []*KVPair) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, c := range d.chans {
		if c == ch {
			d.chans = append(d.chans[:i], d.chans[i+1:]...)
			return
		}
	}
}

// watch is the background goroutine that watches deimos for changes.
func (d *DeimosDiscovery) watch() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-d.stopCh // Wait for the Close() signal.
		cancel()   // Cancel the context to stop the watcher.
	}()

	var waitIndex uint64
	for {
		// Exit loop if the watcher has been stopped.
		select {
		case <-d.stopCh:
			log.Info("discovery watcher is stopped")
			return
		default:
		}

		// Start watching recursively from the last known index.
		watchChan := d.client.Watch(ctx, d.basePath, deimos.WithRecursive(), deimos.WithWaitIndex(waitIndex))

		// Process events from the watch channel.
		for resp := range watchChan {
			waitIndex = resp.Node.ModifiedIndex + 1 // Update index to avoid getting the same event again.

			// Upon any change, re-fetch the entire directory to get the fresh state.
			// This is simpler and more robust than trying to apply individual deltas.
			getResp, err := d.client.Get(context.Background(), d.basePath, deimos.WithRecursive())
			if err != nil {
				log.Warnf("failed to get services after watch event: %v", err)
				time.Sleep(2 * time.Second) // Avoid spamming in case of persistent errors.
				continue
			}

			var currentPairs []*KVPair
			if getResp.Node != nil && getResp.Node.Dir {
				prefix := d.basePath + "/"
				for _, node := range getResp.Node.Nodes {
					if node.Dir {
						continue
					}
					key := strings.TrimPrefix(node.Key, prefix)
					currentPairs = append(currentPairs, &KVPair{Key: key, Value: node.Value})
				}
			}

			// Atomically update internal state and broadcast to all watchers.
			d.mu.Lock()
			d.pairs = currentPairs
			for _, ch := range d.chans {
				ch := ch // Capture channel variable for the goroutine.
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Warn("watcher chan seems to be closed; removing it")
							d.RemoveWatcher(ch)
						}
					}()
					select {
					case ch <- currentPairs:
					case <-time.After(time.Minute): // Prevent blocking forever.
						log.Warn("chan is full and new change has been dropped")
					}
				}()
			}
			d.mu.Unlock()
		}

		// If watchChan closes (e.g., context canceled or network error), the loop will restart
		// and attempt to re-establish the watch, providing resilience.
		log.Warn("watcher chan is closed and will rewatch")
	}
}

// Close stops the discovery watcher.
func (d *DeimosDiscovery) Close() {
	close(d.stopCh)
}
