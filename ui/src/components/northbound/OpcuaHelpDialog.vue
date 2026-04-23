<template>
  <a-modal v-model:visible="visible" title="OPC UA 接入文档" :width="900" :footer="false" unmount-on-close>
    <a-tabs v-model:active-key="activeTab" type="line">
      <a-tab-pane key="connection" title="连接配置">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">连接配置 (Connection)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">使用 OPC UA 客户端（如 UaExpert, SCADA）连接到本网关。</p>
        </div>

        <a-card :bordered="true" style="margin-bottom: 16px; border-color: #0ea5e9">
          <div style="font-size: 12px; font-weight: 600; color: #0ea5e9; margin-bottom: 8px">Endpoint URL (服务地址)</div>
          <div style="display: flex; align-items: center; background: #f8fafc; padding: 8px; font-size: 13px; font-weight: 500">
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">opc.tcp://{{ host }}:{{ port }}{{ endpoint }}</span>
            <a-button type="text" size="mini" @click="copyToClipboard('opc.tcp://' + host + ':' + port + endpoint)">
              <template #icon><icon-copy :size="12" /></template>
            </a-button>
          </div>
          <div style="font-size: 12px; color: #6b7280; margin-top: 4px">提示：如果从外部访问，请将 localhost 替换为网关的实际 IP 地址。</div>
        </a-card>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">最佳实践 (Best Practices)</div>
        <a-alert type="success" style="margin-bottom: 16px">
          <div style="font-weight: 600; margin-bottom: 4px">推荐连接方式：</div>
          <ol style="margin: 0; padding-left: 20px">
            <li>安全策略选择：<strong>Basic256Sha256 - SignAndEncrypt</strong></li>
            <li>证书信任：首次连接时，如果客户端提示服务端证书不可信，请选择 <strong>"Trust" (信任)</strong>。</li>
          </ol>
        </a-alert>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">客户端指南</div>
        <a-collapse :default-active-key="['prosys']" style="margin-bottom: 16px">
          <a-collapse-item header="Prosys OPC UA Browser (推荐)" key="prosys">
            <p style="margin: 0 0 8px">功能强大的跨平台 OPC UA 客户端工具。</p>
            <div style="margin-bottom: 8px">
              <a href="https://downloads.prosysopc.com/opc-ua-browser-downloads.php" target="_blank">下载地址 (Download)</a>
            </div>
            <div style="background: #f8fafc; padding: 8px; border-radius: 0">
              <strong>连接步骤：</strong>
              <ol style="margin: 4px 0 0; padding-left: 20px">
                <li>输入 Endpoint URL (上文复制)。</li>
                <li>Security Mode 选择 <strong>SignAndEncrypt</strong>。</li>
                <li>Security Policy 选择 <strong>Basic256Sha256</strong>。</li>
                <li>点击 Connect，在弹出的证书警告中勾选 "Accept Permanently" 并确认。</li>
              </ol>
            </div>
          </a-collapse-item>
          <a-collapse-item header="Unified Automation UaExpert" key="uaexpert">
            <p style="margin: 0 0 8px">专业的 OPC UA 客户端。</p>
            <div style="background: #f8fafc; padding: 8px; border-radius: 0">
              <strong>连接步骤：</strong>
              <ol style="margin: 4px 0 0; padding-left: 20px">
                <li>添加 Server，双击 Custom Discovery 下的 URL。</li>
                <li>选择 <strong>Basic256Sha256 - SignAndEncrypt</strong> 策略。</li>
                <li>连接时点击 "Trust Server Certificate"。</li>
              </ol>
            </div>
          </a-collapse-item>
        </a-collapse>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">安全策略</div>
        <a-table :columns="securityColumns" :data="securityPolicies" size="small" :bordered="{ cell: true }" :pagination="false" />
      </a-tab-pane>

      <a-tab-pane key="auth" title="身份认证">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">身份认证 (Authentication)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">配置客户端连接时的身份验证方式。</p>
        </div>

        <a-collapse :default-active-key="['anonymous', 'username']">
          <a-collapse-item header="匿名登录 (Anonymous)" key="anonymous">
            <p style="margin: 0">如果配置中启用了匿名访问，客户端可以选择 <strong>Anonymous</strong> 方式登录，无需用户名和密码。</p>
            <a-alert type="warning" style="margin-top: 8px">注意：生产环境建议禁用匿名访问。</a-alert>
          </a-collapse-item>
          <a-collapse-item header="用户名/密码 (Username/Password)" key="username">
            <p style="margin: 0">客户端选择 <strong>Username</strong> 方式，并输入配置中预设的用户名和密码。</p>
          </a-collapse-item>
        </a-collapse>
      </a-tab-pane>

      <a-tab-pane key="subscription" title="数据订阅">
        <div style="margin-bottom: 16px">
          <h4 style="margin: 0 0 8px">数据订阅 (Subscription)</h4>
          <p style="color: #6b7280; margin: 0 0 16px">浏览地址空间并订阅点位数据。</p>
        </div>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">地址空间结构</div>
        <pre style="background: #f8fafc; padding: 12px; border-radius: 0; font-size: 13px; line-height: 1.5; border: 1px solid #e5e7eb; margin-bottom: 16px">Root
└── Objects
    └── DeviceName (设备名称)
        └── PointName (点位名称)</pre>

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">NodeID 格式</div>
        <p style="color: #6b7280; margin: 0 0 16px; font-size: 13px">点位 NodeID 通常采用 String 类型，格式为 <code>ns=2;s=DeviceName/PointName</code>。</p>

        <a-table :columns="nodeIdColumns" :data="nodeIdData" size="small" :bordered="{ cell: true }" :pagination="false" style="margin-bottom: 16px" />

        <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px">常见问题</div>
        <ul style="color: #6b7280; font-size: 13px; padding-left: 20px; margin: 0">
          <li style="margin-bottom: 4px">如果无法浏览到设备节点，请检查设备是否已在"设备管理"中添加并启用。</li>
          <li style="margin-bottom: 4px">如果读取值为 BadWaitingForInitialData，表示设备尚未采集到有效数据。</li>
          <li>客户端订阅间隔建议不低于设备采集周期的 1/2。</li>
        </ul>
      </a-tab-pane>
    </a-tabs>
  </a-modal>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { IconCopy } from '@arco-design/web-vue/es/icon'
import { showMessage } from '@/composables/useGlobalState'

const props = defineProps({
  visible: { type: Boolean, default: false },
  port: { type: Number, default: 4840 },
  endpoint: { type: String, default: '' }
})

const emit = defineEmits(['update:visible'])

const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})
const activeTab = ref('connection')
const host = ref('localhost')

onMounted(() => {
  host.value = window.location.host ? window.location.host.split(':')[0] : 'localhost'
})

const securityColumns = [
  { title: '策略', dataIndex: 'policy' },
  { title: '说明', dataIndex: 'desc' }
]

const securityPolicies = [
  { policy: 'None', desc: '不加密 (仅用于调试)' },
  { policy: 'Basic256Sha256', desc: '签名并加密 (推荐)' },
  { policy: 'Aes128_Sha256_RsaOaep', desc: '签名并加密' }
]

const nodeIdColumns = [
  { title: '属性', dataIndex: 'attr' },
  { title: '值', dataIndex: 'value' },
  { title: '说明', dataIndex: 'desc' }
]

const nodeIdData = [
  { attr: 'Namespace Index (ns)', value: '2', desc: '默认命名空间索引' },
  { attr: 'Identifier Type', value: 'String (s)', desc: '字符串标识符' },
  { attr: 'Identifier', value: 'Device/Point', desc: '设备名/点位名组合' }
]

const copyToClipboard = (text) => {
  navigator.clipboard.writeText(text).then(() => {
    showMessage('已复制到剪贴板', 'success')
  }).catch(() => {
    showMessage('复制失败', 'error')
  })
}
</script>

