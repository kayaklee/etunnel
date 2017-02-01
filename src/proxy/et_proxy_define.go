package proxy

import (
	"math"
)

const (
	QK_CONN_KEY = "c"
	QK_ADDR     = "a"
	QK_SEQ      = "s"

	QP_DATA    = "d"
	QP_CONNECT = "c"
)

const (
	DataBlockSize int64 = math.MaxUint16
	DataQueueSize int64 = 100
)
