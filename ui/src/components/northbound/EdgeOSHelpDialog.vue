<template>
  <a-modal
    v-model:visible="visible"
    title="edgeOS 通信协议帮助"
    width="800px"
    @cancel="handleCancel"
  >
    <template #icon>
      <icon-question-circle style="font-size: 24px; color: rgb(var(--primary-6))" />
    </template>
    <a-tabs default-active-key="mqtt">
      <a-tab-pane key="mqtt" title="edgeOS(MQTT)">
        <div class="help-content">
          <h3>概述</h3>
          <p>edgeOS(MQTT) 是基于 MQTT 协议的北向通信通道，用于将 EdgeX 网关的数据上报到 edgeOS 蜂群网络。</p>

          <h3>主要特性</h3>
          <ul>
            <li>支持 MQTT 3.1.1 协议</li>
            <li>双向通信：数据上报 + 命令接收</li>
            <li>节点注册与心跳</li>
            <li>设备状态上报</li>
            <li>设备发现命令</li>
            <li>写入命令支持</li>
          </ul>

          <h3>消息主题 (Topics)</h3>
          <table class="topic-table">
            <thead>
              <tr>
                <th>Topic</th>
                <th>方向</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>edgex/nodes/register</code></td>
                <td>EdgeX → EdgeOS</td>
                <td>节点注册</td>
              </tr>
              <tr>
                <td><code>edgex/data/{node_id}/{device_id}</code></td>
                <td>EdgeX → EdgeOS</td>
                <td>实时数据上报</td>
              </tr>
              <tr>
                <td><code>edgex/nodes/{node_id}/heartbeat</code></td>
                <td>EdgeX → EdgeOS</td>
                <td>节点心跳</td>
              </tr>
              <tr>
                <td><code>edgex/cmd/{node_id}/discover</code></td>
                <td>EdgeOS → EdgeX</td>
                <td>设备发现命令</td>
              </tr>
              <tr>
                <td><code>edgex/cmd/{node_id}/{device_id}/write</code></td>
                <td>EdgeOS → EdgeX</td>
                <td>写入设备数据</td>
              </tr>
            </tbody>
          </table>

          <h3>消息格式</h3>
          <pre><code>{
  "header": {
    "message_id": "msg-001",
    "timestamp": 1744680000000,
    "source": "edgex-node-001",
    "message_type": "data",
    "version": "1.0"
  },
  "body": {
    "node_id": "edgex-node-001",
    "device_id": "device-001",
    "timestamp": 1744680000000,
    "points": {
      "Temperature": 25.5,
      "Humidity": 65.2
    },
    "quality": "good"
  }
}</code></pre>
        </div>
      </a-tab-pane>

      <a-tab-pane key="nats" title="edgeOS(NATS)">
        <div class="help-content">
          <h3>概述</h3>
          <p>edgeOS(NATS) 是基于 NATS 协议的北向通信通道，提供高性能的消息传递和 JetStream 持久化支持。</p>

          <h3>主要特性</h3>
          <ul>
            <li>支持 NATS 2.x 协议</li>
            <li>JetStream 消息持久化</li>
            <li>请求/响应模式</li>
            <li>Subject 通配符支持</li>
            <li>高吞吐、低延迟</li>
            <li>自动重连机制</li>
          </ul>

          <h3>消息主题 (Subjects)</h3>
          <table class="topic-table">
            <thead>
              <tr>
                <th>Subject</th>
                <th>方向</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>edgex.nodes.register</code></td>
                <td>EdgeX → EdgeOS</td>
                <td>节点注册</td>
              </tr>
              <tr>
                <td><code>edgex.data.{node_id}.{device_id}</code></td>
                <td>EdgeX → EdgeOS</td>
                <td>实时数据上报</td>
              </tr>
              <tr>
                <td><code>edgex.nodes.{node_id}.heartbeat</code></td>
                <td>EdgeX → EdgeOS</td>
                <td>节点心跳</td>
              </tr>
              <tr>
                <td><code>edgex.cmd.{node_id}.discover</code></td>
                <td>EdgeOS → EdgeX</td>
                <td>设备发现命令</td>
              </tr>
              <tr>
                <td><code>edgex.cmd.{node_id}.{device_id}.write</code></td>
                <td>EdgeOS → EdgeX</td>
                <td>写入设备数据</td>
              </tr>
            </tbody>
          </table>

          <h3>通配符</h3>
          <ul>
            <li><code>*</code> - 匹配单个 token</li>
            <li><code>></code> - 匹配一个或多个 tokens</li>
          </ul>

          <h3>示例</h3>
          <pre><code>// 订阅所有设备的写入命令
edgex.cmd.edgex-node-001.*.write

// 订阅所有数据上报
edgex.data.>></code></pre>
        </div>
      </a-tab-pane>

      <a-tab-pane key="config" title="配置说明">
        <div class="help-content">
          <h3>edgeOS(MQTT) 配置项</h3>
          <table class="config-table">
            <thead>
              <tr>
                <th>配置项</th>
                <th>类型</th>
                <th>说明</th>
                <th>默认值</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>broker</td>
                <td>string</td>
                <td>MQTT Broker 地址</td>
                <td>tcp://127.0.0.1:1883</td>
              </tr>
              <tr>
                <td>client_id</td>
                <td>string</td>
                <td>MQTT 客户端 ID</td>
                <td>edgex-node-001</td>
              </tr>
              <tr>
                <td>node_id</td>
                <td>string</td>
                <td>节点唯一标识</td>
                <td>edgex-node-001</td>
              </tr>
              <tr>
                <td>qos</td>
                <td>byte</td>
                <td>QoS 级别 (0/1/2)</td>
                <td>1</td>
              </tr>
              <tr>
                <td>keep_alive</td>
                <td>int</td>
                <td>心跳间隔(秒)</td>
                <td>60</td>
              </tr>
            </tbody>
          </table>

          <h3>edgeOS(NATS) 配置项</h3>
          <table class="config-table">
            <thead>
              <tr>
                <th>配置项</th>
                <th>类型</th>
                <th>说明</th>
                <th>默认值</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>url</td>
                <td>string</td>
                <td>NATS 服务器 URL</td>
                <td>nats://127.0.0.1:4222</td>
              </tr>
              <tr>
                <td>client_id</td>
                <td>string</td>
                <td>NATS 客户端名称</td>
                <td>edgex-node-001</td>
              </tr>
              <tr>
                <td>jetstream_enabled</td>
                <td>bool</td>
                <td>是否启用 JetStream</td>
                <td>false</td>
              </tr>
              <tr>
                <td>max_reconnects</td>
                <td>int</td>
                <td>最大重连次数</td>
                <td>10</td>
              </tr>
            </tbody>
          </table>
        </div>
      </a-tab-pane>
    </a-tabs>
  </a-modal>
</template>

<script setup>
import { computed } from 'vue'
import { IconQuestionCircle } from '@arco-design/web-vue/es/icon'

const props = defineProps({
  visible: Boolean
})

const emit = defineEmits(['update:visible'])

const visible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value)
})

const handleCancel = () => {
  emit('update:visible', false)
}
</script>

<style scoped>
.help-content {
  padding: 16px;
}

.help-content h3 {
  font-size: 16px;
  font-weight: 600;
  margin-top: 16px;
  margin-bottom: 8px;
  color: #1e293b;
}

.help-content h3:first-child {
  margin-top: 0;
}

.help-content p {
  color: #64748b;
  line-height: 1.6;
  margin-bottom: 12px;
}

.help-content ul {
  padding-left: 20px;
  color: #64748b;
}

.help-content ul li {
  margin-bottom: 8px;
}

.help-content pre {
  background: #f1f5f9;
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.5;
  color: #334155;
  margin: 12px 0;
}

.help-content code {
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #334155;
}

.topic-table,
.config-table {
  width: 100%;
  border-collapse: collapse;
  margin: 16px 0;
}

.topic-table th,
.topic-table td,
.config-table th,
.config-table td {
  border: 1px solid #e2e8f0;
  padding: 10px 12px;
  text-align: left;
  font-size: 13px;
}

.topic-table th,
.config-table th {
  background: #f8fafc;
  font-weight: 600;
  color: #1e293b;
}

.topic-table tr:nth-child(even),
.config-table tr:nth-child(even) {
  background: #f8fafc;
}

.topic-table tr:hover,
.config-table tr:hover {
  background: #e2e8f0;
}
</style>
