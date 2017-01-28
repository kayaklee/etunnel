package proxy

import (
	"fmt"
	"io"
	"net/http"
	log "third/seelog"
)

type iHTTPWrapper interface {
	popData() (dn *dataBlock)
	setErrorHappened()
	startResponse()
	pushData(dn *dataBlock)
	String() string
}

type httpWrapper struct {
	req       *http.Request
	resWriter http.ResponseWriter
}

func newHTTPWrapper(req *http.Request, res_writer http.ResponseWriter) (hs iHTTPWrapper) {
	hs_impl := &httpWrapper{
		req:       req,
		resWriter: res_writer,
	}
	hs = hs_impl
	return hs
}

func (self *httpWrapper) popData() (dn *dataBlock) {
	dn = &dataBlock{
		data: make([]byte, DataBlockSize),
	}
	read_ret, err := self.req.Body.Read(dn.data)
	if err != nil {
		if err != io.EOF && err != http.ErrBodyReadAfterClose {
			log.Warnf("read fail, read_ret=%d err=[%v]", read_ret, err)
		} else {
			log.Infof("connection close, read_ret=%d err=[%v]", read_ret, err)
		}
		self.req.Body.Close()
	}
	if read_ret > 0 {
		dn.data = dn.data[:read_ret]
	} else {
		dn = nil
	}
	return dn
}

func (self *httpWrapper) setErrorHappened() {
	self.resWriter.WriteHeader(http.StatusBadGateway)
	self.resWriter.Write(nil)
	self.resWriter.(http.Flusher).Flush()
}

func (self *httpWrapper) startResponse() {
	self.resWriter.WriteHeader(http.StatusOK)
	self.resWriter.(http.Flusher).Flush()
}

func (self *httpWrapper) pushData(dn *dataBlock) {
	if dn != nil {
		self.resWriter.Write(dn.data)
		log.Debugf("http response write data, len=%d", len(dn.data))
	} else {
		self.resWriter.Write(nil)
		log.Debugf("http response write nil")
	}
	self.resWriter.(http.Flusher).Flush()
}

func (self *httpWrapper) String() string {
	return fmt.Sprintf("url=[%s]", self.req.URL.String())
}
