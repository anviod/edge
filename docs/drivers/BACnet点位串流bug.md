---
layout: default
title: BACnet 点位串流 Bug 修复
description: EdgeX BACnet 点位串流 Bug 修复
---

# BACnet 点位串流 Bug 修复
基于配置修改 D:\code\edgex\conf\channels.yaml
使用最新配置文件来实现设备读取bacnet点位
比如 D:\code\edgex\conf\devices\bacnet-ip\bacnet-2228316.yaml
严格按照配置文件中的点位进行读取 配置文件不可修改的原则进行代码调整
特别注意：
* **DeviceID**：系统内唯一标识（Edge/平台侧）
* **Instance ID**：BACnet 网络唯一标识
* **ObjectID**：对象标识（Type + Instance）
* **Property**：对象属性标识

当前设备清单（验收范围 ：设备点位必须能正常读取 修复单设备导致整条链路不可用）：

* bacnet-18 → Instance ID 2228318 ->Setpoint.1	AnalogValue 1	318.00 验证点
* bacnet-16 → Instance ID 2228316 ->Setpoint.1	AnalogValue 1	316.00 验证点
* bacnet-17 → Instance ID 2228317 ->Setpoint.1	AnalogValue 1	317.00 验证点
* Room_FC_2014_19 → Instance ID 2228319 ->Setpoint.1	AnalogValue 1	319.00 验证点 (当前已经离线 前端UI必须显示设备离线/隔离)

编写点位测试代码 超时自动跳过 视为测试失败 继续轮询下一台设备
测试结果：
* 所有设备必须能正常读取点位
*  Room_FC_2014_19 离线时 必须能正常显示离线状态
*  其他设备离线时 必须能正常显示离线状态
*  所有设备的接口均不能超时 超时时间测试定为:3s

> ⚠ 验收前提：已确认所有设备物理运行正常且网络正常OK (Instance ID 2228319 手动关闭设备 需要实现前端能正常显示离线功能)

```curl 'http://127.0.0.1:8082/api/channels/jxy3kvpohmetzct0/devices/bacnet-16' \
  -H 'Accept: application/json, text/plain, */*' \
  -H 'Accept-Language: zh,zh-CN;q=0.9,en;q=0.8' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJlbWFpbCI6IiIsImlzcyI6IkluZHVzdHJpYWxFZGdlR2F0ZXdheSIsImV4cCI6MTc3Mjg2NDA3NywibmJmIjoxNzcyMjU5Mjc3fQ.m0k3SQ-B9n7sfSSYnzXjT0X0Vmq_cxjqNM1jw0w03vg' \
  -H 'Connection: keep-alive' \
  -H 'DNT: 1' \
  -H 'Referer: http://127.0.0.1:8082/' \
  -H 'Sec-Fetch-Dest: empty' \
  -H 'Sec-Fetch-Mode: cors' \
  -H 'Sec-Fetch-Site: same-origin' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36' \
  -H 'sec-ch-ua: "Not:A-Brand";v="99", "Google Chrome";v="145", "Chromium";v="145"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-platform: "Windows"' \
  -H 'sec-gpc: 1' \
  -H 'token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJlbWFpbCI6IiIsImlzcyI6IkluZHVzdHJpYWxFZGdlR2F0ZXdheSIsImV4cCI6MTc3Mjg2NDA3NywibmJmIjoxNzcyMjU5Mjc3fQ.m0k3SQ-B9n7sfSSYnzXjT0X0Vmq_cxjqNM1jw0w03vg'

  然后点位查看实时数据

curl 'http://127.0.0.1:8082/api/channels/jxy3kvpohmetzct0/devices/bacnet-16/points' \
  -H 'Accept: application/json, text/plain, */*' \
  -H 'Accept-Language: zh,zh-CN;q=0.9,en;q=0.8' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJlbWFpbCI6IiIsImlzcyI6IkluZHVzdHJpYWxFZGdlR2F0ZXdheSIsImV4cCI6MTc3Mjg2NDA3NywibmJmIjoxNzcyMjU5Mjc3fQ.m0k3SQ-B9n7sfSSYnzXjT0X0Vmq_cxjqNM1jw0w03vg' \
  -H 'Connection: keep-alive' \
  -H 'DNT: 1' \
  -H 'Referer: http://127.0.0.1:8082/' \
  -H 'Sec-Fetch-Dest: empty' \
  -H 'Sec-Fetch-Mode: cors' \
  -H 'Sec-Fetch-Site: same-origin' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36' \
  -H 'sec-ch-ua: "Not:A-Brand";v="99", "Google Chrome";v="145", "Chromium";v="145"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-platform: "Windows"' \
  -H 'sec-gpc: 1' \
  -H 'token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJlbWFpbCI6IiIsImlzcyI6IkluZHVzdHJpYWxFZGdlR2F0ZXdheSIsImV4cCI6MTc3Mjg2NDA3NywibmJmIjoxNzcyMjU5Mjc3fQ.m0k3SQ-B9n7sfSSYnzXjT0X0Vmq_cxjqNM1jw0w03vg'
```
问题点: bacnet-16 设备正常 但是/api/channels/jxy3kvpohmetzct0/devices/bacnet-16/points读取超时