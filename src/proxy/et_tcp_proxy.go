package proxy

import (
	"fmt"
	"net"
	log "third/seelog"
)

type dataBlock struct {
	data []byte
}

type iTCPProxy interface {
	destroy()
	isAlive() bool
	pushData(dn *dataBlock)
	popData(block bool) *dataBlock
	String() string
}

type tcpProxy struct {
	conn      *net.TCPConn
	sendQ     chan *dataBlock
	recvQ     chan *dataBlock
	connAlive bool
}

func newTCPProxy(conn *net.TCPConn) (tp iTCPProxy) {
	tp_impl := &tcpProxy{
		conn:      conn,
		sendQ:     make(chan *dataBlock, DataQueueSize),
		recvQ:     make(chan *dataBlock, DataQueueSize),
		connAlive: true,
	}
	go tp_impl.sendLoop()
	go tp_impl.recvLoop()
	tp = tp_impl
	return tp
}

func (self *tcpProxy) destroy() {
	log.Infof("%s", self.String())
	self.conn.Close()
	close(self.sendQ)
	close(self.recvQ)
}

func (self *tcpProxy) isAlive() bool {
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
		} else {
			log.Debugf("send data succ, len=%d %s", write_ret, self.String())
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
		} else {
			log.Debugf("recv data succ, len=%d %s", read_ret, self.String())
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

func (self *tcpProxy) String() string {
	f, _ := self.conn.File()
	return fmt.Sprintf("this=%p fd=%v remote=[%s] local=[%s] alive=%t",
		self, f.Fd(), self.conn.RemoteAddr().String(), self.conn.LocalAddr().String(), self.connAlive)
}
