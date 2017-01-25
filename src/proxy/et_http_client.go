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
		recvQ:     make(chan *dataBlock, DataQueueSize),
		alive:     true,
	}
	go hc_impl.processLoop()
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
	for self.isAlive() {
		timer := time.NewTicker(time.Duration(common.G.Client.KeepAliveTimeSec) * time.Second)
		select {
		case dn := <-self.sendQ:
			if dn == nil {
				break
			}
			self.sendData(dn)
			continue
		case <-self.sendQsync:
			break
		case <-timer.C:
			log.Infof("timer ticket")
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
	}
}

func (self *httpClient) keepAlive() {
	self.sendData(nil)
}

func (self *httpClient) String() string {
	return fmt.Sprintf("this=%p host=[%s] dest=[%s] seq=%d connKey=%d alive=%t",
		self, self.host, self.dest, self.seq, self.connKey, self.alive)
}
