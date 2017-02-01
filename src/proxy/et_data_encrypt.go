package proxy

import (
	"crypto/aes"
	"crypto/cipher"
	log "third/seelog"
)

type encryptFilter struct {
	block cipher.Block
}

func newEncryptFilter(key []byte) (ef *encryptFilter) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Warnf("NewCipher fail, err=[%s]", err.Error())
	} else {
		log.Infof("NewCipher success")
		ef = &encryptFilter{
			block: block,
		}
	}
	return ef
}

//encrypt
//base64
//json
//stream encoding
func (self *encryptFilter) onDataRecv(dn *dataBlock) (*dataBlock, error) {
	return dn, nil
}

//stream decoding
//json
//base64
//decrypt
func (self *encryptFilter) onDataSend(dn *dataBlock) (*dataBlock, error) {
	return dn, nil
}

func (self *encryptFilter) dataBlockSize() int64 {
	return DataBlockSize
}
