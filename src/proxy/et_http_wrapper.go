package proxy

import (
	"io"
	"net/http"
	log "third/seelog"
)

type iHTTPWrapper interface {
	popData() (dn *dataBlock)
	setErrorHappened()
	pushData(dn *dataBlock)
}

type httpWrapper struct {
	req       *http.Request
	resWriter http.ResponseWriter
}

func newHTTPService(req *http.Request, res_writer http.ResponseWriter) (hs iHTTPWrapper) {
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
	if err == io.EOF {
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
}

func (self *httpWrapper) pushData(dn *dataBlock) {
	if dn != nil {
		self.resWriter.Write(dn.data)
		log.Debugf("http response write data, len=%d", len(dn.data))
	} else {
		self.resWriter.Write(nil)
		log.Debugf("http response write nil")
	}
}
