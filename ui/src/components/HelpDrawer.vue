<template>
  <a-drawer
    :visible="visible"
    @cancel="onCancel"
    :width="400"
    :footer="false"
    unmount-on-close
    class="industrial-drawer"
    render-to-body
  >
    <template #title>
      <div class="drawer-header">
        <IconQuestionCircle class="text-slate-400" />
        <span class="ml-2">帮助文档 / DOCUMENTATION</span>
      </div>
    </template>

    <div class="help-content">
      <section class="help-section">
        <h3 class="section-title">快速说明</h3>
        <p class="section-desc">本文档涵盖了当前 {{ channelProtocol }} 协议下的点位配置规范。</p>
      </section>

      <!-- Modbus 协议帮助 -->
      <template v-if="channelProtocol.includes('modbus')">
        <section class="help-section">
          <h3 class="section-title">协议介绍</h3>
          <p class="section-desc">Modbus 是一种串行通信协议，主要用于工业自动化领域。它支持多种传输介质，包括 RS-232、RS-485 和 TCP/IP。</p>
        </section>

        <section class="help-section">
          <h3 class="section-title">寄存器地址规范</h3>
          <div class="spec-card">
            <div class="spec-item">
              <span class="label">保持寄存器 (Holding)</span>
              <span class="value font-mono">40001 - 49999</span>
            </div>
            <div class="spec-item">
              <span class="label">只读寄存器 (Input)</span>
              <span class="value font-mono">30001 - 39999</span>
            </div>
            <div class="spec-item">
              <span class="label">线圈 (Coil)</span>
              <span class="value font-mono">00001 - 09999</span>
            </div>
            <div class="spec-item">
              <span class="label">离散输入 (Discrete Input)</span>
              <span class="value font-mono">10001 - 19999</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h3 class="section-title">数据格式</h3>
          <div class="spec-card">
            <div class="spec-item">
              <span class="label">16位整数</span>
              <span class="value font-mono">int16, uint16</span>
            </div>
            <div class="spec-item">
              <span class="label">32位整数</span>
              <span class="value font-mono">int32, uint32</span>
            </div>
            <div class="spec-item">
              <span class="label">浮点数</span>
              <span class="value font-mono">float32, float64</span>
            </div>
            <div class="spec-item">
              <span class="label">字符串</span>
              <span class="value font-mono">string</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h3 class="section-title">点位转换</h3>
          <p class="section-desc">Modbus 寄存器地址通常需要转换为实际的内存地址。例如，保持寄存器 40001 对应的实际地址是 0。</p>
        </section>

        <section class="help-section">
          <h3 class="section-title">常见问题 FAQ</h3>
          <a-collapse :bordered="false" expand-icon-position="right">
            <a-collapse-item header="为什么数值显示为 0?" key="1">
              请检查设备通讯状态及寄存器偏移量（Offset）设置是否正确。
            </a-collapse-item>
            <a-collapse-item header="如何配置浮点数 (Float32)?" key="2">
              需要占用两个连续的 16 位寄存器，并确认字节序（Endianness）。
            </a-collapse-item>
            <a-collapse-item header="如何设置寄存器地址?" key="3">
              对于保持寄存器，输入 40001 表示第一个寄存器，系统会自动转换为实际地址 0。
            </a-collapse-item>
          </a-collapse>
        </section>
      </template>

      <!-- BACnet 协议帮助 -->
      <template v-else-if="channelProtocol.includes('bacnet')">
        <section class="help-section">
          <h3 class="section-title">协议介绍</h3>
          <p class="section-desc">BACnet 是建筑自动化和控制网络协议，主要用于智能建筑和 HVAC 系统。它支持多种网络技术，包括 Ethernet、BACnet/IP 和 MS/TP。</p>
        </section>

        <section class="help-section">
          <h3 class="section-title">对象类型</h3>
          <div class="spec-card">
            <div class="spec-item">
              <span class="label">模拟输入</span>
              <span class="value font-mono">AnalogInput</span>
            </div>
            <div class="spec-item">
              <span class="label">模拟输出</span>
              <span class="value font-mono">AnalogOutput</span>
            </div>
            <div class="spec-item">
              <span class="label">模拟值</span>
              <span class="value font-mono">AnalogValue</span>
            </div>
            <div class="spec-item">
              <span class="label">二进制输入</span>
              <span class="value font-mono">BinaryInput</span>
            </div>
            <div class="spec-item">
              <span class="label">二进制输出</span>
              <span class="value font-mono">BinaryOutput</span>
            </div>
            <div class="spec-item">
              <span class="label">二进制值</span>
              <span class="value font-mono">BinaryValue</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h3 class="section-title">数据格式</h3>
          <div class="spec-card">
            <div class="spec-item">
              <span class="label">整数</span>
              <span class="value font-mono">int</span>
            </div>
            <div class="spec-item">
              <span class="label">浮点数</span>
              <span class="value font-mono">real</span>
            </div>
            <div class="spec-item">
              <span class="label">字符串</span>
              <span class="value font-mono">characterstring</span>
            </div>
            <div class="spec-item">
              <span class="label">布尔值</span>
              <span class="value font-mono">boolean</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h3 class="section-title">点位格式</h3>
          <p class="section-desc">BACnet 点位地址格式为：对象类型:实例号，例如 AnalogInput:1。</p>
        </section>

        <section class="help-section">
          <h3 class="section-title">常见问题 FAQ</h3>
          <a-collapse :bordered="false" expand-icon-position="right">
            <a-collapse-item header="如何确定对象实例号?" key="1">
              实例号是设备中对象的唯一标识符，通常由设备制造商分配或在设备配置中设置。
            </a-collapse-item>
            <a-collapse-item header="如何设置优先级?" key="2">
              BACnet 支持 1-16 级优先级，1 为最高，16 为最低。设置为 NULL 表示释放该点位。
            </a-collapse-item>
            <a-collapse-item header="为什么无法写入值?" key="3">
              请检查对象类型是否支持写入操作，例如 AnalogInput 通常为只读。
            </a-collapse-item>
          </a-collapse>
        </section>
      </template>

      <!-- OPC UA 协议帮助 -->
      <template v-else-if="channelProtocol.includes('opc-ua')">
        <section class="help-section">
          <h3 class="section-title">协议介绍</h3>
          <p class="section-desc">OPC UA 是一种工业通信协议，提供安全、可靠的工业数据交换。它支持多种传输协议，包括 TCP/IP、HTTPS 和 WebSocket。</p>
        </section>

        <section class="help-section">
          <h3 class="section-title">节点类型</h3>
          <div class="spec-card">
            <div class="spec-item">
              <span class="label">变量</span>
              <span class="value font-mono">Variable</span>
            </div>
            <div class="spec-item">
              <span class="label">对象</span>
              <span class="value font-mono">Object</span>
            </div>
            <div class="spec-item">
              <span class="label">方法</span>
              <span class="value font-mono">Method</span>
            </div>
            <div class="spec-item">
              <span class="label">引用</span>
              <span class="value font-mono">Reference</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h3 class="section-title">数据格式</h3>
          <div class="spec-card">
            <div class="spec-item">
              <span class="label">布尔值</span>
              <span class="value font-mono">Boolean</span>
            </div>
            <div class="spec-item">
              <span class="label">整数</span>
              <span class="value font-mono">Int16, Int32, Int64</span>
            </div>
            <div class="spec-item">
              <span class="label">浮点数</span>
              <span class="value font-mono">Float, Double</span>
            </div>
            <div class="spec-item">
              <span class="label">字符串</span>
              <span class="value font-mono">String</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h3 class="section-title">点位格式</h3>
          <p class="section-desc">OPC UA 点位地址使用 NodeID，格式为 ns= namespace;s= nodeid 或 ns= namespace;i= nodeid。</p>
        </section>

        <section class="help-section">
          <h3 class="section-title">常见问题 FAQ</h3>
          <a-collapse :bordered="false" expand-icon-position="right">
            <a-collapse-item header="如何找到节点的 NodeID?" key="1">
              使用 OPC UA 客户端工具浏览设备的地址空间，找到目标节点并复制其 NodeID。
            </a-collapse-item>
            <a-collapse-item header="为什么无法连接到 OPC UA 服务器?" key="2">
              请检查端点 URL 是否正确，以及服务器是否运行并接受连接。
            </a-collapse-item>
            <a-collapse-item header="如何处理不同的数据类型?" key="3">
              OPC UA 支持自动数据类型转换，但建议在配置时选择与设备一致的数据类型。
            </a-collapse-item>
          </a-collapse>
        </section>
      </template>

      <!-- 通用帮助 -->
      <template v-else>
        <section class="help-section">
          <h3 class="section-title">协议介绍</h3>
          <p class="section-desc">当前协议为 {{ channelProtocol }}。请参考设备手册了解详细的配置规范。</p>
        </section>

        <section class="help-section">
          <h3 class="section-title">数据格式</h3>
          <div class="spec-card">
            <div class="spec-item">
              <span class="label">整数</span>
              <span class="value font-mono">int, uint</span>
            </div>
            <div class="spec-item">
              <span class="label">浮点数</span>
              <span class="value font-mono">float, double</span>
            </div>
            <div class="spec-item">
              <span class="label">字符串</span>
              <span class="value font-mono">string</span>
            </div>
            <div class="spec-item">
              <span class="label">布尔值</span>
              <span class="value font-mono">bool</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h3 class="section-title">常见问题 FAQ</h3>
          <a-collapse :bordered="false" expand-icon-position="right">
            <a-collapse-item header="为什么数值显示为 0?" key="1">
              请检查设备通讯状态及配置参数是否正确。
            </a-collapse-item>
            <a-collapse-item header="如何配置数据类型?" key="2">
              根据设备支持的数据类型选择合适的类型，确保与设备端一致。
            </a-collapse-item>
            <a-collapse-item header="如何排查通讯问题?" key="3">
              检查网络连接、设备电源、配置参数和设备状态。
            </a-collapse-item>
          </a-collapse>
        </section>
      </template>
    </div>
  </a-drawer>
</template>

<script setup>
import { IconQuestionCircle } from '@arco-design/web-vue/es/icon'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  channelProtocol: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:visible', 'cancel'])

const onCancel = () => {
  emit('update:visible', false)
  emit('cancel')
}
</script>

<style scoped>
/* 抽屉整体风格 */
.industrial-drawer :deep(.arco-drawer) {
  border-left: 1px solid #e5e7eb;
  border-radius: 0;
  box-shadow: -4px 0 12px rgba(0, 0, 0, 0.03); /* 极轻微阴影 */
}

.industrial-drawer :deep(.arco-drawer-header) {
  border-bottom: 1px solid #f1f5f9;
  height: 50px;
}

.drawer-header {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
  letter-spacing: 0.5px;
  display: flex;
  align-items: center;
}

/* 内容排版 */
.help-content {
  padding: 16px;
  font-family: 'Inter', 'JetBrains Mono', 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

.help-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
  margin-bottom: 12px;
  text-transform: uppercase;
  border-left: 3px solid #334155;
  padding-left: 8px;
}

.section-desc {
  font-size: 13px;
  color: #475569;
  line-height: 1.6;
}

/* 规格卡片：硬核线构 */
.spec-card {
  border: 1px solid #e2e8f0;
  background-color: #f8fafc;
  padding: 12px;
}

.spec-item {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  font-size: 12px;
}

.spec-item:not(:last-child) {
  border-bottom: 1px dashed #cbd5e1;
}

.spec-item .label { color: #64748b; }
.spec-item .value { color: #0f172a; font-weight: 600; }

/* 折叠面板美化 */
:deep(.arco-collapse-item) {
  border-bottom: 1px solid #f1f5f9 !important;
}

:deep(.arco-collapse-item-header) {
  font-size: 13px !important;
  background: transparent !important;
}

:deep(.arco-collapse-item-content) {
  font-size: 12px;
  color: #475569;
  line-height: 1.5;
  padding: 8px 16px 16px !important;
}
</style>
