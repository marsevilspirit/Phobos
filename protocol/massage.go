package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/marsevilspirit/m_RPC/util"
)

// MaxMessageLength is the max length of a message.
// Default is 0 that means does not limit length of messages.
// It is used to validate when read messages from io.Reader.
var MaxMessageLength = 0

const (
	magicNumber byte = 0x42
)

var (
	lineSeparator = []byte("\r\n")
)

var (
	ErrMetaKVMissing = errors.New("wrong metadata lines. some keys or values are missing")
	ErrMessageToLong = errors.New("message is too long")
)

const (
	ServiceError = "__mrpc_error__"
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
	ServicePath   string
	ServiceMethod string
	Metadata      map[string]string // map[string]string[]???
	Payload       []byte
	data          []byte
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

	c := GetPoolMsg()
	c.Header = &header
	c.ServicePath = m.ServicePath
	c.ServiceMethod = m.ServiceMethod

	return c
}

func (m Message) Encode() []byte {
	meta := encodeMetadata(m.Metadata)

	ServicePathLength := len(m.ServicePath)
	ServiceMethodLength := len(m.ServiceMethod)

	totalLength := (4 + ServicePathLength) + (4 + ServiceMethodLength) +
		(4 + len(meta)) + (4 + len(m.Payload))

	// header + dataLen + ServicePathLength + ServicePath + ServiceMethodLength + ServiceMethod + metaLen + meta payloadLen + payload
	metaStart := 12 + 4 + (4 + ServicePathLength) + (4 + ServiceMethodLength)

	payloadStart := metaStart + (4 + len(meta))
	l := 12 + 4 + totalLength

	data := make([]byte, l)
	copy(data, m.Header[:])

	// totalLength
	binary.BigEndian.PutUint32(data[12:16], uint32(totalLength))

	binary.BigEndian.PutUint32(data[16:20], uint32(ServicePathLength))
	copy(data[20:20+ServicePathLength], util.StringToSliceByte(m.ServicePath))

	binary.BigEndian.PutUint32(data[20+ServicePathLength:24+ServicePathLength], uint32(ServiceMethodLength))
	copy(data[24+ServicePathLength:metaStart], util.StringToSliceByte(m.ServiceMethod))

	binary.BigEndian.PutUint32(data[metaStart:metaStart+4], uint32(len(meta)))
	copy(data[metaStart+4:metaStart+4+len(meta)], meta)

	binary.BigEndian.PutUint32(data[payloadStart:payloadStart+4], uint32(len(m.Payload)))
	copy(data[payloadStart+4:], m.Payload)

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

	ServicePathLength := len(m.ServicePath)
	ServiceMethodLength := len(m.ServiceMethod)

	// totalLength
	totalLength := (4 + ServicePathLength) + (4 + ServiceMethodLength) + (4 + len(meta)) + (4 + len(m.Payload))
	err = binary.Write(w, binary.BigEndian, uint32(totalLength))
	if err != nil {
		return bytes, err
	}

	// ServicePath
	err = binary.Write(w, binary.BigEndian, uint32(len(m.ServicePath)))
	if err != nil {
		return bytes, err
	}
	bytes += int64(binary.Size(uint32(len(m.ServicePath))))
	n, err = w.Write(util.StringToSliceByte(m.ServicePath))
	if err != nil {
		return bytes, err
	}
	bytes += int64(n)

	// ServiceMethod
	err = binary.Write(w, binary.BigEndian, uint32(len(m.ServiceMethod)))
	if err != nil {
		return bytes, err
	}
	bytes += int64(binary.Size(uint32(len(m.ServiceMethod))))
	n, err = w.Write(util.StringToSliceByte(m.ServiceMethod))
	if err != nil {
		return bytes, err
	}
	bytes += int64(n)

	// metadata
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

	// payload
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

// 编码metadata
func encodeMetadata(m map[string]string) []byte {
	if len(m) == 0 {
		return []byte{}
	}

	var buf bytes.Buffer
	var d = make([]byte, 4)
	for k, v := range m {
		binary.BigEndian.PutUint32(d, uint32(len(k)))
		buf.Write(d)
		buf.Write(util.StringToSliceByte(k))
		binary.BigEndian.PutUint32(d, uint32(len(v)))
		buf.Write(d)
		buf.Write(util.StringToSliceByte(v))
	}

	return buf.Bytes()
}

// 解码metadata
func decodeMetadata(l uint32, data []byte) (map[string]string, error) {
	m := make(map[string]string, 10)
	n := uint32(0)

	for n < l {
		keyLen := binary.BigEndian.Uint32(data[n : n+4])
		n += 4
		if n+keyLen > l-4 {
			return m, ErrMetaKVMissing
		}

		key := util.SliceByteToString(data[n : n+keyLen])
		n += keyLen

		valLen := binary.BigEndian.Uint32(data[n : n+4])
		n += 4
		if n+valLen > l {
			return m, ErrMetaKVMissing
		}

		val := util.SliceByteToString(data[n : n+valLen])
		n += valLen

		m[key] = val
	}

	return m, nil
}

func Read(r io.Reader) (*Message, error) {
	msg := NewMessage()
	err := msg.Decode(r)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (m *Message) Decode(r io.Reader) error {
	// 读取Header
	_, err := io.ReadFull(r, m.Header[:])
	if err != nil {
		return err
	}

	// 读取data长度
	lenData := poolUint32Data.Get().(*[]byte)
	_, err = io.ReadFull(r, *lenData)
	if err != nil {
		poolUint32Data.Put(lenData)
		return err
	}

	l := binary.BigEndian.Uint32(*lenData)
	poolUint32Data.Put(lenData)

	if MaxMessageLength > 0 && int(l) > MaxMessageLength {
		return ErrMessageToLong
	}

	data := make([]byte, int(l))
	_, err = io.ReadFull(r, data)
	if err != nil {
		return err
	}

	m.data = data

	n := 0

	// 读取ServicePath
	l = binary.BigEndian.Uint32(data[n : n+4])
	n += 4
	nEnd := n + int(l)
	m.ServicePath = util.SliceByteToString(data[n:nEnd])
	n = nEnd

	// 读取ServiceMethod
	l = binary.BigEndian.Uint32(data[n : n+4])
	n += 4
	nEnd = n + int(l)
	m.ServiceMethod = util.SliceByteToString(data[n:nEnd])
	n = nEnd

	// 读取metadata
	l = binary.BigEndian.Uint32(data[n : n+4])
	n += 4
	nEnd = n + int(l)
	if l > 0 {
		m.Metadata, err = decodeMetadata(l, data[n:nEnd])
		if err != nil {
			return err
		}
	}
	n = nEnd

	// 读取payload
	l = binary.BigEndian.Uint32(data[n : n+4])
	_ = l
	n += 4
	m.Payload = data[n:]

	return err
}

func (m *Message) Reset() {
	resetHeader(m.Header)

	m.ServicePath = ""
	m.ServiceMethod = ""
	m.Metadata = nil
	m.Payload = m.Payload[:0]
	m.data = m.data[:0]
}

var zeroHeaderArray Header
var zeroHeader = zeroHeaderArray[1:]

func resetHeader(h *Header) {
	copy(h[1:], zeroHeader)
}
