package proxy

import (
	"io"
	"net/http"
)

type iHTTPService interface {
	popData() (dn *dataBlock)
	setErrorHappened()
	pushData(dn *dataBlock)
}

type httpService struct {
	req       *http.Request
	resWriter http.ResponseWriter
}

func newHTTPService(req *http.Request, res_writer http.ResponseWriter) iHTTPService {
	hs_impl := &httpService{
		req:       req,
		resWriter: res_writer,
	}
	return hs_impl
}

func (self *httpService) popData() (dn *dataBlock) {
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

func (self *httpService) setErrorHappened() {
	self.resWriter.WriteHeader(http.StatusBadGateway)
}

func (self *httpService) pushData(dn *dataBlock) {
	if dn != nil {
		self.resWriter.Write(dn.data)
	} else {
		self.resWriter.Write(nil)
	}
}
