// Package l5 纯go版本, l5 agent地址：http://yun.isd.com/index.php/package/versions/?product=public&package=l5_protocol_32os
package l5

/*
@version v1.0
@author esliznwang cheaterlin nickzydeng
@source http://git.code.oa.com/components/l5
@copyright Copyright (c) 2018 Tencent Corporation, All Rights Reserved
@license http://opensource.org/licenses/gpl-license.php GNU Public License

You may not use this file except in compliance with the License.

Most recent version can be found at:
http://git.code.oa.com/going_proj/going_proj

Please see README.md for more information.
*/

// ApiGetSid 通过l5域名获取Domain
func ApiGetSid(sid string) (*Domain, error) {
	return domainss.Query(sid)
}

// ApiGetRouteBySid 通过l5域名获取路由Server wiki: http://km.oa.com/articles/show/361349?ts=1524042713, 需要安装agent：http://yun.isd.com/index.php/package/versions/?product=public&package=dns_l5_agent
func ApiGetRouteBySid(sid string) (*Server, error) {
	domain, err := domainss.Query(sid)
	if err != nil {
		return nil, err
	}
	return domain.Get()
}

// ApiGetRoute 通过modid:cmdid获取路由Server
func ApiGetRoute(mod int32, cmd int32) (*Server, error) {
	return anonymouss.Get(mod, cmd).Get()
}

// ApiRouteResultUpdate 上报错误码耗时结果
func ApiRouteResultUpdate(s *Server, result int32, usetime uint64) error {
	if s == nil {
		return nil
	}

	return s.StatUpdate(result, usetime)
}

// ApiGetRouteTable 通过modid:cmdid获取所有IP列表
func ApiGetRouteTable(mod int32, cmd int32) ([]Dest, error) {
	list, err := getRouteTable(mod, cmd)
	if err == nil {
		return list, err
	}
	anonymouss.Get(mod, cmd).Get()

	list, err = getRouteTable(mod, cmd)
	return list, err
}
