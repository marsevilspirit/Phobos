package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/marsevilspirit/m_RPC/util"
)

const (
	magicNumber byte = 0x42
)

var (
	lineSeparator = []byte("\r\n")
)

var (
	ErrMetaKVMissing = errors.New("wrong metadata lines. some keys or values are missing")
)

type MessageType byte

// MessageType只有两种
// Request 和 Response
const (
	Request MessageType = iota
	Response
)

type MessageStatusType byte

const (
	Normal MessageStatusType = iota
	Error
)

type CompressType byte

const (
	None CompressType = iota
	Gzip
	// TODO: etc...
)

type SerializeType byte

const (
	SerializeNone SerializeType = iota
	JSON
	ProtoBuffer
	MsgPack
)

type Message struct {
	*Header
	Metadata map[string]string // map[string]string[]???
	Payload  []byte
}

func NewMessage() *Message {
	header := Header([12]byte{})
	header[0] = magicNumber
	return &Message{
		Header:   &header,
		Metadata: make(map[string]string),
	}
}

// Header:
// +---------------------------------------------+
// |magicNumber|version|h[2]|SerializeType|Seq...|
// +---------------------------------------------+
//
// h[2]的第8位是MessageType, 第7位是IsHeartbeat, 第6位是IsOneway
// 第5-3位是CompressType
type Header [12]byte

func (h Header) CheckMagicNumber() bool {
	return h[0] == magicNumber
}

func (h *Header) Version() byte {
	return h[1]
}

func (h *Header) SetVersion(v byte) {
	h[1] = v
}

// 0x80在二进制中是1000 0000
// 用 & 提取第三字节的最高位
func (h Header) MessageType() MessageType {
	return MessageType(h[2] & 0x80)
}

func (h *Header) SetMessageType(mt MessageType) {
	h[2] = h[2] | (byte(mt) << 7)
}

func (h Header) IsHeartbeat() bool {
	return h[2]&0x40 == 0x40
}

func (h *Header) SetHeartbeat(hb bool) {
	if hb {
		h[2] = h[2] | 0x40
	} else {
		h[2] = h[2] &^ 0x40
	}
}

func (h Header) IsOneway() bool {
	return h[2]&0x20 == 0x20
}

func (h *Header) SetOneway(oneway bool) {
	if oneway {
		h[2] = h[2] | 0x20
	} else {
		h[2] = h[2] &^ 0x20
	}
}

func (h Header) CompressType() CompressType {
	return CompressType((h[2] & 0x1C) >> 2)
}

func (h *Header) SetCompressType(ct CompressType) {
	h[2] = h[2] | ((byte(ct) << 2) & 0x1C)
}

func (h Header) MessageStatusType() MessageStatusType {
	return MessageStatusType(h[2] & 0x03)
}

func (h *Header) SetMessageStatusType(mt MessageStatusType) {
	h[2] = h[2] | (byte(mt) & 0x03)
}

// 0xF0 是 1111 0000
func (h Header) SerializeType() SerializeType {
	return SerializeType((h[3] & 0xF0) >> 4)
}

func (h *Header) SetSerializeType(st SerializeType) {
	h[3] = h[3] | (byte(st) << 4)
}

func (h Header) Seq() uint64 {
	return binary.BigEndian.Uint64(h[4:])
}

func (h *Header) SetSeq(seq uint64) {
	binary.BigEndian.PutUint64(h[4:], seq)
}

func (m Message) Clone() *Message {
	header := *m.Header
	c := &Message{
		Header:   &header,
		Metadata: make(map[string]string),
	}
	return c
}

func (m Message) Encode() []byte {
	meta := encodeMetadata(m.Metadata)

	// 4 指明长度
	l := 12 + (4 + len(meta)) + (4 + len(m.Payload))

	data := make([]byte, l)
	copy(data, m.Header[:])
	binary.BigEndian.PutUint32(data[12:16], uint32(len(meta)))
	copy(data[12:], meta)
	binary.BigEndian.PutUint32(data[16+len(meta):], uint32(len(m.Payload)))
	copy(data[20+len(meta):], m.Payload)

	return data
}

func (m Message) WriteTo(w io.Writer) (int64, error) {
	var bytes int64

	n, err := w.Write(m.Header[:])
	if err != nil {
		return bytes, err
	}

	bytes = int64(n)

	meta := encodeMetadata(m.Metadata)
	err = binary.Write(w, binary.BigEndian, uint32(len(meta)))
	if err != nil {
		return bytes, err
	}

	bytes += int64(binary.Size(uint32(len(meta))))

	n, err = w.Write(meta)
	if err != nil {
		return bytes, err
	}

	bytes += int64(n)

	err = binary.Write(w, binary.BigEndian, uint32(len(m.Payload)))
	if err != nil {
		return bytes, err
	}

	bytes += int64(binary.Size(uint32(len(m.Payload))))

	n, err = w.Write(m.Payload)
	if err != nil {
		return bytes, err
	}

	bytes += int64(n)

	return bytes, err
}

func encodeMetadata(m map[string]string) []byte {
	var buf bytes.Buffer
	for k, v := range m {
		buf.WriteString(k)
		buf.Write(lineSeparator)
		buf.WriteString(v)
		buf.Write(lineSeparator)
	}

	return buf.Bytes()
}

func decodeMetadata(lenData []byte, r io.Reader) (map[string]string, error) {
	_, err := io.ReadFull(r, lenData)
	if err != nil {
		return nil, err
	}

	l := binary.BigEndian.Uint32(lenData)
	m := make(map[string]string)
	if l == 0 {
		return m, nil
	}

	data := make([]byte, l)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}

	meta := bytes.Split(data, lineSeparator)
	if len(meta)%2 != 1 {
		return nil, ErrMetaKVMissing
	}

	for i := 0; i < len(meta)-1; i = i + 2 {
		m[util.SliceByteToString(meta[i])] = util.SliceByteToString(meta[i+1])
	}

	return m, nil
}

func Read(r io.Reader) (*Message, error) {
	msg := NewMessage()
	_, err := io.ReadFull(r, msg.Header[:])
	if err != nil {
		return nil, err
	}

	lenData := make([]byte, 4)
	msg.Metadata, err = decodeMetadata(lenData, r)
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(r, lenData)
	if err != nil {
		return nil, err
	}
	l := binary.BigEndian.Uint32(lenData)

	msg.Payload = make([]byte, l)

	_, err = io.ReadFull(r, msg.Payload)

	return msg, err
}
