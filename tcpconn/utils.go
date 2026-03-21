package tcpconn

import "sync/atomic"

var nextClientID int64

func getNextClientID() int64 {
	return atomic.AddInt64(&nextClientID, 1)
}
