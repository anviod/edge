BACnet 前端 Web UI 功能审查清单 必须满足这里的所有功能要求。

---

## 一、设备管理（Device Management）

### 1️⃣ 设备发现与注册

* 一键扫描 BACnet 设备（Who-Is）
* 展示字段：

  * DeviceInstance
  * 设备名称
  * Vendor / Model
  * IP / 网络号
  * 在线状态
* 支持：

  * 自动注册
  * 手动添加 / 编辑 / 删除设备

---

## 二、对象与点位管理（Object & Point Management）

### 2️⃣ 对象浏览（Object Explorer）

* 以树或表形式展示：

  * Device → Object Type → Object Instance
* 支持筛选：

  * 按对象类型（AI/AO/AV/BI/BO/BV）
  * 按点位名称 / 单位 / 描述

---

### 3️⃣ 点位详情页（Point Detail View）

展示并支持编辑：

* Object Identifier
* Object Name
* Present Value（实时）
* Units
* Status Flags
* Description
* Reliability
* Priority Array（如支持写入）

---

## 三、点位读写与控制（Read / Write / Control）

### 4️⃣ 点位控制能力

* 支持 AO / BO / AV / BV 写入：

  * 普通写入
  * 优先级写入（1~16）
  * 释放写入（NULL）
* 写入结果即时反馈（成功 / 失败 / 无权限）

---

## 四、网络与通信状态可视化

### 9️⃣ BACnet 通信状态面板

* 展示：

  * 在线设备数
  * 离线设备数
  * 当前通信速率
  * 错误统计（超时、拒绝、无响应）
* 网络异常可定位到设备与错误类型

---

## 五、自动建模与配置管理

### 🔟 自动点位建模管理

* 从 objectList 自动生成点位模型
* 支持：

  * 点位重命名
  * 标签 / 分组 / 区域归属
  * 模型编辑与重生成

---

### 1️⃣1️⃣ 批量操作能力

* 批量启停采集
* 批量修改采集周期
* 批量点位写入（可选）

---

## 八、系统运维与诊断

### 1️⃣2️⃣ 通信日志查看

* 查看 BACnet 请求/响应摘要
* 支持：

  * 按 DeviceID / ObjectID / 错误类型过滤
  * 下载日志

---

### 1️⃣3️⃣ 设备调试工具（推荐）

* 单点 ReadProperty / WriteProperty 测试
* Who-Is / I-Am 手动触发
* objectList 重新拉取

---

## 九、前端交付最低必过项（MVP）

| 模块   | 必须具备               |
| ---- | ------------------ |
| 设备管理 | 扫描、注册、状态显示         |
| 点位管理 | objectList 浏览、点位详情 |
| 实时数据 | Present_Value 实时显示 |
| 点位控制 | AO/BO/AV/BV 写入     |
| 日志   | 通信与操作日志            |

