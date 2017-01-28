package proxy

import (
	"fmt"
	"io"
	"net"
	log "third/seelog"
	"time"
)

type dataBlock struct {
	data []byte
}

type iTCPProxy interface {
	destroy()
	isAlive() bool
	pushData(dn *dataBlock)
	popData(time_wait_us int64) *dataBlock
	String() string
}

type tcpProxy struct {
	conn      *net.TCPConn
	sendQ     chan *dataBlock
	sendQsync chan *dataBlock
	recvQ     chan *dataBlock
	connAlive bool
}

func newTCPProxy(conn *net.TCPConn) (tp iTCPProxy) {
	tp_impl := &tcpProxy{
		conn:      conn,
		sendQ:     make(chan *dataBlock, DataQueueSize),
		sendQsync: make(chan *dataBlock, 1),
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
	self.connAlive = false
	self.conn.Close()
	close(self.sendQsync)
	self.recvQ <- nil
}

func (self *tcpProxy) isAlive() bool {
	return (self.connAlive || 0 != len(self.recvQ))
}

func (self *tcpProxy) popFromSendQ() (dn *dataBlock) {
	select {
	case dn = <-self.sendQ:
	case dn = <-self.sendQsync:
	}
	return dn
}

func (self *tcpProxy) sendLoop() {
	for self.isAlive() {
		dn := self.popFromSendQ()
		if dn == nil {
			break
		}
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
	for self.isAlive() {
		dn := &dataBlock{
			data: make([]byte, DataBlockSize),
		}
		read_ret, err := self.conn.Read(dn.data)
		if read_ret > 0 {
			dn.data = dn.data[:read_ret]
			self.recvQ <- dn
		}
		if err != nil {
			if err != io.EOF {
				log.Warnf("read fail, read_ret=%d err=[%v]", read_ret, err)
			} else {
				log.Infof("connection close, read_ret=%d err=[%v]", read_ret, err)
			}
			self.connAlive = false
			break
		} else {
			log.Debugf("recv data succ, len=%d %s", read_ret, self.String())
		}
	}
}

func (self *tcpProxy) pushData(dn *dataBlock) {
	self.sendQ <- dn
}

func (self *tcpProxy) popData(time_wait_us int64) (dn *dataBlock) {
	select {
	case dn = <-self.recvQ:
	default:
	}
	if dn == nil && 0 != time_wait_us && (self.isAlive() || 0 < len(self.recvQ)) {
		if 0 < time_wait_us {
			timer := time.NewTicker((time.Duration)(time_wait_us) * time.Microsecond)
			select {
			case dn = <-self.recvQ:
			case <-timer.C:
			}
		} else {
			dn = <-self.recvQ
		}
	}
	return dn
}

func (self *tcpProxy) String() string {
	return fmt.Sprintf("this=%p remote=[%s] local=[%s] alive=%t sendQLen=%d recvQLen=%d",
		self, self.conn.RemoteAddr().String(), self.conn.LocalAddr().String(), self.connAlive, len(self.sendQ), len(self.recvQ))
}
