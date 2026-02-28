当前要实现的架构特点（多设备隔离、设备级质量裁决、全部设备必须 Good）进行强化，并统一使用：
基于配置修改 D:\code\edgex\conf\channels.yaml
使用最新配置文件来实现设备读取bacnet点位
比如 D:\code\edgex\conf\devices\bacnet-ip\bacnet-2228316.yaml
严格按照配置文件中的点位进行读取 配置文件不可修改的原则进行代码调整
特别注意：
* **DeviceID**：系统内唯一标识（Edge/平台侧）
* **Instance ID**：BACnet 网络唯一标识
* **ObjectID**：对象标识（Type + Instance）
* **Property**：对象属性标识

当前设备清单（验收范围 ：设备点位不可串流）：

* bacnet-18 → Instance ID 2228318 ->Setpoint.1	AnalogValue 1	318.00 验证点
* bacnet-16 → Instance ID 2228316 ->Setpoint.1	AnalogValue 1	316.00 验证点
* bacnet-17 → Instance ID 2228317 ->Setpoint.1	AnalogValue 1	317.00 验证点
* Room_FC_2014_19 → Instance ID 2228319 ->Setpoint.1	AnalogValue 1	319.00 验证点

> ⚠ 验收前提：已确认所有设备物理运行正常且网络正常OK 