在现有 Windows 实现基础上，补全五项能力：事务回滚、静态路由 UI、Linux Adapter、网络连通性验证、IPv6 支持。

# 网络模块技术设计说明（Linux 适配与能力增强版）

## 1. 设计目标

在现有 Windows 网络配置能力基础上，扩展并增强系统网络模块，目标包括：

1. 支持 **Linux 平台网络配置**（主流发行版通用）。
2. 引入 **事务化网络变更机制**，保障配置失败可自动回滚。
3. 提供 **静态路由可视化管理能力（UI + API）**。
4. 增加 **网络连通性验证机制**，作为配置生效与否的判定标准。
5. 引入 **IPv6 配置与路由支持**，满足未来网络环境演进需求。

---

## 2. 当前能力评估与增强目标

| 项目            | 当前实现  | 增强目标              |
| ------------- | ----- | ----------------- |
| 事务回滚          | ❌ 未实现 | 引入配置事务与自动回滚       |
| 静态路由管理 UI     | ❌ 未完成 | 增加路由 CRUD 与 UI 管理 |
| Linux Adapter | ❌ 未实现 | 按统一接口设计实现         |
| 网络连通性验证       | ❌ 未实现 | 增加可配置验证策略         |
| IPv6 支持       | ❌ 未实现 | IPv4/IPv6 双栈支持    |

---

## 3. 总体架构设计

### 3.1 分层结构

```
UI层 (Vue)
   ↓
API层 (SystemHandler)
   ↓
业务层 (SystemManager / NetworkManager)
   ↓
适配层 (NetworkAdapter Interface)
   ↓
平台实现 (WindowsAdapter / LinuxAdapter)
   ↓
系统命令 / 网络栈 (netsh / ip / nmcli / systemd-networkd)
```

### 3.2 统一 Adapter 接口设计（核心）

```go
type NetworkAdapter interface {
    GetInterfaces() ([]NetworkInterface, error)
    ApplyInterfaceConfig(iface NetworkInterface) error

    GetRoutes() ([]RouteEntry, error)
    ApplyRoutes(routes []RouteEntry) error

    ValidateConnectivity(targets []ConnectivityTarget) (ConnectivityReport, error)
}
```

---

## 4. Linux Adapter 设计（核心增强）

### 4.1 支持的网络管理方式

优先支持顺序：

1. `ip` + `route`（通用 Linux）
2. `nmcli`（NetworkManager 环境）
3. `systemd-networkd`（嵌入式 / 服务器环境）

适配器初始化时自动检测可用工具。

---

### 4.2 接口发现实现

| 功能       | 实现命令                                      |
| -------- | ----------------------------------------- |
| 获取接口列表   | `ip link show`                            |
| 获取 IP 地址 | `ip addr show <iface>`                    |
| 获取网关     | `ip route show default`                   |
| DHCP 状态  | 通过 NetworkManager / systemd-networkd 配置判断 |
| Metric   | `ip route show` 中 metric 字段               |

---

### 4.3 网络配置应用实现

#### 静态 IP：

```bash
ip addr add 192.168.1.10/24 dev eth0
ip link set eth0 up
ip route replace default via 192.168.1.1 dev eth0 metric 100
```

#### DHCP：

```bash
dhclient eth0
```

或：

```bash
nmcli con mod <conn> ipv4.method auto
nmcli con up <conn>
```

---

## 5. 事务回滚机制设计

### 5.1 设计目标

* 确保网络配置变更 **原子性**
* 如果变更后网络不可达，自动恢复到变更前状态
* 支持配置链路断开后的自动恢复（防止设备“失联”）

---

### 5.2 实现机制

#### 事务模型：

```go
type NetworkTransaction struct {
    Before NetworkSnapshot
    After  NetworkSnapshot
    Status TransactionStatus
}
```

#### 执行流程：

1. 执行前：

   * 获取当前接口配置、路由配置，生成 `Before Snapshot`
2. 应用新配置：

   * 执行 `ApplyInterfaceConfig` / `ApplyRoutes`
3. 验证连通性：

   * 调用 `ValidateConnectivity`
4. 若验证失败：

   * 自动回滚到 `Before Snapshot`
   * 标记事务失败并输出错误日志
5. 若成功：

   * 标记事务成功并提交配置

---

### 5.3 回滚机制

* Windows：通过 `netsh` 反向重放配置
* Linux：通过 `ip`/`nmcli` 反向应用旧配置

---

## 6. 静态路由管理设计（API + UI）

### 6.1 数据模型

```go
type RouteEntry struct {
    Destination string // 例如 192.168.100.0/24 或 2001:db8::/64
    Gateway     string
    Interface   string
    Metric      int
    Protocol    string // static / dhcp / kernel
}
```

---

### 6.2 API 设计

| 方法     | 路径                         | 说明      |
| ------ | -------------------------- | ------- |
| GET    | /api/system/network/routes | 查询当前路由表 |
| POST   | /api/system/network/routes | 添加静态路由  |
| PUT    | /api/system/network/routes | 修改静态路由  |
| DELETE | /api/system/network/routes | 删除静态路由  |

---

### 6.3 UI 设计要点

* 表格展示当前路由（IPv4/IPv6）
* 支持新增/编辑/删除
* 提供 Metric 输入与接口选择
* 应用配置时自动走事务 + 连通性验证流程

---

## 7. 网络连通性验证设计

### 7.1 验证目标

在网络配置变更后，确保至少满足以下条件之一：

* 默认网关可达
* 管理服务器可达（例如平台地址）
* DNS 可解析
* 自定义探测目标可达

---

### 7.2 验证模型

```go
type ConnectivityTarget struct {
    Type   string // gateway | ip | domain | http
    Target string // 192.168.1.1 | www.baidu.com | http://example.com/health
    Timeout int
}
```

---

### 7.3 验证方式

| 类型           | 方法                 |
| ------------ | ------------------ |
| gateway / ip | ICMP Ping          |
| domain       | DNS resolve + ping |
| http         | HTTP GET/HEAD 请求   |

返回：

```go
type ConnectivityReport struct {
    Success bool
    Details []ConnectivityResult
}
```

---

## 8. IPv6 支持设计

### 8.1 接口模型扩展

```go
type IPAddress struct {
    Address string // 支持 IPv4 / IPv6
    Prefix  int
    Version int // 4 or 6
}
```

NetworkInterface 增加：

* IPv6Addresses []IPAddress
* IPv6Gateway string

---

### 8.2 Linux 实现示例

```bash
ip -6 addr add 2001:db8::10/64 dev eth0
ip -6 route add default via 2001:db8::1 dev eth0 metric 100
```

---

### 8.3 UI 扩展

* IP 输入支持 IPv4 / IPv6 格式校验
* 网关字段支持 IPv6
* 路由页面支持 IPv6 目的网段

---

## 9. 风险与控制点

| 风险           | 控制策略                  |
| ------------ | --------------------- |
| 网络配置失败导致设备失联 | 强制启用事务 + 自动回滚         |
| Linux 发行版差异  | Adapter 内部工具检测与策略分支   |
| IPv6 兼容性问题   | 可配置开关 + 双栈测试覆盖        |
| 路由误配置        | UI 层校验 + 后端验证 + 连通性检测 |

---

## 10. 实施阶段规划（建议）

### 第一阶段：Linux Adapter 基础能力

* 接口发现
* IP/DHCP/网关配置
* 路由读取与应用

### 第二阶段：事务与连通性验证

* 引入 NetworkTransaction
* 实现 ValidateConnectivity
* 自动回滚机制上线

### 第三阶段：路由 UI + IPv6 支持

* 静态路由管理 API + UI
* IPv6 地址与路由支持
* 双栈验证与测试

---

## 11. 验证与测试建议

### 功能测试

* 静态 IP / DHCP 切换
* 路由增删改
* IPv6 地址配置

### 稳定性测试

* 错误配置场景回滚
* 网络中断后恢复
* 多接口、多路由场景

### 平台兼容性测试

* Ubuntu / Debian / CentOS / OpenWrt / Yocto

---

## 12. 是否建议按此方案推进？

**是，强烈建议。**

理由：

1. 与你当前 Windows 实现结构完全兼容（Adapter 抽象层不变）。
2. 引入事务与验证后，系统将具备工程级安全性（避免设备失联）。
3. 路由 UI + IPv6 将使产品具备完整网络管理能力，满足政企/工业场景要求。
4. Linux Adapter 的设计具备长期可维护性，适配嵌入式与服务器环境。
