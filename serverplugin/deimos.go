package serverplugin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	deimos "github.com/marsevilspirit/deimos-client"

	"github.com/marsevilspirit/phobos/log"
)

const (
	phobosDir = "/phobos"
)

// DeimosRegisterPlugin implements deimos registry.
type DeimosRegisterPlugin struct {
	ServiceAddress string
	DeimosServers  []string
	UpdateInterval time.Duration

	client *deimos.Client

	// Internal state for tracking registered services.
	servicesLock sync.RWMutex
	services     []string
	metaMap      map[string]string
}

func (p *DeimosRegisterPlugin) Start() error {
	if p.client == nil {
		p.client = deimos.NewClient(p.DeimosServers)
	}

	p.services = make([]string, 0)
	p.metaMap = make(map[string]string)

	// TODO: handle already register
	p.client.Set(context.Background(), phobosDir, "phobos_path", deimos.WithDir())

	if p.UpdateInterval > 0 {
		ticker := time.NewTicker(p.UpdateInterval)
		go func() {
			for range ticker.C {
				// Iterate over all registered services and refresh their TTL.
				p.servicesLock.RLock()
				for _, name := range p.services {
					nodePath := fmt.Sprintf("%s/%s/%s", phobosDir, name, p.ServiceAddress)
					metadata := p.metaMap[name]
					p.servicesLock.RUnlock()

					// We simply re-register the service with a new TTL.
					// The TTL should be longer than the update interval to avoid expiration.
					_, err := p.client.Set(context.Background(), nodePath, metadata, deimos.WithTTL(p.UpdateInterval*2))
					if err != nil {
						log.Errorf("failed to refresh TTL for service node %s: %v", nodePath, err)
					}
				}
			}
		}()
	}

	return nil
}

// TODO:
// // HandleConnAccept handles connections from clients
// func (p *DeimosRegisterPlugin) HandleConnAccept(conn net.Conn) (net.Conn, bool) {
// 	if p.Metrics != nil {
// 		clientMeter := metrics.GetOrRegisterMeter("clientMeter", p.Metrics)
// 		clientMeter.Mark(1)
// 	}
// 	return conn, true
// }

// Register registers a service with deimos.
// It creates a node at <BasePath>/<serviceName>/<ServiceAddress> with the provided metadata.
func (p *DeimosRegisterPlugin) Register(name string, rcvr interface{}, metadata string) (err error) {
	if strings.TrimSpace(name) == "" {
		return errors.New("register service 'name' can't be empty")
	}

	// Initialize the client if it hasn't been started explicitly.
	if p.client == nil {
		p.client = deimos.NewClient(p.DeimosServers)
	}

	// Ensure the service-specific directory (/) exists.
	servicePath := fmt.Sprintf("%s/%s", phobosDir, name)
	// TODO: handle already register
	p.client.Set(context.Background(), servicePath, "", deimos.WithDir())

	// Create the ephemeral service node with a TTL.
	nodePath := fmt.Sprintf("%s/%s/%s", phobosDir, name, p.ServiceAddress)
	_, err = p.client.Set(context.Background(), nodePath, metadata, deimos.WithTTL(p.UpdateInterval*2))
	if err != nil {
		log.Errorf("failed to create service node '%s': %v", nodePath, err)
		return err
	}

	// Store the service info in the plugin's state.
	p.servicesLock.Lock()
	p.services = append(p.services, name)
	p.metaMap[name] = metadata
	p.servicesLock.Unlock()

	return nil
}
