<template>
  <a-modal v-model:visible="visible" :title="title" :width="900" :footer="false" unmount-on-close>
    <template v-if="isClientPushMode">
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="6">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">发送成功</div>
            <div style="font-size: 24px; font-weight: 600; color: #00b42a; margin-top: 4px">{{ stats.success_count || 0 }}</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">发送失败</div>
            <div style="font-size: 24px; font-weight: 600; color: #f53f3f; margin-top: 4px">{{ stats.fail_count || 0 }}</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">重连次数</div>
            <div style="font-size: 24px; font-weight: 600; color: #ff7d00; margin-top: 4px">{{ stats.reconnect_count || 0 }}</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">断线时长</div>
            <div style="font-size: 24px; font-weight: 600; color: #4e5969; margin-top: 4px">{{ disconnectDuration }}</div>
          </a-card>
        </a-col>
      </a-row>
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="12">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">运行时长</div>
            <div style="font-size: 24px; font-weight: 600; color: #4e5969; margin-top: 4px">{{ formatUptime(stats.uptime || 0) }}</div>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">推送成功率</div>
            <div style="font-size: 24px; font-weight: 600; color: #00b42a; margin-top: 4px">{{ successRate }}%</div>
          </a-card>
        </a-col>
      </a-row>
    </template>

    <template v-else-if="isOpcuaServerMode">
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="6">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">当前连接客户端</div>
            <div style="font-size: 24px; font-weight: 600; color: #165dff; margin-top: 4px">{{ stats.client_count || 0 }}</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">当前订阅数量</div>
            <div style="font-size: 24px; font-weight: 600; color: #0ea5e9; margin-top: 4px">{{ stats.subscription_count || 0 }}</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">最近读操作</div>
            <div style="font-size: 24px; font-weight: 600; color: #00b42a; margin-top: 4px">{{ stats.read_count || 0 }}</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">最近写操作</div>
            <div style="font-size: 24px; font-weight: 600; color: #165dff; margin-top: 4px">{{ stats.write_count || 0 }}</div>
          </a-card>
        </a-col>
      </a-row>
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="12">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">运行时长</div>
            <div style="font-size: 24px; font-weight: 600; color: #4e5969; margin-top: 4px">{{ formatUptime(stats.uptime || 0) }}</div>
          </a-card>
        </a-col>
        <a-col :span="12">
          <a-card :bordered="true" style="text-align: center">
            <div style="color: #6b7280; font-size: 12px">数据吞吐量</div>
            <div style="font-size: 24px; font-weight: 600; color: #0ea5e9; margin-top: 4px">{{ formatThroughput(stats.throughput || 0) }}</div>
          </a-card>
        </a-col>
      </a-row>
    </template>

    <a-divider style="margin: 12px 0" />

    <div style="display: flex; align-items: center; margin-bottom: 8px">
      <span style="font-size: 13px; font-weight: 600">实时日志 ({{ logTitle }})</span>
      <div style="flex: 1" />
      <a-switch v-model="isStreaming" size="small" style="margin-right: 8px" />
      <span style="font-size: 12px; color: #6b7280; margin-right: 16px">实时滚动</span>
      <a-button type="outline" size="small" @click="downloadLogs">
        <template #icon><icon-download :size="12" /></template>
        下载日志
      </a-button>
    </div>

    <div class="log-viewer">
      <div v-if="paginatedLogs.length === 0" style="text-align: center; color: #6b7280; padding: 48px 0">暂无日志...</div>
      <div v-for="(log, idx) in paginatedLogs" :key="idx" class="log-line">
        <span style="color: #6b7280; margin-right: 8px">[{{ formatTime(log.ts) }}]</span>
        <span :style="{ color: getLevelColor(log.level), fontWeight: 'bold', marginRight: '8px' }">{{ (log.level || 'INFO').toUpperCase() }}</span>
        <span style="color: #1e293b">{{ log.msg }}</span>
        <span v-for="(val, key) in getExtraFields(log)" :key="key" style="color: #6b7280; margin-left: 8px; font-size: 12px">
          {{ key }}={{ val }}
        </span>
      </div>
    </div>

    <div style="display: flex; justify-content: center; padding: 8px 0">
      <a-pagination v-model:current="page" :page-size="20" :total="logs.length" size="small" show-page-size />
    </div>
  </a-modal>
</template>

<script setup>
import { ref, watch, computed, onUnmounted } from 'vue'
import { IconDownload } from '@arco-design/web-vue/es/icon'
import { showMessage } from '@/composables/useGlobalState'
import request from '@/utils/request'

const props = defineProps({
  visible: { type: Boolean, default: false },
  type: { type: String, default: 'mqtt' },
  itemId: { type: String, default: '' }
})

const emit = defineEmits(['update:visible'])

const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})
const isStreaming = ref(true)
const stats = ref({})
const logs = ref([])
const page = ref(1)
let timer = null
let ws = null

const title = computed(() => {
  if (props.type === 'mqtt') return 'MQTT 运行监控'
  if (props.type === 'http') return 'HTTP 运行监控'
  if (props.type === 'sparkplug_b') return 'Sparkplug B 运行监控'
  if (props.type === 'opcua') return 'OPC UA 运行监控'
  if (props.type === 'edgeos-mqtt') return 'edgeOS(MQTT) 运行监控'
  if (props.type === 'edgeos-nats') return 'edgeOS(NATS) 运行监控'
  return '运行监控'
})

const isClientPushMode = computed(() => {
  return props.type === 'mqtt' || props.type === 'http' || props.type === 'sparkplug_b' || props.type === 'edgeos-mqtt' || props.type === 'edgeos-nats'
})

const isOpcuaServerMode = computed(() => {
  return props.type === 'opcua'
})

const paginatedLogs = computed(() => {
  const start = (page.value - 1) * 20
  return logs.value.slice(start, start + 20)
})

const logTitle = computed(() => {
  if (props.type === 'mqtt') return 'MQTT'
  if (props.type === 'http') return 'HTTP'
  if (props.type === 'sparkplug_b') return 'Sparkplug B'
  if (props.type === 'opcua') return 'OPC UA'
  if (props.type === 'edgeos-mqtt') return 'edgeOS(MQTT)'
  if (props.type === 'edgeos-nats') return 'edgeOS(NATS)'
  return '日志'
})

const disconnectDuration = computed(() => {
  const offlineTime = stats.value.last_offline_time
  const onlineTime = stats.value.last_online_time
  if (!offlineTime) return '0s'
  const now = Date.now()
  if (offlineTime > onlineTime) {
    return formatUptime(Math.floor((now - offlineTime) / 1000))
  }
  return '0s'
})

const successRate = computed(() => {
  const success = stats.value.success_count || 0
  const fail = stats.value.fail_count || 0
  const total = success + fail
  if (total === 0) return 0
  return Math.round((success / total) * 100)
})

const formatThroughput = (bytes) => {
  if (!bytes || bytes === 0) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

const cleanup = () => {
  if (timer) { clearInterval(timer); timer = null }
  if (ws) { ws.close(); ws = null }
}

onUnmounted(cleanup)

watch(() => props.visible, (val) => {
  if (val) {
    logs.value = []
    page.value = 1
    isStreaming.value = true
    refreshStats()
    timer = setInterval(refreshStats, props.type === 'mqtt' ? 1000 : 3000)
    connectWs()
  } else {
    cleanup()
  }
})

const refreshStats = async () => {
  if (!props.itemId) return
  try {
    let apiType = props.type
    // 转换类型名称以匹配后端 API 路径
    if (apiType === 'edgeos-mqtt') apiType = 'edgeos-mqtt'
    if (apiType === 'edgeos-nats') apiType = 'edgeos-nats'
    if (apiType === 'sparkplug_b') apiType = 'sparkplugb'
    
    const data = await request.get(`/api/northbound/${apiType}/${props.itemId}/stats`, { silent: true })
    stats.value = data
  } catch (e) {}
}

const connectWs = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  let token = ''
  try {
    const raw = localStorage.getItem('loginInfo')
    if (raw) {
      const parsed = JSON.parse(raw)
      token = parsed.token || (parsed.data && parsed.data.token) || ''
    }
  } catch (e) {}

  ws = new WebSocket(`${protocol}//${host}/api/ws/logs?token=${token}`)
  ws.onmessage = (event) => {
    if (!isStreaming.value) return
    try {
      const log = JSON.parse(event.data)
      let targetComponents = []
      
      if (isClientPushMode.value) {
        targetComponents = ['mqtt-client', 'http-client', 'sparkplugb-client', 'edgos-mqtt-client', 'edgos-nats-client']
      } else if (isOpcuaServerMode.value) {
        targetComponents = ['opcua-server']
      }

      if (targetComponents.includes(log.component)) {
        logs.value.unshift(log)
        if (logs.value.length > 500) logs.value.pop()
        if (page.value !== 1) page.value = 1
      }
    } catch (e) {}
  }
}

const formatTime = (ts) => {
  if (!ts) return ''
  return new Date(ts).toLocaleTimeString() + '.' + new Date(ts).getMilliseconds().toString().padStart(3, '0')
}

const formatUptime = (seconds) => {
  if (seconds < 60) return seconds + '秒'
  if (seconds < 3600) return Math.floor(seconds / 60) + '分' + (seconds % 60) + '秒'
  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return hours + '小时' + mins + '分'
}

const getLevelColor = (level) => {
  const l = (level || '').toUpperCase()
  if (l === 'ERROR' || l === 'FATAL') return '#f53f3f'
  if (l === 'WARN') return '#ff7d00'
  return '#00b42a'
}

const getExtraFields = (log) => {
  const { ts, level, msg, caller, component, ...rest } = log
  return rest
}

const downloadLogs = () => {
  const rows = logs.value.map(log => {
    const ts = log.ts ? new Date(log.ts).toLocaleString() : ''
    const level = (log.level || 'INFO').toUpperCase()
    const msg = log.msg || ''
    return `[${ts}] [${level}] ${msg}`
  })
  const content = rows.join('\n')
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `${props.type}_logs_${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.log`
  link.click()
  URL.revokeObjectURL(link.href)
}
</script>

<style scoped>
.log-viewer {
  height: 300px;
  overflow-y: auto;
  font-family: 'JetBrains Mono', 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  line-height: 1.4;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 0;
  padding: 8px;
}

.log-line {
  white-space: pre-wrap;
  word-break: break-all;
  padding: 2px 0;
  border-bottom: 1px solid #f5f5f5;
}
</style>

