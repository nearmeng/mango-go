#L5Api for go#

不同于cgo版本，纯Go实现的l5api，基于l5api version:40007


#Protocol#
##QOS_CMD_QUERY_SNAME##
根据名称查询mod+cmd

##QOS_CMD_BATCH_GET_ROUTE_WEIGHT##
根据mod+cmd批量查询带权重的server

##QOS_CMD_GET_STAT##
上报分配统计

##QOS_CMD_CALLER_UPDATE_BIT64##
上报调用状态


#Tips#
抓包（与agent通信）：
tcpdump -i lo udp and \(\(dst host 127.0.0.1 and dst port 8888\) or \(src host 127.0.0.1 and src port 8888\)\) -x -nn


#Todo#
manage api
