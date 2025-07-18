package gateway

import (
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/marsevilspirit/phobos/protocol"
)

const (
	GatewayVersion           = "PHOBOS-Gateway-Version"
	GatewayMessageType       = "PHOBOS-Gateway-MessageType"
	GatewayHeartbeat         = "PHOBOS-Gateway-Heartbeat"
	GatewayOneway            = "PHOBOS-Gateway-Oneway"
	GatewayMessageStatusType = "PHOBOS-Gateway-MessageStatusType"
	GatewaySerializeType     = "PHOBOS-Gateway-SerializeType"
	GatewayMessageID         = "PHOBOS-Gateway-MessageID"
	GatewayServicePath       = "PHOBOS-Gateway-ServicePath"
	GatewayServiceMethod     = "PHOBOS-Gateway-ServiceMethod"
	GatewayMeta              = "PHOBOS-Gateway-Meta"
	GatewayErrorMessage      = "PHOBOS-Gateway-ErrorMessage"
)

func HttpRequest2PHOBOSRequest(r *http.Request) (*protocol.Message, error) {
	req := protocol.NewMessage()
	req.SetMessageType(protocol.Request)

	h := r.Header
	seq := h.Get(GatewayMessageID)
	if seq == "" {
		id, err := strconv.ParseUint(seq, 10, 64)
		if err != nil {
			return nil, err
		}
		req.SetSeq(id)
	}

	heartbeat := h.Get(GatewayHeartbeat)
	if heartbeat != "" {
		req.SetHeartbeat(true)
	}

	oneway := h.Get(GatewayOneway)
	if oneway != "" {
		req.SetOneway(true)
	}

	if h.Get("Content-Encoding") == "gzip" {
		req.SetCompressType(protocol.Gzip)
	}

	st := h.Get(GatewaySerializeType)
	if st != "" {
		rst, err := strconv.Atoi(st)
		if err != nil {
			return nil, err
		}
		req.SetSerializeType(protocol.SerializeType(rst))
	}

	meta := h.Get(GatewayMeta)
	if meta != "" {
		metadata, err := url.ParseQuery(meta)
		if err != nil {
			return nil, err
		}
		mm := make(map[string]string)
		for k, v := range metadata {
			if len(v) > 0 {
				mm[k] = v[0]
			}
		}
		req.Metadata = mm
	}

	sp := h.Get(GatewayServicePath)
	if sp != "" {
		req.ServicePath = sp
	}

	sm := h.Get(GatewayServiceMethod)
	if sm != "" {
		req.ServiceMethod = sm
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	req.Payload = payload

	return req, nil
}
