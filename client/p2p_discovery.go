package client

type p2pDiscovery struct {
	server   string
	metadata string
}

func NewP2PDiscovery(server, metadata string) ServiceDiscovery {
	return &p2pDiscovery{server: server, metadata: metadata}
}

func (d p2pDiscovery) GetServices() []*KVPair {
	return []*KVPair{&KVPair{Key: d.server, Value: d.metadata}}
}

func (d p2pDiscovery) WatchService() chan []*KVPair {
	return nil
}
