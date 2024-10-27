package protocol

import (
	"bytes"
	"testing"
)

func TestHeader_CheckMagicNumber(t *testing.T) {
	header := Header([12]byte{})
	header[0] = magicNumber
	if !header.CheckMagicNumber() {
		t.Errorf("expected magic number %x, got %x", magicNumber, header[0])
	}
}

func TestHeader_Version(t *testing.T) {
	header := Header([12]byte{})
	version := byte(1)
	header.SetVersion(version)
	if header.Version() != version {
		t.Errorf("expected version %d, got %d", version, header.Version())
	}
}

func TestHeader_Heartbeat(t *testing.T) {
	header := Header([12]byte{})

	header.SetHeartbeat(true)
	if !header.IsHeartbeat() {
		t.Errorf("expected Heartbeat true, got false")
	}

	header.SetHeartbeat(false)
	if header.IsHeartbeat() {
		t.Errorf("expected Heartbeat false, got true")
	}
}

func TestHeader_Oneway(t *testing.T) {
	header := Header([12]byte{})

	header.SetOneway(true)
	if !header.IsOneway() {
		t.Errorf("expected Oneway true, got false")
	}

	header.SetOneway(false)
	if header.IsOneway() {
		t.Errorf("expected Oneway false, got true")
	}
}

func TestHeader_CompressType(t *testing.T) {
	header := Header([12]byte{})

	header.SetCompressType(None)
	if header.CompressType() != None {
		t.Errorf("expected compress %d, got %d", None, header.CompressType())
	}

	header.SetCompressType(Gzip)
	if header.CompressType() != Gzip {
		t.Errorf("expected compress %d, got %d", Gzip, header.CompressType())
	}
}

func TestHeader_SerializeType(t *testing.T) {
	header1 := Header([12]byte{})

	header1.SetSerializeType(JSON)
	if header1.SerializeType() != JSON {
		t.Errorf("expected serialize %d, got %d", JSON, header1.SerializeType())
	}

	header2 := Header([12]byte{})

	header2.SetSerializeType(ProtoBuffer)
	if header2.SerializeType() != ProtoBuffer {
		t.Errorf("expected serialize %d, got %d", ProtoBuffer, header2.SerializeType())
	}
}

func TestHeader_Seq(t *testing.T) {
	header := Header([12]byte{})

	seq := uint64(114514)
	header.SetSeq(seq)
	if header.Seq() != seq {
		t.Errorf("expected seq %d, got %d", seq, header.Seq())
	}
}

func TestMessage_EncodeDecode(t *testing.T) {
	req := NewMessage()
	req.SetVersion(0)
	req.SetMessageType(Request)
	req.SetHeartbeat(false)
	req.SetOneway(false)
	req.SetCompressType(None)
	req.SetMessageStatusType(Normal)
	req.SetSerializeType(JSON)

	req.SetSeq(114514)

	m := make(map[string]string)
	m["__METHOD"] = "Mrpc.Test"
	m["__ID"] = "41235123613476347134623"
	req.Metadata = m

	payload := `{
		"A": 1,
		"B": 2,
	}`

	req.Payload = []byte(payload)

	var buf bytes.Buffer
	_, err := req.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	res.SetMessageType(Response)

	if !res.CheckMagicNumber() {
		t.Errorf("expect MagicNumber true, got flase")
	}

	if res.Version() != 0 {
		t.Errorf("expect Version 0, got %d", res.Version())
	}

	if res.Seq() != 114514 {
		t.Errorf("expect Seq 114514, got %d", res.Seq())
	}

	if res.Metadata["__METHOD"] != "Mrpc.Test" || res.Metadata["__ID"] != "41235123613476347134623" {
		t.Errorf("got wrong metadata: %v", res.Metadata)
	}

	if string(res.Payload) != payload {
		t.Errorf("got wrong payload: %v", string(res.Payload))
	}
}
