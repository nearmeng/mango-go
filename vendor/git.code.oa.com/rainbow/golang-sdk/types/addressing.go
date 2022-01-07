package types

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
)

// AddressType 地址类型
type AddressType string

// Addressing 地址访问接口
type Addressing interface {
	ParseAddress(addr string) error
	GetAddress() string
	UpdateAddress(ip, port string) error
}

// AddressBase basic address
type AddressBase struct {
	Type   AddressType
	Addr   string
	ModID  int
	CmdID  int
	IPList []string
}

// GetAddress 获取地址
func (ab *AddressBase) GetAddress() string {
	if ab.Type == "http" {
		return ab.Addr
	}
	if ab.Type == "ip" {
		return string("ip://") + ab.IPList[rand.Int()%len(ab.IPList)]
	}
	if ab.Type == "cl5" {
		return ab.Addr
	}
	return ""
}

// UpdateAddress update address
func (ab *AddressBase) UpdateAddress() error {
	return nil
}

// ParseAddress 地址解析
// cl5://65026305:65536
// http://api.cc.sid.oa.com
// ip://9.56.2.161:8080,9.56.2.161:8080
// polaris://65026305:65536
func (ab *AddressBase) ParseAddress(addr string) error {
	// cl5
	if len(addr) > 6 && addr[:3] == "cl5" {
		addrs := strings.Split(addr, "://")
		if len(addrs) != 2 {
			return errors.New("address invalid")
		}
		ids := strings.Split(addrs[1], ":")
		if len(ids) != 2 {
			return errors.New("cl5 address invalid")
		}
		ab.ModID, _ = strconv.Atoi(ids[0])
		ab.CmdID, _ = strconv.Atoi(ids[1])
		ab.Addr = addr
		ab.Type = "cl5"
		return nil
	}
	if len(addr) > 7 && addr[:4] == "http" {
		addrs := strings.Split(addr, "://")
		if len(addrs) != 2 {
			return errors.New("address invalid")
		}
		ab.Addr = addr
		ab.Type = "http"
		return nil
	}

	if len(addr) > 5 && addr[:5] == "ip://" {
		ab.IPList = strings.Split(addr[5:], ",")
		if len(ab.IPList) < 1 {
			return errors.New("address invalid")
		}
		ab.Addr = addr
		ab.Type = "ip"
		return nil
	}

	if len(addr) > 10 && addr[:10] == "polaris://" {
		addrs := strings.Split(addr, "://")
		if len(addrs) != 2 {
			return errors.New("address invalid")
		}
		ab.Addr = addrs[1]
		ab.Type = "polaris"
		return nil
	}
	return errors.New("address invalid")
}
