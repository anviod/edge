<template>
  <div class="edge-compute-container">
    <a-tabs v-model:active-key="activeTab" class="mb-4">
      <a-tab-pane key="time" title="时间同步" />
      <a-tab-pane key="network" title="网络接口" />
      <a-tab-pane key="routes" title="静态路由" />
      <a-tab-pane key="ha" title="高可用集群" />
      <a-tab-pane key="hostname" title="主机设置" />
      <a-tab-pane key="ldap" title="LDAP 认证" />
      <a-tab-pane key="status" title="系统维护" />
    </a-tabs>

    <!-- 时间同步 -->
    <TimeSettings 
      v-if="activeTab === 'time'" 
      v-model="timeConfig"
      @save="saveConfig"
    />

    <!-- 网络接口和静态路由 -->
    <NetworkSettings 
      v-if="activeTab === 'network' || activeTab === 'routes'" 
      :active-tab="activeTab"
      :network-interfaces="networkInterfaces"
      :connectivity-targets="connectivityTargets"
      :static-routes="staticRoutes"
      @update:network-interfaces="(value) => networkInterfaces = value"
      @update:connectivity-targets="(value) => connectivityTargets = value"
      @update:static-routes="(value) => staticRoutes = value"
      @save="saveConfig"
    />

    <!-- 高可用集群 -->
    <HASettings 
      v-if="activeTab === 'ha'" 
      v-model="haConfig"
      @save="saveConfig"
    />

    <!-- 主机设置 -->
    <HostnameSettings 
      v-if="activeTab === 'hostname'" 
      v-model="hostnameConfig"
      :network-interfaces="networkInterfaces"
      @save="saveConfig"
    />

    <!-- LDAP 认证 -->
    <LDAPSettings 
      v-if="activeTab === 'ldap'" 
      v-model="ldapConfig"
      @save="saveConfig"
    />

    <!-- 系统维护 -->
    <SystemMaintenance v-if="activeTab === 'status'" />
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'

// 导入组件
import TimeSettings from './components/TimeSettings.vue'
import NetworkSettings from './components/NetworkSettings.vue'
import HASettings from './components/HASettings.vue'
import HostnameSettings from './components/HostnameSettings.vue'
import LDAPSettings from './components/LDAPSettings.vue'
import SystemMaintenance from './components/SystemMaintenance.vue'

const activeTab = ref('time')

// Time Settings Data
const timeConfig = reactive({
  mode: 'manual',
  manual: {
    datetime: '',
    timezone: 'Asia/Shanghai',
    sync_rtc: true
  },
  ntp: {
    servers: ['pool.ntp.org'],
    interval: 1,
    enabled: true
  }
})

// Network Settings Data
const networkInterfaces = ref([])
const connectivityTargets = ref([])

// Routes Data
const staticRoutes = ref([])

// HA Config
const haConfig = reactive({
  role: 'master',
  heartbeat_type: 'UDP',
  interval: 2,
  timeout: 5,
  retries: 3
})

// Hostname Config
const hostnameConfig = reactive({
  name: 'edge-gateway',
  enable_mdns: true,
  enable_bare: true,
  http_port: 8082,
  https_port: 443,
  interfaces: []
})

// LDAP Config
const ldapConfig = reactive({
  enabled: false,
  server: '',
  port: 389,
  base_dn: '',
  bind_dn: '',
  bind_password: '',
  user_filter: '(uid=%s)',
  attributes: '',
  use_ssl: false,
  skip_verify: false
})

const API_BASE = '/api/system'

const safeArray = (arr) => Array.isArray(arr) ? arr : []

const loadConfig = async () => {
  try {
    const [sysRes, netRes] = await Promise.all([
      request.get(API_BASE),
      request.get(API_BASE + '/network/interfaces')
    ])
    
    const configData = sysRes || {}
    if (configData.time) Object.assign(timeConfig, configData.time)
    if (Array.isArray(configData.routes)) staticRoutes.value = configData.routes
    if (configData.ha) Object.assign(haConfig, configData.ha)
    if (configData.hostname) Object.assign(hostnameConfig, configData.hostname)
    if (configData.ldap) Object.assign(ldapConfig, configData.ldap)

    if (Array.isArray(netRes)) {
      const liveInterfaces = netRes
      // Merge config state (e.g. enabled status) into live interfaces
      if (Array.isArray(configData.network)) {
        liveInterfaces.forEach(live => {
          const cfg = configData.network.find(c => c.name === live.name)
          if (cfg) {
            live.enabled = cfg.enabled
            // If live has no IPs (e.g. disconnected), but config has static, maybe we should show config's IPs?
            // For now, trust live for IPs, but trust config for administrative state.
          }
        })
      }
      // 清洗数据，确保 ip_configs 和 gateways 是干净的
      networkInterfaces.value = (liveInterfaces || []).map(iface => {
        const cleanedIface = {
          ...iface,
          ip_configs: [],
          gateways: []
        }
        // 处理 IP 配置
        cleanedIface.ip_configs = safeArray(iface.ip_configs)
          .filter(ip => ip && ip.address && typeof ip.address === 'string')
        // 处理网关配置
        cleanedIface.gateways = safeArray(iface.gateways)
          .filter(g => g && g.gateway)
        return cleanedIface
      })
    } else if (Array.isArray(configData.network)) {
      // 清洗数据，确保 ip_configs 和 gateways 是干净的
      networkInterfaces.value = (configData.network || []).map(iface => {
        const cleanedIface = {
          ...iface,
          ip_configs: [],
          gateways: []
        }
        // 处理 IP 配置
        cleanedIface.ip_configs = safeArray(iface.ip_configs)
          .filter(ip => ip && ip.address && typeof ip.address === 'string')
        // 处理网关配置
        cleanedIface.gateways = safeArray(iface.gateways)
          .filter(g => g && g.gateway)
        return cleanedIface
      })
    }
  } catch (e) {
    console.error('Failed to load system config', e)
  }
}

const saveConfig = async () => {
  const fullConfig = {
    time: timeConfig,
    network: networkInterfaces.value,
    routes: staticRoutes.value,
    ha: haConfig,
    hostname: hostnameConfig,
    ldap: ldapConfig
  }
  
  try {
    await request.put(API_BASE, fullConfig)
    // alert('配置保存成功') 
  } catch (e) {
    console.error('Failed to save config', e)
    // alert('配置保存失败')
  }
}

onMounted(loadConfig)
</script>

<style scoped>
.edge-compute-container {
  padding: 24px;
  min-height: calc(100vh - 56px);
  background: #f1f5f9;
}

.card-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-gray-900);
  letter-spacing: 0.5px;
}

/* --- 表单样式 --- */
.industrial-form :deep(.arco-form-item) {
  margin-bottom: 16px;
}

.industrial-form :deep(.arco-form-item-label) {
  font-size: 11px;
  color: var(--color-gray-50);
  font-weight: 500;
}

.rect-input {
  border-radius: 0 !important;
  font-family: 'JetBrains Mono', monospace;
}

/* --- 表格样式 --- */
.table-toolbar-industrial {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.toolbar-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--color-gray-900);
}

.industrial-table :deep(.arco-table-th) {
  background: var(--color-gray-50);
  border-bottom: 1px solid #e5e7eb;
  font-size: 11px;
  color: var(--color-gray-900);
  font-weight: 500;
}

.industrial-table :deep(.arco-table-td) {
  font-size: 12px;
  border-bottom: 1px solid #f1f3f5;
}

.industrial-table :deep(.arco-table-tr:hover .arco-table-td) {
  background: #f9fafb;
}

.mono-text { font-family: 'JetBrains Mono', monospace; font-size: 12px; }
.bold { font-weight: bold; }

.subscribers-line {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ip-group {
  margin-bottom: 8px;
}

.ip-group-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--color-gray-50);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.ip-group-items {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-left: 8px;
}

.sub-item {
  font-size: 12px;
  color: var(--color-gray-900);
  padding: 2px 4px;
  border-radius: 0;
  background: var(--color-gray-50);
  border-left: 2px solid #e5e7eb;
}

/* --- 指标卡片样式 --- */
.metrics-col {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.metrics-card {
  border: 1px solid #e5e7eb;
  border-radius: 0;
  padding: 16px;
  height: 100%;
  background: #ffffff;
  position: relative;
}

.metrics-card::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 1px;
  background: #0f172a;
  opacity: 0.05;
}

.metrics-label {
  font-size: 14px;
  color: var(--color-gray-50);
  margin-bottom: 8px;
}

.metrics-value {
  font-size: 26px;
  font-weight: 600;
  color: var(--color-gray-900);
  font-family: 'JetBrains Mono', monospace;
  letter-spacing: 0.5px;
}

/* --- 订阅者标签样式 --- */
.subscribers-line {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.sub-item {
  font-size: 11px;
  padding: 2px 6px;
  border: 1px solid #e5e7eb;
  border-radius: 0;
  color: var(--color-gray-900);
  background: var(--color-gray-50);
}

/* --- 访问状态区域 --- */
.access-list {
  margin-top: 16px;
}

.access-item {
  margin-bottom: 12px;
}

.access-title {
  font-size: 12px;
  font-weight: bold;
  margin-bottom: 4px;
}

.access-subtitle {
  margin-left: 16px;
}

/* --- 弹窗样式 --- */
:deep(.industrial-white-modal.arco-modal) {
  border-radius: 0 !important;
  border: 1px solid #e5e7eb !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1) !important;
}

:deep(.industrial-white-modal .arco-modal-header) {
  border-bottom: 1px solid #e5e7eb;
  padding: 16px 24px;
}

:deep(.industrial-white-modal .arco-modal-footer) {
  border-top: 1px solid #e5e7eb;
  padding: 16px 24px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .edge-compute-container {
    padding: 16px;
  }

  :deep(.arco-tabs-nav) {
    overflow-x: auto;
    white-space: nowrap;
  }

  :deep(.arco-tabs-tab) {
    padding: 12px 8px !important;
  }
}
</style>

