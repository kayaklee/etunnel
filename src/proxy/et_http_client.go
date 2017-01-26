package proxy

import (
	"bytes"
	"common"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	log "third/seelog"
	"time"
)

type iHTTPClient interface {
	destroy()
	isAlive() bool
	pushTCPRequest(dn *dataBlock)
	popTCPResponse() (dn *dataBlock)
	String() string
}

type httpClient struct {
	hc        *http.Client
	host      string
	dest      string
	seq       int64
	connKey   int64
	sendQ     chan *dataBlock
	sendQsync chan *dataBlock
	sendNop   chan *dataBlock
	respQ     chan *http.Response
	recvQ     chan *dataBlock
	alive     bool
}

func newHTTPClient(host string, dest string) (hc iHTTPClient) {
	hc_impl := &httpClient{
		hc:        &http.Client{},
		host:      host,
		dest:      dest,
		seq:       0,
		connKey:   common.GetCurrentTime(),
		sendQ:     make(chan *dataBlock, DataQueueSize),
		sendQsync: make(chan *dataBlock, 1),
		sendNop:   make(chan *dataBlock, 1),
		recvQ:     make(chan *dataBlock, DataQueueSize),
		respQ:     make(chan *http.Response, DataQueueSize),
		alive:     true,
	}
	go hc_impl.processLoop()
	go hc_impl.recvLoop()
	hc = hc_impl
	return hc
}

func (self *httpClient) destroy() {
	log.Infof("%s", self.String())
	self.alive = false
	close(self.sendQsync)
	self.recvQ <- nil
}

func (self *httpClient) pushTCPRequest(dn *dataBlock) {
	self.sendQ <- dn
}

func (self *httpClient) popTCPResponse() (dn *dataBlock) {
	select {
	case dn = <-self.recvQ:
	default:
	}
	if dn == nil && self.isAlive() {
		select {
		case dn = <-self.recvQ:
		}
	}
	return dn
}

func (self *httpClient) isAlive() bool {
	return self.alive
}

func (self *httpClient) processLoop() {
	self.sendNop <- nil
	for self.isAlive() {
		timer := time.NewTicker(time.Duration(common.G.Client.KeepAliveTimeSec) * time.Second)
		select {
		case dn := <-self.sendQ:
			if dn == nil {
				break
			}
			self.sendData(dn, false)
			continue
		case <-self.sendNop:
			self.sendData(nil, false)
			continue
		case <-self.sendQsync:
			break
		case <-timer.C:
			log.Infof("timer ticket")
			self.keepAlive()
		}
	}
	self.respQ <- nil
}

func (self *httpClient) sendData(send_dn *dataBlock, is_keep_alive bool) {
	path := QP_DATA
	if !is_keep_alive {
		self.seq += 1
	} else {
		path = QP_KEEPALIVE
	}

	u := url.URL{
		Scheme: "http",
		Host:   self.host,
		Path:   path,
	}
	q := u.Query()
	q.Set(QK_CONN_KEY, strconv.FormatInt(self.connKey, 10))
	q.Set(QK_ADDR, self.dest)
	if !is_keep_alive {
		q.Set(QK_SEQ, strconv.FormatInt(self.seq, 10))
	}
	u.RawQuery = q.Encode()

	log.Debugf("send date to url=[%s]", u.String())
	var body io.Reader
	if send_dn != nil {
		body = bytes.NewReader(send_dn.data)
	}
	req, _ := http.NewRequest(http.MethodGet, u.String(), body)
	res, err := self.hc.Do(req)
	if nil != err ||
		http.StatusOK != res.StatusCode {
		status := ""
		if res != nil {
			status = res.Status
		}
		log.Warnf("do http request fail, err=[%v] status=[%s]", err, status)
		self.alive = false
	} else if is_keep_alive {
		res.Body.Close()
	} else {
		self.respQ <- res
	}
}

func (self *httpClient) recvLoop() {
	for res := range self.respQ {
		if res == nil {
			break
		}
		for {
			recv_dn := &dataBlock{
				data: make([]byte, DataBlockSize),
			}
			read_ret, err := res.Body.Read(recv_dn.data)
			if read_ret > 0 {
				recv_dn.data = recv_dn.data[:read_ret]
				self.recvQ <- recv_dn
			}
			if err != nil {
				if err != io.EOF {
					log.Warnf("read fail, read_ret=%d err=[%v]", read_ret, err)
				} else {
					log.Infof("connection close, read_ret=%d err=[%v]", read_ret, err)
				}
				res.Body.Close()
				break
			} else {
				log.Debugf("recv data succ, len=%d", read_ret)
			}
		}
		select {
		case self.sendNop <- nil:
		default:
		}
	}
}

func (self *httpClient) keepAlive() {
	self.sendData(nil, true)
}

func (self *httpClient) String() string {
	return fmt.Sprintf("this=%p host=[%s] dest=[%s] seq=%d connKey=%d alive=%t sendQLen=%d respQLen=%d recvQLen=%d",
		self, self.host, self.dest, self.seq, self.connKey, self.alive, len(self.sendQ), len(self.respQ), len(self.recvQ))
}
