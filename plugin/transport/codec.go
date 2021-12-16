package transport

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/nearmeng/mango-go/plugin/log"
)

type Codec interface {
	Encode(c Conn, buff []byte) ([]byte, error)
	Decode(c Conn) ([]byte, error)
}

var (
	_codec Codec = &DefaultCodec{}
)

func GetCodec() Codec {
	return _codec
}

func SetCodec(codec Codec) {
	_codec = codec
}

//=====================================================

type DefaultCodec struct {
}

var (
	_headSize = 8
)

type PreHead struct {
	HeaderSize uint32
	BodySize   uint32
}

func (codec *DefaultCodec) Decode(c Conn) ([]byte, error) {
	headBuff := make([]byte, _headSize)

	n, err := c.Read(headBuff)
	if err != nil {
		var e net.Error
		if errors.As(err, &e) && e.Timeout() {
			log.Error("client %s is stopped for timeout", c.GetRemoteAddr().String())
			return nil, err
		} else {
			log.Error("client %s is stopped for err %s", c.GetRemoteAddr().String(), err.Error())
			return nil, err
		}
	}

	if n != _headSize {
		log.Error("client %s head is not match", c.GetRemoteAddr().String())
		return nil, errors.New("head not match")
	}

	header := PreHead{
		HeaderSize: binary.LittleEndian.Uint32(headBuff),
		BodySize:   binary.LittleEndian.Uint32(headBuff[4:8]),
	}

	fmt.Printf("decode recv headbuff size %d header_size %d body_size %d\n", n, header.HeaderSize, header.BodySize)

	dataBuff := make([]byte, 4+header.HeaderSize+header.BodySize)
	binary.LittleEndian.PutUint32(dataBuff[0:4], header.HeaderSize)

	n, err = c.Read(dataBuff[4:])
	if err != nil {
		var e net.Error
		if errors.As(err, &e) && e.Timeout() {
			log.Error("client %s is stopped for timeout", c.GetRemoteAddr().String())
			return nil, err
		} else {
			log.Error("client %s is stopped for err %s", c.GetRemoteAddr().String(), err.Error())
			return nil, err
		}
	}

	if uint32(n) != (header.HeaderSize + header.BodySize) {
		log.Error("client %s body not match", c.GetRemoteAddr().String())
		return nil, errors.New("body not match")
	}

	fmt.Printf("decode recv bodyBuff size %d\n", n)

	return dataBuff, nil
}

func (codec *DefaultCodec) Encode(c Conn, buff []byte) ([]byte, error) {
	headerSize := binary.LittleEndian.Uint32(buff)
	pkgLen := len(buff) - 4

	result := make([]byte, _headSize+pkgLen)

	binary.LittleEndian.PutUint32(result[0:4], headerSize)
	binary.LittleEndian.PutUint32(result[4:8], uint32(pkgLen))

	copy(result[8:], buff[4:])

	return result, nil
}
