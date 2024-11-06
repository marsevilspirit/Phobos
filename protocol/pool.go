package protocol

import "sync"

var poolUint32Data = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4)
	},
}
