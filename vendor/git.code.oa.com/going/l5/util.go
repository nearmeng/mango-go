package l5

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

//免拷贝[]byte转string
func String(b []byte) (s string) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pstring.Data = pbytes.Data
	pstring.Len = pbytes.Len
	return
}

//免拷贝string转[]byte
func Slice(s string) (b []byte) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pbytes.Data = pstring.Data
	pbytes.Len = pstring.Len
	pbytes.Cap = pstring.Len
	return
}

//计算最大公因数
func GreatestCommonDivider(a int32, b int32) int32 {
	if b > 0 {
		return GreatestCommonDivider(b, a%b)
	} else {
		return a
	}
}

//ip字符串转raw ip
func String2Ip(ip string) int32 {
	bytes := strings.Split(ip, ".")
	val := 0
	for k, v := range bytes {
		i, err := strconv.Atoi(v)
		if err != nil || i > 255 {
			return 0
		}
		val = val | i<<uint(8*(3-k))
	}
	return int32(val)
}

func Ip2String(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ip>>24)&0xff, (ip>>16)&0xff, (ip>>8)&0xff, ip&0xff)
}

func String2IpLittle(ip string) uint32 {
	bytes := strings.Split(ip, ".")
	val := 0
	for k, v := range bytes {
		i, err := strconv.Atoi(v)
		if err != nil || i > 255 {
			return 0
		}
		val = val | i<<uint(8*k)
	}
	return uint32(val)
}

func Ip2StringLittle(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", ip&0xff, (ip>>8)&0xff, (ip>>16)&0xff, (ip>>24)&0xff)
}

func HostInt16ToLittle(i uint16) uint16 {
	vec := []byte{byte(i >> 8), byte(i)}
	return binary.LittleEndian.Uint16(vec)
}
