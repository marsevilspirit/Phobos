package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/julienschmidt/httprouter"
	"github.com/marsevilspirit/phobos/client"
)

type Gateway struct {
	Addr string

	serviceDiscovery client.ServiceDiscovery
	FailMode         client.FailMode
	SelectMode       client.SelectMode
	Option           client.Option

	mu       sync.RWMutex
	xclients map[string]client.XClient
}

func NewGateway(addr string, serviceDiscovery client.ServiceDiscovery, failMode client.FailMode, selectMode client.SelectMode, option client.Option) *Gateway {
	return &Gateway{
		Addr:             addr,
		serviceDiscovery: serviceDiscovery,
		FailMode:         failMode,
		SelectMode:       selectMode,
		Option:           option,
		xclients:         make(map[string]client.XClient),
	}
}

func (g *Gateway) Serve() {
	router := httprouter.New()
	router.POST("/*servicePath", g.handleRequest)
	router.GET("/*servicePath", g.handleRequest)
	router.PUT("/*servicePath", g.handleRequest)

	if err := http.ListenAndServe(g.Addr, router); err != nil {
		log.Fatalf("Gateway listen error: %s", err)
	}
}

func (g *Gateway) handleRequest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if r.Header.Get(GatewayServicePath) == "" {
		servicePath := ps.ByName("servicePath")
		if strings.HasPrefix(servicePath, "/") {
			servicePath = servicePath[1:]
		}

		r.Header.Set(GatewayServicePath, servicePath)
	}

	servicePath := r.Header.Get(GatewayServicePath)

	wh := w.Header()
	req, err := HttpRequest2MRPCRequest(r)
	if err != nil {
		rh := r.Header
		for k, v := range rh {
			if strings.HasPrefix(k, "MRPC-Gateway-") && len(v) > 0 {
				wh.Set(k, v[0])
			}
		}

		wh.Set(GatewayMessageStatusType, "Error")
		wh.Set(GatewayErrorMessage, err.Error())
		return
	}

	var xc client.XClient
	g.mu.Lock()
	if g.xclients[servicePath] == nil {
		g.xclients[servicePath] = client.NewXClient(servicePath, g.FailMode, g.SelectMode, g.serviceDiscovery.Clone(servicePath), g.Option)
	}
	xc = g.xclients[servicePath]
	g.mu.Unlock()

	fmt.Printf("@@@@@: %+v\n", req)
	m, payload, err := xc.SendRaw(context.Background(), req)
	for k, v := range m {
		wh.Set(k, v)
	}
	if err != nil {
		wh.Set(GatewayMessageStatusType, "Error")
		wh.Set(GatewayErrorMessage, err.Error())
		return
	}

	w.Write(payload)
}
