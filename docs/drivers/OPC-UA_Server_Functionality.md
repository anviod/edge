任务一：基础能力建设

目标：引入 OPC UA Server 库，完成基本服务启动与地址空间骨架。

任务：

引入依赖：

github.com/awcullen/opcua

在 go.mod 中锁定版本并完成编译验证。

新建模块：

internal/northbound/opcua/

定义 Server、AddressSpaceBuilder、WriteHandler 等核心结构。

实现 OPC UA Server 启动与监听（支持端口配置）。

创建基础 Address Space：

Objects

Gateway

Gateway/Info（CPU、内存、Uptime 等占位节点）

验收标准：

网关启动后 OPC UA Server 可监听端口并可被 UA Expert 等客户端发现并连接。

基础节点结构可浏览。

任务二：南向模型映射

目标：将 Channel / Device / Point 模型动态映射为 OPC UA 节点树。

任务：

扩展 ChannelManager 接口：

GetAllChannels()

GetDevices(channelID)

GetPoints(channelID, deviceID)

实现 AddressSpace 动态构建逻辑：

创建 Channels / Devices / Points 层级结构

为每个 Point 创建 VariableNode，设置数据类型、访问权限、初始值

定义统一 NodeID 命名规范并固化实现。

验收标准：

启动后 OPC UA 客户端可看到完整设备拓扑结构。

点位数量、结构与 Web UI / CLI 展示一致。

任务三：数据同步机制

目标：实现南向采集数据实时更新到 OPC UA Server，并支持订阅。

任务：

在 NorthboundManager 中订阅 ValueBus / 数据管道。

实现 Update(value model.Value) 方法：

查找对应 VariableNode

更新 Value、Timestamp、Quality

验证 Subscription / MonitoredItem 功能：

客户端订阅点位后，值变化可自动推送。

验收标准：

南向设备数据变化 → OPC UA 客户端实时更新。

Subscription 延迟、稳定性满足工业使用要求。

任务四：写控制通路实现

目标：实现 OPC UA → 南向设备的写控制能力。

任务：

实现 OPC UA WriteHandler：

解析 NodeID → ChannelID / DeviceID / PointID

调用 ChannelManager.WritePoint(...)

实现写入结果标准化映射：

成功 → Good

失败 → 映射为标准 OPC UA StatusCode

对写操作进行日志记录（审计用途）。

验收标准：

OPC UA 客户端对点位写值可成功下发到真实设备。

写失败可返回明确错误码并记录日志。

任务五：安全、运维与配置完善

目标：使 OPC UA Server 达到生产可用级别。

任务：

增加安全配置支持：

- 支持多种认证方式：匿名 (Anonymous)、用户名密码 (UserName)、证书认证 (Certificate)
- 支持配置用户列表 (Username/Password)
- 支持配置服务器证书与私钥路径 (CertFile/KeyFile)
- UI 界面集成安全配置选项 (Northbound.vue)
- 后端 Server 实现基于配置的认证策略 (server.go)

增加运维监控接口：

当前连接客户端数量

当前订阅数量

最近写操作统计

扩展配置系统：

opcua.server.enable

opcua.server.port

opcua.server.securityPolicy

opcua.server.users

扩展 CLI 工具：

gateway opcua status

gateway opcua cert generate

gateway opcua restart

验收标准：

支持开启/关闭 OPC UA Server。

支持至少一种安全策略与用户认证方式。

管理员可通过 CLI 查看 OPC UA Server 运行状态。

任务六：测试与交付

目标：形成可交付、可验收、可推广的 OPC UA Server 能力。

任务：

编写测试方案：

地址空间结构验证

数据更新正确性验证

写控制验证

异常场景验证（设备离线、写失败、网络中断）

使用工具验证：

UA Expert

Ignition

Kepware（如可用）

输出文档：

OPC UA 地址空间模型说明书

NodeID 命名规范

写控制接口规范

安全与部署说明

验收标准：

通过所有功能测试与稳定性测试。

文档完整，可交付给客户或集成商使用。