// Copyright (c) 2017, Tencent. All rights reserved.
// Authors esliznwang cheaterlin

package l5

import (
	"errors"
	"fmt"
	"os"
)

var (
	apiRouteWeightMapFile    = "/data/L5Backup/apiRouteMap.weight.bin"
	ErrOpenStaticRouteTable  = errors.New("open static route table file fail")
	ErrParseStaticRouteTable = errors.New("parse static route table file fail")
	ErrNotStandard           = errors.New("not standard api implemention err")
)

// api not standard
type Dest struct {
	Ip     string
	Port   int
	Weight int
}

func NotSTLGetDestByModCmd(mod, cmd int32) (dests []Dest, err error) {
	_, err = anonymouss.Get(mod, cmd).Get()
	if err != nil {
		return
	}

	d := anonymouss.Get(mod, cmd)
	weightedRoundRobinBalancer, ok := d.balancer.(*weightedRoundRobin)
	if !ok || weightedRoundRobinBalancer == nil {
		return nil, ErrNotStandard
	}

	weightedRoundRobinBalancer.l.Lock()
	for i := range weightedRoundRobinBalancer.list {
		dests = append(dests, Dest{
			Ip:     weightedRoundRobinBalancer.list[i].Ip(),
			Port:   weightedRoundRobinBalancer.list[i].Port(),
			Weight: int(weightedRoundRobinBalancer.list[i].weight),
		})
	}
	weightedRoundRobinBalancer.l.Unlock()

	return dests, nil
}

func getRouteTable(mod, cmd int32) (dests []Dest, err error) {
	var fp *os.File

	if fp, err = os.Open(apiRouteWeightMapFile); err != nil {
		return nil, ErrOpenStaticRouteTable
	}
	defer fp.Close()

	for {
		var (
			_mod    int32
			_cmd    int32
			_ip     string
			_port   uint16
			_weight int32
		)

		if n, fail := fmt.Fscanln(fp, &_mod, &_cmd, &_ip, &_port, &_weight); n == 0 || fail != nil {
			break
		}
		if mod != _mod || cmd != _cmd {
			continue
		}
		dests = append(dests, Dest{
			Ip:     _ip,
			Port:   int(_port),
			Weight: int(_weight),
		})
	}

	return dests, nil
}
