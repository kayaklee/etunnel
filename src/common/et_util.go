package common

import (
	"time"
)

func GetCurrentTime() int64 {
	return time.Now().UnixNano() / 1000
}

func upper_align(v uint64, align_size uint64) uint64 {
	return (v + align_size - 1) & (^(align_size - 1))
}
