package proxy

import (
	"net/http"
	"strconv"
	"sync"
	log "third/seelog"
)

type httpRequest struct {
	httpService iHTTPService
	wg          sync.WaitGroup
}

type proxyServer struct {
	lock         sync.RWMutex
	tcpClientMgr map[string]iTCPClient
}

type tcpClientMgrCallback struct {
	proxyServer *proxyServer
	connKey     string
}

func NewProxyServer() *proxyServer {
	ps := &proxyServer{
		tcpClientMgr: make(map[string]iTCPClient),
	}
	return ps
}

func (self *proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn_key := r.URL.Query().Get("conn")
	tcp_client := self.getTCPClient(conn_key)
	if tcp_client == nil {
		addr := r.URL.Query().Get("addr")
		tcp_client = newTCPClient(addr, &tcpClientMgrCallback{self, conn_key})
		if tcp_client != nil {
			log.Infof("newTCPClient succ, client=[%s] addr=[%s] conn_key=[%s]", r.RemoteAddr, addr, conn_key)
			self.addTCPClient(conn_key, tcp_client)
		}
	}

	hr := &httpRequest{
		httpService: newHTTPService(r, w),
	}
	if tcp_client == nil {
		hr.httpService.setErrorHappened()
	} else {
		seq_number, _ := strconv.ParseInt(r.URL.Query().Get("seq"), 10, 64)
		tcp_client.pushHTTPRequest(seq_number, hr)
		hr.wg.Wait()
	}
}

func (self *proxyServer) deleteTCPClient(conn_key string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.tcpClientMgr, conn_key)
}

func (self *proxyServer) addTCPClient(conn_key string, tcp_client iTCPClient) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.tcpClientMgr[conn_key] = tcp_client
}

func (self *proxyServer) getTCPClient(conn_key string) (tcp_client iTCPClient) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	tcp_client = self.tcpClientMgr[conn_key]
	return tcp_client
}

func (self *tcpClientMgrCallback) onDestroy() {
	self.proxyServer.deleteTCPClient(self.connKey)
}

func (self *tcpClientMgrCallback) getConnKey() string {
	return self.connKey
}
