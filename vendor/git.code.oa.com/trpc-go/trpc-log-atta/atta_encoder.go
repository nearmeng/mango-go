package attalog

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const _hex = "0123456789abcdef"
const logMaxLength = 60 * 1024 // atta udp包大小有上限

var bufferpool = buffer.NewPool()
var nullLiteralBytes = []byte("")
var separatorBytes = []byte("|")

var _attaPool = sync.Pool{New: func() interface{} {
	return &attaEncoder{}
}}

func getAttaEncoder() *attaEncoder {
	return _attaPool.Get().(*attaEncoder)
}

func putAttaEncoder(enc *attaEncoder) {
	if enc.reflectBuf != nil {
		enc.reflectBuf.Free()
	}

	enc.EncoderConfig = nil
	enc.fields = nil
	enc.messageKey = ""
	enc.autoEscape = false

	enc.buf = nil
	enc.field2Buffer = nil

	enc.reflectBuf = nil
	enc.reflectEnc = nil

	_attaPool.Put(enc)
}

type attaEncoder struct {
	*zapcore.EncoderConfig
	fields     []string // atta申请字段，严格有序
	messageKey string
	autoEscape bool // 是否转义

	buf          *buffer.Buffer    // encode过程中复用的buffer
	field2Buffer map[string]string // 每个field对应的log内容

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc *json.Encoder
}

func newAttaEncoder(cfg zapcore.EncoderConfig, fields []string, messageKey string, autoEscape bool) *attaEncoder {
	return &attaEncoder{
		EncoderConfig: &cfg,
		fields:        fields,
		messageKey:    messageKey,
		autoEscape:    autoEscape,
		buf:           bufferpool.Get(),
		field2Buffer:  make(map[string]string),
	}
}

func (enc *attaEncoder) saveBuf(key string) {
	enc.field2Buffer[key] = enc.buf.String()
	enc.truncate()
}

// AddArray encode 数组
func (enc *attaEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	err := arr.MarshalLogArray(enc)
	if err != nil {
		return err
	}

	enc.saveBuf(key)
	return nil
}

// AddObject encode 结构体
func (enc *attaEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	err := obj.MarshalLogObject(enc)
	if err != nil {
		return err
	}

	enc.saveBuf(key)
	return nil
}

// AddBinary encode 二进制
func (enc *attaEncoder) AddBinary(key string, val []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

// AddByteString encode byte字符串
func (enc *attaEncoder) AddByteString(key string, val []byte) {
	enc.AppendByteString(val)
	enc.saveBuf(key)
}

// AddBool encode bool
func (enc *attaEncoder) AddBool(key string, val bool) {
	enc.AppendBool(val)

	enc.saveBuf(key)
}

// AddComplex128 encode Complex128
func (enc *attaEncoder) AddComplex128(key string, val complex128) {
	enc.AppendComplex128(val)
	enc.saveBuf(key)
}

// AddDuration encode Duration
func (enc *attaEncoder) AddDuration(key string, val time.Duration) {
	enc.AppendDuration(val)

	enc.saveBuf(key)
}

// AddFloat64 encode float64
func (enc *attaEncoder) AddFloat64(key string, val float64) {
	enc.AppendFloat64(val)
	enc.saveBuf(key)
}

// AddInt64 encode int64
func (enc *attaEncoder) AddInt64(key string, val int64) {
	enc.AppendInt64(val)
	enc.saveBuf(key)
}

// AddReflected encode interface
func (enc *attaEncoder) AddReflected(key string, obj interface{}) error {
	err := enc.AppendReflected(obj)
	if err != nil {
		return err
	}

	enc.saveBuf(key)
	return nil
}

// OpenNamespace 命名空间，atta格式不需要
func (enc *attaEncoder) OpenNamespace(key string) {
	// atta 没有namespace
	//enc.saveBuf(key)
}

// AddString encode 字符串
func (enc *attaEncoder) AddString(key, val string) {
	enc.AppendString(val)
	enc.saveBuf(key)
}

// AddTime encode time
func (enc *attaEncoder) AddTime(key string, val time.Time) {
	enc.AppendTime(val)
	enc.saveBuf(key)
}

// AddUint64 encode uint64
func (enc *attaEncoder) AddUint64(key string, val uint64) {
	enc.AppendUint64(val)
	enc.saveBuf(key)
}

// AddComplex64 encode complex64
func (enc *attaEncoder) AddComplex64(k string, v complex64) { enc.AddComplex128(k, complex128(v)) }

// AddFloat32 encode float32
func (enc *attaEncoder) AddFloat32(k string, v float32) { enc.AddFloat64(k, float64(v)) }

// AddInt encode int
func (enc *attaEncoder) AddInt(k string, v int) { enc.AddInt64(k, int64(v)) }

// AddInt32 encode int32
func (enc *attaEncoder) AddInt32(k string, v int32) { enc.AddInt64(k, int64(v)) }

// AddInt16 encode int16
func (enc *attaEncoder) AddInt16(k string, v int16) { enc.AddInt64(k, int64(v)) }

// AddInt8 encode int8
func (enc *attaEncoder) AddInt8(k string, v int8) { enc.AddInt64(k, int64(v)) }

// AddUint encode uint
func (enc *attaEncoder) AddUint(k string, v uint) { enc.AddUint64(k, uint64(v)) }

// AddUint32 encode uint32
func (enc *attaEncoder) AddUint32(k string, v uint32) { enc.AddUint64(k, uint64(v)) }

// AddUint16 encode uint16
func (enc *attaEncoder) AddUint16(k string, v uint16) { enc.AddUint64(k, uint64(v)) }

// AddUint8 encode uint8
func (enc *attaEncoder) AddUint8(k string, v uint8) { enc.AddUint64(k, uint64(v)) }

// AddUintptr encode uintptr
func (enc *attaEncoder) AddUintptr(k string, v uintptr) { enc.AddUint64(k, uint64(v)) }

// Clone attaEncoder拷贝
func (enc *attaEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *attaEncoder) clone() *attaEncoder {
	clone := getAttaEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.fields = enc.fields
	clone.messageKey = enc.messageKey
	clone.autoEscape = enc.autoEscape
	clone.buf = bufferpool.Get()
	clone.field2Buffer = make(map[string]string)
	for k, v := range enc.field2Buffer {
		clone.field2Buffer[k] = v
	}
	return clone
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}

// EncodeEntry 日志数据encode 入口
func (enc *attaEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := enc.clone()

	if final.LevelKey != "" {
		cur := final.buf.Len()
		final.EncodeLevel(ent.Level, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeLevel was a no-op. Fall back to strings to keep
			// output JSON valid.
			final.AppendString(ent.Level.String())
		}

		final.saveBuf(final.LevelKey)
	}
	if final.TimeKey != "" {
		final.AddTime(final.TimeKey, ent.Time)
	}
	if ent.LoggerName != "" && final.NameKey != "" {
		nameEncoder := final.EncodeName

		// if no name encoder provided, fall back to FullNameEncoder for backwards
		// compatibility
		if nameEncoder == nil {
			nameEncoder = zapcore.FullNameEncoder
		}

		cur := final.buf.Len()
		nameEncoder(ent.LoggerName, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeName was a no-op. Fall back to strings to
			// keep output JSON valid.
			final.AppendString(ent.LoggerName)
		}
		final.saveBuf(final.NameKey)
	}
	if ent.Caller.Defined && final.CallerKey != "" {
		cur := final.buf.Len()
		final.EncodeCaller(ent.Caller, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeCaller was a no-op. Fall back to strings to
			// keep output JSON valid.
			final.AppendString(ent.Caller.String())
		}
		final.saveBuf(final.CallerKey)
	}
	if final.MessageKey != "" {
		if enc.autoEscape {
			final.AddString(enc.MessageKey, ent.Message)
		} else {
			final.field2Buffer[enc.MessageKey] = ent.Message
		}
	}

	if len(enc.field2Buffer) > 0 {
		for k, v := range enc.field2Buffer {
			final.field2Buffer[k] = v
		}
	}

	addFields(final, fields)

	if ent.Stack != "" && final.StacktraceKey != "" {
		final.AddString(final.StacktraceKey, ent.Stack)
	}

	final.generateAttaLog(0)

	logLength := final.buf.Len()
	if logLength > logMaxLength {
		final.truncate()

		// 日志截断msg字段
		extraLength := logLength - logMaxLength
		final.generateAttaLog(extraLength)
	}

	ret := final.buf
	putAttaEncoder(final)

	return ret, nil
}

func (enc *attaEncoder) generateAttaLog(extraLength int) {
	endIndex := len(enc.fields) - 1
	for i, field := range enc.fields {
		b, ok := enc.field2Buffer[field]
		if ok {
			if field == enc.messageKey {
				remainLength := len(b) - extraLength
				if remainLength > 0 {
					enc.buf.Write([]byte(b[0:remainLength]))
				} else {
					enc.buf.Write(nullLiteralBytes)
				}
			} else {
				enc.buf.Write([]byte(b))
			}
		} else {
			enc.buf.Write(nullLiteralBytes)
		}
		if i != endIndex {
			enc.buf.Write(separatorBytes)
		}
	}
}

func (enc *attaEncoder) truncate() {
	enc.buf.Reset()
}

func (enc *attaEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = bufferpool.Get()
		enc.reflectEnc = json.NewEncoder(enc.reflectBuf)

		// For consistency with our custom JSON encoder.
		enc.reflectEnc.SetEscapeHTML(false)
	} else {
		enc.reflectBuf.Reset()
	}
}

// AppendArray encode 数组
func (enc *attaEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	return arr.MarshalLogArray(enc)
}

// AppendObject encode 结构体
func (enc *attaEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	//return obj.MarshalLogObject(enc)
	return nil
}

// AppendBool encode bool
func (enc *attaEncoder) AppendBool(val bool) {
	enc.buf.AppendBool(val)
}

// AppendByteString encode bytes
func (enc *attaEncoder) AppendByteString(val []byte) {
	enc.safeAddByteString(val)
}

// AppendComplex128 encode complex128
func (enc *attaEncoder) AppendComplex128(val complex128) {
	// Cast to enc platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	enc.buf.AppendByte('"')
	// Because we're always in enc quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, 64)
	enc.buf.AppendByte('+')
	enc.buf.AppendFloat(i, 64)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

// AppendDuration encode time.Duration
func (enc *attaEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	enc.EncodeDuration(val, enc)
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is enc no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

// AppendInt64 encode int64
func (enc *attaEncoder) AppendInt64(val int64) {
	enc.buf.AppendInt(val)
}

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (enc *attaEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	if err := enc.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	enc.reflectBuf.TrimNewline()
	return enc.reflectBuf.Bytes(), nil
}

// AppendReflected encode interface
func (enc *attaEncoder) AppendReflected(val interface{}) error {
	valueBytes, err := enc.encodeReflected(val)
	if err != nil {
		return err
	}
	_, err = enc.buf.Write(valueBytes)
	return err
}

// AppendString encode string
func (enc *attaEncoder) AppendString(val string) {
	enc.safeAddString(val)
}

// AppendTime encode time.Time
func (enc *attaEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	enc.EncodeTime(val, enc)
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is enc no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

// AppendUint64 encode uint64
func (enc *attaEncoder) AppendUint64(val uint64) {
	enc.buf.AppendUint(val)
}

func (enc *attaEncoder) appendFloat(val float64, bitSize int) {
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString escapes a string and appends it to the internal buffer.
func (enc *attaEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *attaEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character or atta seperator represented in a single byte.
func (enc *attaEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '|' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '|':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(_hex[b>>4])
		enc.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (enc *attaEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

// AppendComplex64 encode complex64
func (enc *attaEncoder) AppendComplex64(v complex64) { enc.AppendComplex128(complex128(v)) }

// AppendFloat64 encode float64
func (enc *attaEncoder) AppendFloat64(v float64) { enc.appendFloat(v, 64) }

// AppendFloat32 encode float32
func (enc *attaEncoder) AppendFloat32(v float32) { enc.appendFloat(float64(v), 32) }

// AppendInt encode int
func (enc *attaEncoder) AppendInt(v int) { enc.AppendInt64(int64(v)) }

// AppendInt32 encode int32
func (enc *attaEncoder) AppendInt32(v int32) { enc.AppendInt64(int64(v)) }

// AppendInt16 encode int16
func (enc *attaEncoder) AppendInt16(v int16) { enc.AppendInt64(int64(v)) }

// AppendInt8 encode int8
func (enc *attaEncoder) AppendInt8(v int8) { enc.AppendInt64(int64(v)) }

// AppendUint encode uint
func (enc *attaEncoder) AppendUint(v uint) { enc.AppendUint64(uint64(v)) }

// AppendUint32 encode uint32
func (enc *attaEncoder) AppendUint32(v uint32) { enc.AppendUint64(uint64(v)) }

// AppendUint16 encode uint16
func (enc *attaEncoder) AppendUint16(v uint16) { enc.AppendUint64(uint64(v)) }

// AppendUint8 encode uint8
func (enc *attaEncoder) AppendUint8(v uint8) { enc.AppendUint64(uint64(v)) }

// AppendUintptr encode uintptr
func (enc *attaEncoder) AppendUintptr(v uintptr) { enc.AppendUint64(uint64(v)) }
