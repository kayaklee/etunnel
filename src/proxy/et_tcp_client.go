package proxy

import (
	"common"
	"fmt"
	"net"
	"sync"
	log "third/seelog"
	"time"
)

const (
	TCPClientAliveExpiredSec int64 = 100
)

type iTCPClientMgrCallback interface {
	onDestroy()
}

type iTCPClient interface {
	destroy()
	pushHTTPRequest(seq_number int64, hr *httpRequest) (err error)
	keepAlive()
	isAlive(expire_time_sec int64) bool
}

type tcpClient struct {
	addr               string
	mgrCallback        iTCPClientMgrCallback
	lock               sync.Mutex
	seqNumber          int64
	keeyAliveTimestamp int64
	tcpConn            *net.TCPConn
	tcpProxy           iTCPProxy
	reqQueue           chan *httpRequest
	checkTimer         *time.Ticker
}

func newTCPClient(addr string, mgr_callback iTCPClientMgrCallback) iTCPClient {
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
			addr:               addr,
			mgrCallback:        mgr_callback,
			seqNumber:          0,
			keeyAliveTimestamp: common.GetCurrentTime(),
			tcpConn:            tcp_conn,
			tcpProxy:           tcp_proxy,
			reqQueue:           make(chan *httpRequest, DataQueueSize),
			checkTimer:         time.NewTicker(time.Second),
		}
		go tc_impl.processLoop()
		go tc_impl.checkLoop()
	}
	if tc_impl == nil {
		if tcp_proxy != nil {
			tcp_proxy.destroy()
		}
		if tcp_conn != nil {
			tcp_conn.Close()
		}
	}
	return tc_impl
}

func (self *tcpClient) destroy() {
	self.mgrCallback.onDestroy()
	self.checkTimer.Stop()
	close(self.reqQueue)
	self.tcpConn.Close()
	self.tcpProxy.destroy()
}

func (self *tcpClient) pushHTTPRequest(seq_number int64, hr *httpRequest) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.seqNumber+1 != seq_number {
		log.Warnf("invalid seq number, self=%d param=%d", self.seqNumber, seq_number)
		err = fmt.Errorf("invalid seq number, self=%d param=%d", self.seqNumber, seq_number)
	} else {
		hr.wg.Add(1)
		self.seqNumber += 1
		self.reqQueue <- hr
	}
	return err
}

func (self *tcpClient) keepAlive() {
	self.keeyAliveTimestamp = common.GetCurrentTime()
}

func (self *tcpClient) isAlive(expire_time_sec int64) bool {
	bret := false
	if common.GetCurrentTime()-self.keeyAliveTimestamp <= expire_time_sec*1000000 &&
		self.tcpProxy.isConnAlive() {
		bret = true
	}
	return bret
}

func (self *tcpClient) processLoop() {
	for req := range self.reqQueue {
		dn := req.httpService.popData()
		if dn != nil {
			self.tcpProxy.pushData(dn)
		} else {
			blocked := true
			dn := self.tcpProxy.popData(blocked)
			if dn == nil {
				req.httpService.setErrorHappened()
			} else {
				req.httpService.pushData(dn)
				for {
					blocked = false
					dn := self.tcpProxy.popData(blocked)
					req.httpService.pushData(dn)
					if dn == nil {
						break
					}
				}
			}
		}
		req.wg.Done()
	}
}

func (self *tcpClient) checkLoop() {
	for _ = range self.checkTimer.C {
		if !self.isAlive(TCPClientAliveExpiredSec) {
			log.Infof("tcp client not alive, will destroy, addr=[%s]", self.addr)
			self.destroy()
			break
		}
	}
}
