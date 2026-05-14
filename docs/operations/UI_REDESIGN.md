# UI 改造说明 - 三级导航页面

## 📱 页面结构

### 第一级：采集通道页面
- 显示所有采集通道列表
- 以卡片网格形式展示
- 每个通道显示：ID、名称、协议、启用状态
- 点击通道进入设备列表

### 第二级：设备列表页面
- 显示该通道下的所有设备
- 以表格形式展示
- 每个设备显示：ID、名称、协议、状态、采集间隔
- 点击"查看点位"按钮进入点位数据页面
- 面包屑导航回到采集通道

### 第三级：点位数据页面
- 显示该设备的所有点位数据
- 以表格形式展示
- 每个点位显示：ID、数值、质量、时间戳
- 支持"写入"操作
- 面包屑导航支持回到采集通道或设备列表

---

## 🎯 核心功能

### 1. 多级导航
```javascript
// 视图状态管理
const currentView = ref('channels'); // 'channels', 'devices', 'points'
const selectedChannel = ref(null);   // 当前选中的通道
const selectedDevice = ref(null);    // 当前选中的设备
```

### 2. 数据获取

#### 采集通道列表
```javascript
async refreshChannels() {
    const response = await fetch('/api/devices');
    const data = await response.json();
    channelList.value = data;
}
```

#### 设备信息
```javascript
async selectChannel(channel) {
    const response = await fetch(`/api/devices/${channel.id}`);
    const data = await response.json();
    // 为该通道创建设备列表
}
```

#### 点位数据
```javascript
async refreshPointData() {
    const response = await fetch(`/api/devices/${selectedChannel.value.id}/points`);
    const data = await response.json();
    pointDataList.value = data;
}
```

### 3. WebSocket 实时更新
- 连接到 `/ws/values` 端点
- 实时接收点位数据更新
- 仅在点位数据页面更新数据

### 4. 面包屑导航
```
采集通道 / 设备列表 / 点位数据
  ↑         ↑
  可点击    可点击
```

---

## 🎨 界面改进

### 采集通道页面
- 网格卡片布局（responsive design）
- 卡片 hover 效果（上移+阴影）
- 显示通道基本信息
- 启用/禁用状态标签

### 设备列表页面
- 表格视图（批量查看）
- 返回按钮
- 快速操作按钮

### 点位数据页面
- 详细表格视图
- 实时数据更新
- 写入功能
- 格式化数值（小数点后2位）
- 格式化时间戳（本地时间）

---

## 📊 API 端点

| 功能 | 方法 | 端点 | 说明 |
|------|------|------|------|
| 获取通道列表 | GET | /api/devices | 返回所有设备（通道） |
| 获取设备详情 | GET | /api/devices/{id} | 返回单个设备详情 |
| 获取点位数据 | GET | /api/devices/{id}/points | 返回该设备的所有点位数据 |
| 写入点位 | POST | /api/write | 写入点位数值 |
| WebSocket | WS | /ws/values | 实时数据推送 |

---

## 🔄 导航流程

```
启动应用
  ↓
加载采集通道列表 (currentView = 'channels')
  ↓
用户点击通道
  ↓
加载该通道的设备信息 (currentView = 'devices')
  ↓
用户点击"查看点位"
  ↓
加载该设备的点位数据 (currentView = 'points')
  ↓
WebSocket 实时更新点位数据
  ↓
用户点击"返回" → 返回上一级
```

---

## 💾 数据模型

### Channel (采集通道)
```javascript
{
    id: "gateway-1",
    name: "Industrial Edge Gateway 1",
    protocol: "modbus-tcp",
    enable: true,
    interval: "2s",
    config: {...}
}
```

### Point (点位)
```javascript
{
    PointID: "dev1_temp",
    DeviceID: "gateway-1",
    Value: 25.5,
    Quality: "Good",
    TS: "2026-01-22T08:45:30Z"
}
```

---

## 🎯 使用场景

### 场景 1：查看所有采集通道
1. 打开应用
2. 第一页显示所有采集通道
3. 了解系统中有多少个采集通道

### 场景 2：查看特定通道的数据
1. 点击采集通道卡片
2. 进入该通道的设备列表
3. 点击设备查看其点位数据

### 场景 3：实时监控点位数据
1. 进入点位数据页面
2. 数据通过 WebSocket 实时更新
3. 支持手动刷新（刷新按钮）

### 场景 4：写入点位数值
1. 在点位数据页面点击"写入"
2. 输入新数值
3. 确认提交

---

## 🔧 技术细节

### Vue 3 Composition API
- 使用 `ref()` 管理响应式状态
- 使用 `reactive()` 管理复杂对象
- 在 `onMounted()` 钩子初始化数据

### Element Plus 组件
- `el-card`: 通道卡片
- `el-table`: 设备和点位数据表格
- `el-button`: 各类按钮
- `el-tag`: 状态标签
- `el-dialog`: 写入数值对话框
- `el-message`: 提示信息

### 网络通信
- Fetch API: REST 接口调用
- WebSocket: 实时数据推送

---

## 📝 文件变更

| 文件 | 变更 | 说明 |
|-----|------|------|
| ui/index.html | 完全重写 | 三级导航页面 |

### 主要改动
1. **视图管理**: 三个独立的页面视图
2. **数据结构**: 增加了通道和设备的状态管理
3. **API 调用**: 新增三个 API 调用函数
4. **样式改进**: 响应式网格布局、面包屑导航
5. **国际化**: 改用中文界面

---

## 🚀 使用方法

### 访问方式
```
http://localhost:8080
```

### 页面导航
1. **第一页（采集通道）**: 应用启动时显示
2. **第二页（设备列表）**: 点击采集通道卡片进入
3. **第三页（点位数据）**: 点击"查看点位"按钮进入

### 南向点位 FormatPoint 配置与快速验证

1. 在采集通道页面点击某设备的“点位管理”，进入点位列表。
2. 点击“新增点位”或编辑已有点位，打开点位配置对话框。
3. 在“协议模板”按钮中选择 BACnet/OPC UA 模板：
   - 模板覆盖布尔、整型、浮点、双字、字符串等常用类型。
   - 模板包含名称占位符、数据类型、单位、换算公式、默认值、读写权限和描述字段。
4. 根据实际寄存器长度填写“字节数”：
   - 1 字节：自动禁用 WordOrder；
   - 2 字节：可选 AB / BA；
   - 4/8 字节：可选 ABCD / BADC / CDAB / DCBA。
5. 在“解析类型”下拉框中选择与字节数匹配的数据类型：
   - 1 字节：BIT、UINT8、INT8、BCD8
   - 2 字节：UINT16、INT16、UINT16_SWAP、INT16_SWAP、BCD16、FLOAT16
   - 4 字节：UINT32、INT32、UINT32_SWAP、INT32_SWAP、FLOAT32、FLOAT32_SWAP、BCD32
   - 8 字节：UINT64、INT64、FLOAT64、FLOAT64_SWAP
6. 可在“读公式/写公式”中配置 FormatPoint 公式，实现寄存器值与工程值互转。
7. 点击“快速验证”按钮，输入原始十六进制报文：
   - 系统按当前字节数 + 字序 + 解析类型 + 读公式即时计算工程值；
   - 若填写了期望工程值，验证结果以绿色（通过）/红色（未通过）状态标识；
   - 验证通过后，可一键“保存为模板”，在本会话内复用。
8. 确认配置正确后保存点位，返回列表观察实时采集值。

> 运营演示建议：准备 2~3 个示例点位（如温度、电压、开关量），分别演示模板套用、字节数/字序切换和快速验证流程，并配合截图用于培训材料。

### 返回导航
- 点击面包屑中的上一级链接返回
- 点击"返回"按钮返回上一页

---

## ⚠️ 注意事项

1. **API 依赖**: 应用依赖后端提供的 `/api/devices`、`/api/devices/{id}` 等端点
2. **WebSocket 连接**: 确保后端支持 `/ws/values` 端点
3. **CORS**: 如果前后端分离，需要配置 CORS
4. **数据格式**: 点位数据需要包含 DeviceID 字段

---

## 📱 响应式设计

- 采集通道页面: 自动调整网格列数
- 设备列表页面: 表格自适应宽度
- 点位数据页面: 表格自适应宽度
- 所有页面: 在移动设备上也能正常显示

---

## 🎓 示例 API 响应

### GET /api/devices
```json
[
    {
        "id": "gateway-1",
        "name": "Industrial Edge Gateway 1",
        "protocol": "modbus-tcp",
        "enable": true,
        "interval": "2s"
    }
]
```

### GET /api/devices/{id}/points
```json
[
    {
        "PointID": "dev1_temp",
        "DeviceID": "gateway-1",
        "Value": 25.5,
        "Quality": "Good",
        "TS": "2026-01-22T08:45:30Z"
    }
]
```

---

## 🔗 相关文件

- [ui/index.html](ui/index.html) - 前端页面
- [internal/server/server.go](internal/server/server.go) - 后端 API 实现
- [COMPLETION_SUMMARY.md](COMPLETION_SUMMARY.md) - 项目总结

---

**更新时间**: 2026-01-22
**版本**: 2.0.0
**状态**: ✅ 完成
