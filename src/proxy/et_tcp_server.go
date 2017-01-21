package proxy

import (
	"fmt"
	"net"
	log "third/seelog"
	"time"
)

type iClientServer interface {
	Start()
}

type clientServer struct {
	bindAddress   string
	proxyAddress  string
	remoteAddress string
	l             *net.TCPListener
}

type tcpServer struct {
	httpClient iHTTPClient
	tcpProxy   iTCPProxy
}

func NewClientServer(bindAddress string, proxyAddress string, remoteAddress string) (cs iClientServer) {
	cs_impl := &clientServer{
		bindAddress:   bindAddress,
		proxyAddress:  proxyAddress,
		remoteAddress: remoteAddress,
	}
	cs = cs_impl
	return cs
}

func (self *clientServer) Start() {
	tcp_addr, _ := net.ResolveTCPAddr("tcp", self.bindAddress)
	listener, _ := net.ListenTCP("tcp", tcp_addr)
	for {
		tcp_conn, _ := listener.AcceptTCP()
		if tcp_conn != nil {
			ts := newTCPServer(self.proxyAddress, self.remoteAddress, tcp_conn)
			if ts != nil {
				log.Infof("new tcp server, %s", ts.String())
			} else {
				tcp_conn.Close()
			}
		} else {
			break
		}
	}
}

func newTCPServer(host string, dest string, conn *net.TCPConn) (ts *tcpServer) {
	ts = &tcpServer{
		httpClient: newHTTPClient(host, dest),
		tcpProxy:   newTCPProxy(conn),
	}
	go ts.sendLoop()
	go ts.recvLoop()
	go ts.checkLoop()
	return ts
}

func (self *tcpServer) destroy() {
	log.Infof("%s", self.String())
	self.httpClient.destroy()
	self.tcpProxy.destroy()
}

func (self *tcpServer) sendLoop() {
	for self.tcpProxy.isAlive() {
		dn := self.tcpProxy.popData(true)
		if dn != nil {
			self.httpClient.pushTCPRequest(dn)
		}
	}
}

func (self *tcpServer) recvLoop() {
	for self.httpClient.isAlive() {
		dn := self.httpClient.popTCPResponse()
		if dn != nil {
			self.tcpProxy.pushData(dn)
		}
	}
}

func (self *tcpServer) checkLoop() {
	timer := time.NewTicker(time.Second)
	for _ = range timer.C {
		if !self.httpClient.isAlive() {
			log.Infof("http client not alive, will destroy, %s", self.String())
			self.destroy()
			break
		}
		if !self.tcpProxy.isAlive() {
			log.Infof("tcp proxy not alive, will destroy, %s", self.String())
			self.destroy()
			break
		}
	}
}

func (self *tcpServer) String() string {
	return fmt.Sprintf("httpClient:{%s} tcpProxy:{%s}",
		self.httpClient.String(), self.tcpProxy.String())
}
