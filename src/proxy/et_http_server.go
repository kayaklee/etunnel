package proxy

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	log "third/seelog"
)

type httpRequest struct {
	httpWrapper iHTTPWrapper
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
	conn_key := r.URL.Query().Get(QK_CONN_KEY)
	tcp_client := self.getTCPClient(conn_key)
	http_request := &httpRequest{
		httpWrapper: newHTTPWrapper(r, w),
	}
	switch strings.TrimPrefix(r.URL.Path, "/") {
	case QP_CONNECT:
		if tcp_client != nil {
			log.Warnf("connection already exist, url=[%s]", r.URL.String())
			http_request.httpWrapper.setErrorHappened()
		} else {
			addr := r.URL.Query().Get(QK_ADDR)
			tcp_client = newTCPClient(addr, &tcpClientMgrCallback{self, conn_key})
			if tcp_client == nil {
				log.Warnf("newTCPClient fail, url=[%s]", r.URL.String())
				http_request.httpWrapper.setErrorHappened()
			} else {
				log.Infof("newTCPClient succ, %s", tcp_client.String())
				self.addTCPClient(conn_key, tcp_client)
			}
		}
	case QP_DATA:
		if tcp_client == nil {
			log.Warnf("connection not exist, url=[%s]", r.URL.String())
			http_request.httpWrapper.setErrorHappened()
		} else {
			seq_number, _ := strconv.ParseInt(r.URL.Query().Get(QK_SEQ), 10, 64)
			tcp_client.pushHTTPRequest(seq_number, http_request)
			http_request.wg.Wait()
		}
	default:
		log.Warnf("invalid path, url=[%s]", r.URL.String())
		http_request.httpWrapper.setErrorHappened()
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
