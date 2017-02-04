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
	popData(time_wait_us int64) *dataBlock // pop entire encrypt block
	String() string
}

type iFilter interface {
	onDataRecv(dn *dataBlock) (*dataBlock, error)
	onDataSend(dn *dataBlock) (*dataBlock, error)
	dataBlockSize() int64
}

type tcpProxy struct {
	conn      *net.TCPConn
	dnFilter  iFilter
	sendQ     chan *dataBlock
	sendQsync chan *dataBlock
	recvQ     chan *dataBlock
	connAlive bool
}

type dummyFilter struct{}

func newTCPProxy(conn *net.TCPConn, dn_filter iFilter) (tp iTCPProxy) {
	tp_impl := &tcpProxy{
		conn:      conn,
		dnFilter:  dn_filter,
		sendQ:     make(chan *dataBlock, DataQueueSize),
		sendQsync: make(chan *dataBlock, 1),
		recvQ:     make(chan *dataBlock, DataQueueSize),
		connAlive: true,
	}
	conn.SetReadBuffer(int(dn_filter.dataBlockSize()))
	conn.SetWriteBuffer(int(dn_filter.dataBlockSize()))
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
			data: make([]byte, self.dnFilter.dataBlockSize()),
		}
		read_ret, err := self.conn.Read(dn.data)
		if read_ret > 0 {
			dn.data = dn.data[:read_ret]
			filtered_dn, tmp_err := self.dnFilter.onDataRecv(dn)
			if filtered_dn != nil && tmp_err == nil {
				self.recvQ <- filtered_dn
			}
			if tmp_err != nil {
				err = tmp_err
			}
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
	filtered_dn, tmp_err := self.dnFilter.onDataSend(dn)
	if filtered_dn != nil && tmp_err == nil {
		self.sendQ <- filtered_dn
	}
	if tmp_err != nil {
		self.connAlive = false
	}
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

func (self *dummyFilter) onDataRecv(dn *dataBlock) (*dataBlock, error) {
	return dn, nil
}

func (self *dummyFilter) onDataSend(dn *dataBlock) (*dataBlock, error) {
	return dn, nil
}

func (self *dummyFilter) dataBlockSize() int64 {
	return DataBlockSize
}
