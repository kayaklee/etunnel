package proxy

import (
	"net"
	log "third/seelog"
)

const (
	DataBlockSize int64 = 65536
	DataQueueSize int64 = 100
)

type dataBlock struct {
	data []byte
}

type iTCPProxy interface {
	destroy()
	isConnAlive() bool
	pushData(dn *dataBlock)
	popData(block bool) *dataBlock
}

type tcpProxy struct {
	conn      *net.TCPConn
	sendQ     chan *dataBlock
	recvQ     chan *dataBlock
	connAlive bool
}

func newTCPProxy(conn *net.TCPConn) iTCPProxy {
	tp_impl := &tcpProxy{
		conn:      conn,
		sendQ:     make(chan *dataBlock, DataQueueSize),
		recvQ:     make(chan *dataBlock, DataQueueSize),
		connAlive: true,
	}
	go tp_impl.sendLoop()
	go tp_impl.recvLoop()
	return tp_impl
}

func (self *tcpProxy) destroy() {
	close(self.sendQ)
	close(self.recvQ)
}

func (self *tcpProxy) isConnAlive() bool {
	return self.connAlive
}

func (self *tcpProxy) sendLoop() {
	for dn := range self.sendQ {
		write_ret, err := self.conn.Write(dn.data)
		if write_ret != len(dn.data) ||
			err != nil {
			log.Warnf("write fail, write_ret=%d err=[%v]", write_ret, err)
			self.connAlive = false
			break
		}
	}
}

func (self *tcpProxy) recvLoop() {
	for {
		dn := &dataBlock{
			data: make([]byte, DataBlockSize),
		}
		read_ret, err := self.conn.Read(dn.data)
		if err != nil {
			log.Warnf("read fail, read_ret=%d err=[%v]", read_ret, err)
			self.connAlive = false
			break
		}
		dn.data = dn.data[:read_ret]
		self.recvQ <- dn
	}
}

func (self *tcpProxy) pushData(dn *dataBlock) {
	self.sendQ <- dn
}

func (self *tcpProxy) popData(block bool) (dn *dataBlock) {
	if block {
		dn = <-self.recvQ
	} else {
		select {
		case dn = <-self.recvQ:
		default:
		}
	}
	return dn
}
