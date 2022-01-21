package msg

import (
	"encoding/binary"
	"errors"
	"strings"

	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/proto/csproto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type CSCodec interface {
	Encode(header *csproto.SCHead, body proto.Message) ([]byte, error)
	Decode(data []byte) (*csproto.CSHead, proto.Message)
}

var (
	CODEC_DEFAULT = "default"
)

var (
	_csCodecFactory = make(map[string]CSCodec)
	_maxBuffSize    = 500 * 1024
)

func registerCodec(t string, code CSCodec) {
	_csCodecFactory[t] = code
}

func getCodec(t string) CSCodec {
	return _csCodecFactory[t]
}

type DefaultCSCodec struct {
}

func (c *DefaultCSCodec) Encode(header *csproto.SCHead, body proto.Message) ([]byte, error) {
	buff := make([]byte, _maxBuffSize)

	data, err := proto.Marshal(header)
	if err != nil {
		return nil, err
	}

	headerSize := len(data)
	binary.LittleEndian.PutUint32(buff[0:4], uint32(headerSize))

	n := copy(buff[4:], data)
	if n != headerSize {
		return nil, errors.New("buff copy failed")
	}

	data, err = proto.Marshal(body)
	if err != nil {
		return nil, err
	}

	n = copy(buff[4+headerSize:], data)
	if n != len(data) {
		return nil, errors.New("buff copy failed")
	}

	return buff[0 : 4+int(headerSize)+n], nil
}

func (c *DefaultCSCodec) Decode(data []byte) (*csproto.CSHead, proto.Message) {
	var header csproto.CSHead

	headerSize := binary.LittleEndian.Uint32(data[0:4])
	err := proto.Unmarshal(data[4:4+headerSize], &header)
	if err != nil {
		log.Error("proto unmarshal failed")
		return nil, nil
	}

	msgid := header.GetMsgid()
	msgStr, ok := csproto.CSMessageID_name[msgid]
	if !ok {
		log.Error("msgid %d is not implement in proto", msgid)
		return nil, nil
	}

	msgName := protoreflect.FullName("proto." + strings.ToUpper(msgStr))
	msgType, err := protoregistry.GlobalTypes.FindMessageByName(msgName)
	if err != nil {
		log.Error("find message by name %s failed, err %v", msgName, err)
		return nil, nil
	}

	msg := msgType.New().Interface()
	err = proto.Unmarshal(data[4+headerSize:], msg)
	if err != nil {
		log.Error("proto unmarshal failed")
		return nil, nil
	}

	return &header, msg
}

func init() {
	registerCodec(CODEC_DEFAULT, &DefaultCSCodec{})
}
