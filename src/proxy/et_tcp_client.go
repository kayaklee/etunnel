package proxy

import (
	"common"
	"fmt"
	"net"
	"sync"
	log "third/seelog"
	"time"
)

type iTCPClientMgrCallback interface {
	getConnKey() string
	onDestroy()
}

type iTCPClient interface {
	destroy()
	pushHTTPRequest(seq_number int64, hr *httpRequest) (err error)
	String() string
}

type tcpClient struct {
	mgrCallback        iTCPClientMgrCallback
	lock               sync.Mutex
	seqNumber          int64
	keeyAliveTimestamp int64
	tcpConn            *net.TCPConn
	tcpProxy           iTCPProxy
	reqQueue           chan *httpRequest
}

func newTCPClient(addr string, mgr_callback iTCPClientMgrCallback) (tc iTCPClient) {
	var err error
	var tcp_addr *net.TCPAddr
	var tcp_conn *net.TCPConn
	var tcp_proxy iTCPProxy
	var tc_impl *tcpClient
	if tcp_addr, err = net.ResolveTCPAddr("tcp", addr); err != nil {
		log.Warnf("ResolveTCPAddr fail, err=[%v] addr=[%s]", err, addr)
	} else if tcp_conn, err = net.DialTCP("tcp", nil, tcp_addr); err != nil {
		log.Warnf("DialTCP fail, err=[%v] addr=[%s]", err, addr)
	} else if tcp_proxy = newTCPProxy(tcp_conn); tcp_proxy == nil {
		log.Warnf("newTCPProxy fail, err=[%v] addr=[%s]", err, addr)
	} else {
		tc_impl = &tcpClient{
			mgrCallback:        mgr_callback,
			seqNumber:          0,
			keeyAliveTimestamp: common.GetCurrentTime(),
			tcpConn:            tcp_conn,
			tcpProxy:           tcp_proxy,
			reqQueue:           make(chan *httpRequest, DataQueueSize),
		}
		go tc_impl.processLoop()
		go tc_impl.checkLoop()
		tc = tc_impl
	}
	if tc_impl == nil {
		if tcp_proxy != nil {
			tcp_proxy.destroy()
		}
		if tcp_conn != nil {
			tcp_conn.Close()
		}
	}
	return tc
}

func (self *tcpClient) destroy() {
	log.Infof("%s", self.String())
	self.mgrCallback.onDestroy()
	close(self.reqQueue)
	self.tcpConn.Close()
	self.tcpProxy.destroy()
}

func (self *tcpClient) pushHTTPRequest(seq_number int64, hr *httpRequest) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.seqNumber+1 != seq_number {
		log.Warnf("invalid seq number, current=%d input=%d", self.seqNumber, seq_number)
		err = fmt.Errorf("invalid seq number, current=%d input=%d", self.seqNumber, seq_number)
	} else {
		hr.wg.Add(1)
		self.seqNumber += 1
		self.keeyAliveTimestamp = common.GetCurrentTime()
		self.reqQueue <- hr
	}
	return err
}

func (self *tcpClient) isAlive(expire_time_sec int64) bool {
	bret := false
	if common.GetCurrentTime()-self.keeyAliveTimestamp <= expire_time_sec*1000000 {
		bret = true
	}
	return bret
}

func (self *tcpClient) processLoop() {
	for req := range self.reqQueue {
		for {
			dn := req.httpWrapper.popData()
			if dn != nil {
				self.tcpProxy.pushData(dn)
			} else {
				break
			}
		}

		blocked := true
		dn := self.tcpProxy.popData(blocked)
		if dn == nil {
			log.Infof("setErrorHappened")
			req.httpWrapper.setErrorHappened()
		} else {
			req.httpWrapper.pushData(dn)
			for {
				blocked = false
				dn := self.tcpProxy.popData(blocked)
				req.httpWrapper.pushData(dn)
				if dn == nil {
					break
				}
			}
		}
		req.wg.Done()
	}
}

func (self *tcpClient) checkLoop() {
	timer := time.NewTicker(time.Second)
	for _ = range timer.C {
		if !self.isAlive(common.G.Server.ConnectionTimeoutSec) {
			log.Infof("tcp client not alive, will destroy, addr=[%s] conn_key=[%s]",
				self.tcpConn.RemoteAddr().String(), self.mgrCallback.getConnKey())
			self.destroy()
			break
		}
	}
}

func (self *tcpClient) String() string {
	return fmt.Sprintf("this=%p seq=%d aliveTimestamp=%d %s",
		self, self.seqNumber, self.keeyAliveTimestamp, self.tcpProxy.String())
}
