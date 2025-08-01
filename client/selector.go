package client

import (
	"context"
	"math"
	"math/rand"
	"net/url"
	"strconv"
	"time"

	"github.com/valyala/fastrand"
)

type Selector interface {
	Select(ctx context.Context, servicePath, serviceMethod string, args any) string
	UpdateServer(servers map[string]string)
}

func newSelector(selectMode SelectMode, servers map[string]string) Selector {
	switch selectMode {
	case RandomSelect:
		return newRandomSelector(servers)
	case RoundRobin:
		return newRoundRobinSelector(servers)
	case WeightedRoundRobin:
		return newWeightRoundRobinSelector(servers)
	case WeightedICMP:
		return newWeightedICMPSelector(servers)
	case ConsistentHash:
		return newConsistentHashSelector(servers)
	case Closest:
		return newConsistentHashSelector(servers)
	case SelectByUser:
		return nil
	default:
		return newRandomSelector(servers)
	}
}

type randomSelector struct {
	servers []string
}

func newRandomSelector(servers map[string]string) Selector {
	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
	}

	return &randomSelector{servers: ss}
}

func (s randomSelector) Select(ctx context.Context, servicePath, serviceMethod string, args any) string {
	ss := s.servers

	if len(ss) == 0 {
		return ""
	}

	i := fastrand.Uint32n(uint32(len(ss)))

	return ss[i]
}

func (s *randomSelector) UpdateServer(servers map[string]string) {
	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
	}

	s.servers = ss
}

type roundRobinSelector struct {
	servers []string
	i       int
}

func newRoundRobinSelector(servers map[string]string) Selector {
	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
	}

	return &roundRobinSelector{servers: ss}
}

func (s *roundRobinSelector) Select(ctx context.Context, servicePath, serviceMethod string, args any) string {
	ss := s.servers

	if len(ss) == 0 {
		return ""
	}

	i := s.i
	i = i % len(ss)
	s.i = i + 1
	return ss[i]
}

func (s *roundRobinSelector) UpdateServer(servers map[string]string) {
	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
	}

	s.servers = ss
}

type weightedRoundRobinSelector struct {
	servers []*Weighted
}

func newWeightRoundRobinSelector(servers map[string]string) Selector {
	ss := createWeighted(servers)
	return &weightedRoundRobinSelector{servers: ss}
}

func (s *weightedRoundRobinSelector) Select(ctx context.Context, servicePath, serviceMethod string, args any) string {
	ss := s.servers
	if len(ss) == 0 {
		return ""
	}
	w := nextWeighted(ss)
	if w == nil {
		return ""
	}

	return w.Server
}

func (s *weightedRoundRobinSelector) UpdateServer(servers map[string]string) {
	ss := createWeighted(servers)
	s.servers = ss
}

// example:
//
//	servers := map[string]string{
//	    "server1": "weight=2",
//	    "server2": "weight=3&other=value",
//	    "server3": "other=value&weight=5",
//	}
func createWeighted(servers map[string]string) []*Weighted {
	ss := make([]*Weighted, 0, len(servers))
	for k, metadata := range servers {
		w := &Weighted{Server: k, Weight: 1, EffectiveWeight: 1}

		if v, err := url.ParseQuery(metadata); err == nil {
			ww := v.Get("weight")
			if ww != "" {
				if weight, err := strconv.Atoi(ww); err == nil {
					w.Weight = weight
					w.EffectiveWeight = weight
				}
			}
		}

		ss = append(ss, w)
	}

	return ss
}

type geoServer struct {
	Server     string
	Latitude   float64
	Longtitude float64
}

type geoSelector struct {
	servers   []*geoServer
	Latitude  float64
	Longitude float64
	r         *rand.Rand
}

func newGeoSelector(servers map[string]string, latitude, longitude float64) Selector {
	ss := createGeoServer(servers)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &geoSelector{servers: ss, Latitude: latitude, Longitude: longitude, r: r}
}

func (s *geoSelector) Select(ctx context.Context, servicePath, serviceMethod string, args any) string {
	if len(s.servers) == 0 {
		return ""
	}

	var server []string
	min := math.MaxFloat64
	for _, gs := range s.servers {
		d := getDistanceFrom(s.Latitude, s.Longitude, gs.Latitude, gs.Longtitude)
		if d < min {
			server = []string{gs.Server}
			min = d
		} else if d == min {
			server = append(server, gs.Server)
		}
	}

	if len(server) == 1 {
		return server[0]
	}

	return server[s.r.Intn(len(server))]
}

func (s *geoSelector) UpdateServer(servers map[string]string) {
	ss := createGeoServer(servers)
	s.servers = ss
}

func createGeoServer(servers map[string]string) []*geoServer {
	geoServers := make([]*geoServer, 0, len(servers))

	for s, metadata := range servers {
		if v, err := url.ParseQuery(metadata); err == nil {
			latStr := v.Get("latitude")
			lonStr := v.Get("longitude")

			if latStr == "" || lonStr == "" {
				continue
			}

			lat, err := strconv.ParseFloat(latStr, 64)
			if err != nil {
				continue
			}
			lon, err := strconv.ParseFloat(lonStr, 64)
			if err != nil {
				continue
			}

			geoServers = append(geoServers, &geoServer{Server: s, Latitude: lat, Longtitude: lon})
		}
	}

	return geoServers
}

type consistentHashSelector struct {
	servers []string
}

func newConsistentHashSelector(servers map[string]string) Selector {
	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
	}

	return &consistentHashSelector{servers: ss}
}

func (s *consistentHashSelector) Select(ctx context.Context, servicePath, serviceMethod string, args any) string {
	ss := s.servers

	if len(ss) == 0 {
		return ""
	}

	i := JumpConsistentHash(len(ss), servicePath, serviceMethod, args)
	return ss[i]
}

func (s *consistentHashSelector) UpdateServer(servers map[string]string) {
	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
	}

	s.servers = ss
}

// weightedICMPSelector selects servers with ping result.
type weightedICMPSelector struct {
	servers []*Weighted
}

func newWeightedICMPSelector(servers map[string]string) Selector {
	ss := createICMPWeighted(servers)
	return &weightedICMPSelector{servers: ss}
}

func (s weightedICMPSelector) Select(ctx context.Context, servicePath, serviceMethod string, args any) string {
	ss := s.servers
	if len(ss) == 0 {
		return ""
	}
	w := nextWeighted(ss)
	if w == nil {
		return ""
	}
	return w.Server
}

func (s *weightedICMPSelector) UpdateServer(servers map[string]string) {
	ss := createICMPWeighted(servers)
	s.servers = ss
}
