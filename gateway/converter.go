package gateway

import (
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/marsevilspirit/m_RPC/protocol"
)

const (
	GatewayVersion           = "MRPC-Gateway-Version"
	GatewayMessageType       = "MRPC-Gateway-MessageType"
	GatewayHeartbeat         = "MRPC-Gateway-Heartbeat"
	GatewayOneway            = "MRPC-Gateway-Oneway"
	GatewayMessageStatusType = "MRPC-Gateway-MessageStatusType"
	GatewaySerializeType     = "MRPC-Gateway-SerializeType"
	GatewayMessageID         = "MRPC-Gateway-MessageID"
	GatewayServicePath       = "MRPC-Gateway-ServicePath"
	GatewayServiceMethod     = "MRPC-Gateway-ServiceMethod"
	GatewayMeta              = "MRPC-Gateway-Meta"
	GatewayErrorMessage      = "MRPC-Gateway-ErrorMessage"
)

func HttpRequest2MRPCRequest(r *http.Request) (*protocol.Message, error) {
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
