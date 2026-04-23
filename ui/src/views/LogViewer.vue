<template>
    <div class="log-viewer-container">
        <!-- Toolbar -->
        <div class="log-toolbar">
            <div class="toolbar-left">
                <icon-cloud class="toolbar-icon" />
                <span class="toolbar-title">实时日志</span>
            </div>
            <div class="toolbar-right">
                <a-select v-model="selectedLevel" :options="logLevelOptions" placeholder="日志级别" size="small" class="mr-4">
                    <template #option="option">
                        <span>{{ option.label }}</span>
                    </template>
                </a-select>
                <div class="switch-container">
                    <span class="switch-label">实时打印</span>
                    <a-switch v-model="isStreaming" size="small" />
                </div>
                <a-button type="outline" size="small" @click="clearLogs" class="mr-2">
                    <template #icon><icon-delete /></template>
                    清空屏幕
                </a-button>
                <a-button type="primary" size="small" @click="downloadLogs">
                    <template #icon><icon-download /></template>
                    导出 CSV
                </a-button>
            </div>
        </div>

        <!-- Log Terminal -->
        <div class="log-terminal" ref="terminalRef">
            <div v-if="displayLogs.length === 0" class="no-logs">
                暂无日志...
            </div>
            <div v-for="(log, index) in displayLogs" :key="index" class="log-line">
                <span class="log-time">{{ formatTime(log.ts) }}</span>
                <span :class="['log-level', getLevelClass(log.level)]">{{ (log.level || 'INFO').toUpperCase() }}</span>
                <span class="log-message">{{ log.msg }}</span>
                <!-- Render extra fields -->
                <span v-for="(val, key) in getExtraFields(log)" :key="key" class="log-extra">
                    {{ key }}={{ val }}
                </span>
            </div>
        </div>

        <!-- Pagination -->
        <div v-if="logs.length > 0" class="log-pagination">
            <a-pagination v-model:current="page" :page-size="perPage" :total="filteredLogs.length" size="small" />
        </div>
    </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { IconCloud, IconDelete, IconDownload } from '@arco-design/web-vue/es/icon'

const logs = ref([])
const isStreaming = ref(true)
const terminalRef = ref(null)
let ws = null
const maxLogs = 1000
const perPage = 30
const page = ref(1)

const selectedLevel = ref('ALL')
const logLevelOptions = [
  { label: 'ALL', value: 'ALL' },
  { label: 'INFO', value: 'INFO' },
  { label: 'WARN', value: 'WARN' },
  { label: 'ERROR', value: 'ERROR' },
  { label: 'DEBUG', value: 'DEBUG' }
]

const filteredLogs = computed(() => {
    if (selectedLevel.value === 'ALL') return logs.value
    return logs.value.filter(log => {
        const lvl = (log.level || 'INFO').toUpperCase()
        return lvl === selectedLevel.value
    })
})

const pageCount = computed(() => {
    return Math.ceil(filteredLogs.value.length / perPage) || 1
})

const displayLogs = computed(() => {
    const start = (page.value - 1) * perPage
    const end = start + perPage
    return filteredLogs.value.slice(start, end)
})

// Auto-switch to first page when streaming
watch(() => logs.value.length, () => {
    if (isStreaming.value && page.value !== 1) {
        page.value = 1
    }
})

// Reset page when filter changes
watch(selectedLevel, () => {
    page.value = 1
})

// Pause streaming when user changes page manually (unless it's the first page)
watch(page, (newVal) => {
    if (newVal !== 1) {
        isStreaming.value = false
    }
})

const connectWs = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    
    // Get token from localStorage
    let token = ''
    try {
        const raw = localStorage.getItem('loginInfo')
        if (raw) {
            const parsed = JSON.parse(raw)
            token = parsed.token || (parsed.data && parsed.data.token) || ''
        }
    } catch (e) {
        console.error('Failed to parse loginInfo', e)
    }
    
    ws = new WebSocket(`${protocol}//${host}/api/ws/logs${token ? `?token=${token}` : ''}`)

    ws.onopen = () => {
        console.log('Log WS connected')
    }

    ws.onmessage = (event) => {
        if (!isStreaming.value) return

        try {
            const log = JSON.parse(event.data)
            logs.value.unshift(log)
            if (logs.value.length > maxLogs) {
                logs.value.pop()
            }
            // Stay on first page
            if (page.value !== 1) {
                page.value = 1
            }
        } catch (e) {
            if (!isStreaming.value) return
            
            logs.value.unshift({ ts: new Date().toISOString(), level: 'INFO', msg: event.data })
            if (page.value !== 1) {
                page.value = 1
            }
        }
    }

    ws.onclose = () => {
        console.log('Log WS closed')
    }
}

const scrollToBottom = () => {
    // No longer needed for reverse order
}


const clearLogs = () => {
    logs.value = []
}

const downloadLogs = () => {
    // Export filteredLogs as CSV
    const headers = ['Timestamp', 'Level', 'Message', 'Details']
    const rows = filteredLogs.value.map(log => {
        const ts = log.ts ? new Date(log.ts).toLocaleString() : ''
        const level = (log.level || 'INFO').toUpperCase()
        const msg = (log.msg || '').replace(/"/g, '""') // Escape quotes
        const details = JSON.stringify(getExtraFields(log)).replace(/"/g, '""')
        return `"${ts}","${level}","${msg}","${details}"`
    })
    
    // Add BOM for Excel utf-8 compatibility
    const bom = '\uFEFF'
    const csvContent = bom + [headers.join(','), ...rows].join('\n')
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `edge_logs_${new Date().toISOString().slice(0,19).replace(/[:T]/g, '-')}.csv`
    link.click()
    URL.revokeObjectURL(link.href)
}

const formatTime = (ts) => {
    if (!ts) return ''
    return new Date(ts).toLocaleTimeString() + '.' + new Date(ts).getMilliseconds().toString().padStart(3, '0')
}

const getLevelClass = (level) => {
    const l = (level || '').toUpperCase()
    if (l === 'ERROR' || l === 'FATAL') return 'error'
    if (l === 'WARN') return 'warn'
    if (l === 'DEBUG') return 'debug'
    return 'info'
}

const getExtraFields = (log) => {
    const { ts, level, msg, caller, ...rest } = log
    return rest
}

onMounted(() => {
    connectWs()
})

onUnmounted(() => {
    if (ws) ws.close()
})
</script>

<style scoped>
.log-viewer-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
  background-color: #ffffff;
}

.log-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px 16px;
  background-color: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 0;
}

.toolbar-left {
  display: flex;
  align-items: center;
}

.toolbar-icon {
  margin-right: 8px;
  color: #0ea5e9;
  font-size: 18px;
}

.toolbar-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}

.toolbar-right {
  display: flex;
  align-items: center;
}

.toolbar-right .switch-container {
  display: flex;
  align-items: center;
  min-width: 100px;
  margin-right: 16px;
}

.toolbar-right .switch-label {
  margin-right: 8px;
  font-size: 13px;
  color: #64748b;
  white-space: nowrap;
}

.log-terminal {
  flex: 1;
  overflow-y: auto;
  font-family: 'JetBrains Mono', 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.5;
  padding: 16px;
  background-color: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 0;
  margin-bottom: 16px;
  color: #1e293b;
}

.no-logs {
  text-align: center;
  color: #94a3b8;
  padding: 48px 0;
}

.log-line {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  padding: 6px 0;
  border-bottom: 1px solid #f1f5f9;
}

.log-time {
  margin-right: 12px;
  color: #64748b;
  font-size: 12px;
  min-width: 100px;
}

.log-level {
  margin-right: 12px;
  font-weight: 600;
  font-size: 12px;
  min-width: 60px;
}

.log-level.error {
  color: #ef4444;
}

.log-level.warn {
  color: #f59e0b;
}

.log-level.info {
  color: #10b981;
}

.log-level.debug {
  color: #6366f1;
}

.log-message {
  flex: 1;
  word-break: break-all;
  white-space: pre-wrap;
  color: #1e293b;
}

.log-extra {
  margin-left: 8px;
  color: #94a3b8;
  font-size: 11px;
  white-space: nowrap;
}

.log-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

/* 滚动条样式 */
.log-terminal::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.log-terminal::-webkit-scrollbar-track {
  background: #f1f5f9;
  border-radius: 0;
}

.log-terminal::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 0;
}

.log-terminal::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}
</style>

