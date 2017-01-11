package proxy

import (
	"bytes"
	"common"
	"io"
	"net/http"
	"net/url"
	"strconv"
	log "third/seelog"
	"time"
)

type iHTTPClient interface {
	pushTCPRequest(dn *dataBlock)
	popTCPRequest() (dn *dataBlock)
	isAlive() bool
}

type httpClient struct {
	hc      *http.Client
	host    string
	dest    string
	seq     int64
	connKey int64
	sendQ   chan *dataBlock
	recvQ   chan *dataBlock
	alive   bool
}

func newHTTPClient(host string, dest string) (hc iHTTPClient) {
	hc_impl := &httpClient{
		hc:      &http.Client{},
		host:    host,
		dest:    dest,
		seq:     0,
		connKey: common.GetCurrentTime(),
		sendQ:   make(chan *dataBlock, DataQueueSize),
		recvQ:   make(chan *dataBlock, DataQueueSize),
		alive:   true,
	}
	go hc_impl.processLoop()
	hc = hc_impl
	return hc
}

func (self *httpClient) destroy() {
	close(self.sendQ)
	close(self.recvQ)
}

func (self *httpClient) pushTCPRequest(dn *dataBlock) {
	self.sendQ <- dn
}

func (self *httpClient) popTCPRequest() (dn *dataBlock) {
	dn = <-self.recvQ
	return dn
}

func (self *httpClient) isAlive() bool {
	return self.alive
}

func (self *httpClient) processLoop() {
	for {
		timer := time.NewTicker(time.Second)
		select {
		case dn := <-self.sendQ:
			if dn == nil {
				break
			}
			self.sendData(dn)
		case <-timer.C:
			self.keepAlive()
		}
	}
}

func (self *httpClient) sendData(send_dn *dataBlock) {
	self.seq += 1

	u := url.URL{
		Scheme: "http",
		Host:   self.host,
		Path:   QP_DATA,
	}
	q := u.Query()
	q.Set(QK_CONN_KEY, strconv.FormatInt(self.connKey, 10))
	q.Set(QK_ADDR, self.dest)
	q.Set(QK_SEQ, strconv.FormatInt(self.seq, 10))
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
		log.Warnf("do http request fail, err=[%v] status=[%s]", err, res.Status)
		self.alive = false
	} else {
		for {
			recv_dn := &dataBlock{
				data: make([]byte, DataBlockSize),
			}
			read_ret, err := res.Body.Read(recv_dn.data)
			if err != nil {
				log.Warnf("read fail, read_ret=%d err=[%v]", read_ret, err)
				break
			} else {
				log.Debugf("recv data succ, len=%d", read_ret)
			}
			recv_dn.data = recv_dn.data[:read_ret]
			self.recvQ <- recv_dn
		}
	}
}

func (self *httpClient) keepAlive() {
	self.sendData(nil)
}
