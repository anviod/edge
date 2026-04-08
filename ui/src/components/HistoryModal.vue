<template>
  <a-modal
    v-model:visible="historyDialog"
    title="历史数据"
    width="1440px"
    class="history-modal industrial-style"
    @cancel="onClose"
  >
    <template #footer>
      <a-space>
        <a-button @click="onClose">关闭</a-button>
        <a-button type="primary" :loading="historyLoading" @click="fetchHistory">查询</a-button>
        <a-button @click="downloadHistoryCSV" :disabled="historyData.length === 0">导出CSV</a-button>
      </a-space>
    </template>

    <div class="history-head">
      <span class="history-title">历史数据</span>
      <a-space>
        <span class="history-device">设备：{{ device?.name || '-' }}</span>
        <a-dropdown @select="handleColumnSelect">
          <a-button size="small">
            列筛选
            <icon-filter />
          </a-button>
          <template #dropdown>
            <a-menu>
              <a-menu-item key="ts" :disabled="true">
                <a-checkbox :checked="true" disabled>时间</a-checkbox>
              </a-menu-item>
              <a-menu-divider />
              <a-menu-item v-for="header in historyHeaders" :key="header.key">
                <a-checkbox 
                  :checked="selectedColumns.includes(header.key)" 
                  @change="checked => toggleColumn(header.key, checked)"
                >
                  {{ header.title }}
                </a-checkbox>
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </a-space>
    </div>

    <a-space direction="vertical" :size="16" fill>
      <a-row :gutter="16">
        <a-col :span="6">
          <a-select
            v-model:value="historyMode"
            :options="[
              { label: '最近记录', value: 'limit' },
              { label: '时间范围', value: 'range' }
            ]"
            placeholder="查询模式"
          />
        </a-col>
        <a-col :span="6" v-if="historyMode === 'limit'">
          <a-input-number v-model:value="historyLimit" :min="1" placeholder="记录数量" />
        </a-col>
        <a-col :span="12" v-if="historyMode === 'range'">
          <a-range-picker v-model:value="historyDateRange" show-time />
        </a-col>
      </a-row>

      <a-table
        :columns="tableColumns"
        :data="historyData"
        :loading="historyLoading"
        :pagination="{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          onChange: handlePaginationChange
        }"
        size="small"
        :bordered="{ cell: true }"
        class="history-table"
        :scroll="{ x: 'max-content' }"
      >
        <template #ts="{ record }">
          <span class="cell-content" :data-text="formatFriendlyTime(record.ts)" @click="copyToClipboard(formatFriendlyTime(record.ts))">
            {{ formatFriendlyTime(record.ts) }}
          </span>
        </template>
        <template #cell="{ text }">
          <span class="cell-content" :data-text="text" @click="copyToClipboard(text)">
            {{ text }}
          </span>
        </template>
      </a-table>
    </a-space>
  </a-modal>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconFilter } from '@arco-design/web-vue/es/icon'
import request from '@/utils/request'

const props = defineProps({
  visible: Boolean,
  device: {
    type: Object,
    default: () => null
  }
})

const emit = defineEmits(['update:visible'])

const historyDialog = ref(false)
const historyDevice = ref(null)
const historyLoading = ref(false)
const historyData = ref([])
const historyHeaders = ref([])
const historyDateRange = ref([])
const historyLimit = ref(100)
const historyMode = ref('limit')
const pagination = ref({
  current: 1,
  pageSize: 20,
  total: 0
})
const selectedColumns = ref([])

const tableColumns = computed(() => {
  const columns = [
    {
      title: '时间',
      dataIndex: 'ts',
      key: 'ts',
      width: 180,
      slotName: 'ts',
      ellipsis: true
    }
  ]
  
  historyHeaders.value.forEach(header => {
    if (selectedColumns.value.includes(header.key)) {
      columns.push({
        title: header.title,
        dataIndex: `data.${header.key.split('.')[1]}`,
        key: header.key,
        ellipsis: true,
        scopedSlots: {
          customRender: 'cell'
        }
      })
    }
  })
  
  return columns
})

const handleColumnSelect = (key) => {
  // 这个函数可以用来处理菜单项的额外逻辑，目前主要通过复选框控制
  console.log('Column selected:', key)
}

const toggleColumn = (key, checked) => {
  if (checked) {
    if (!selectedColumns.value.includes(key)) {
      selectedColumns.value.push(key)
    }
  } else {
    selectedColumns.value = selectedColumns.value.filter(col => col !== key)
  }
}

watch(
  () => props.visible,
  (val) => {
    historyDialog.value = val
    if (val) {
      resetHistoryQuery()
      fetchHistory()
    }
  }
)

watch(
  () => props.device,
  (val) => {
    historyDevice.value = val
    if (historyDialog.value) {
      resetHistoryQuery()
      fetchHistory()
    }
  }
)

const onClose = () => {
  historyDialog.value = false
}

watch(historyDialog, (val) => {
  if (!val) {
    emit('update:visible', false)
  }
})

const resetHistoryQuery = () => {
  historyData.value = []
  historyHeaders.value = []
  historyMode.value = 'limit'
  historyLimit.value = 100
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  const toLocalISO = (d) => {
    const offset = d.getTimezoneOffset() * 60000
    return new Date(d.getTime() - offset).toISOString().slice(0, 16)
  }
  historyDateRange.value = [toLocalISO(start), toLocalISO(end)]
}

const formatFriendlyTime = (ts) => {
  if (!ts && ts !== 0) return '-'
  const date = new Date(Number(ts) * 1000)
  if (Number.isNaN(date.getTime())) return '-'

  const diff = Date.now() - date.getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (seconds < 30) return '刚刚'
  if (seconds < 60) return `${seconds}秒前`
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`

  const fmt = (n) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${fmt(date.getMonth() + 1)}-${fmt(date.getDate())} ${fmt(date.getHours())}:${fmt(date.getMinutes())}:${fmt(date.getSeconds())}`
}

const fetchHistory = async () => {
  if (!historyDevice.value || !historyDevice.value.id) {
    return
  }

  historyLoading.value = true
  historyData.value = []
  historyHeaders.value = []

  try {
    let url = `/api/devices/${historyDevice.value.id}/history`
    if (historyMode.value === 'range') {
      const start = historyDateRange.value[0] + ':00'
      const end = historyDateRange.value[1] + ':00'
      url += `?start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}`
    } else {
      url += `?limit=${historyLimit.value}`
    }
    
    // 添加分页参数
    url += `&page=${pagination.value.current}&pageSize=${pagination.value.pageSize}`

    const res = await request.get(url, { timeout: 60000 })
    // 处理 API 响应，支持两种格式：对象（带 data 和 total）或直接数组
    if (Array.isArray(res)) {
      // 直接返回数组的情况
      historyData.value = res
      pagination.value.total = res.length
    } else {
      // 返回对象的情况（支持分页）
      historyData.value = res?.data || []
      pagination.value.total = res?.total || 0
    }

    if (historyData.value.length > 0) {
      const keys = new Set()
      historyData.value.forEach(row => {
        if (row.data) {
          Object.keys(row.data).forEach(k => keys.add(k))
        }
      })
      historyHeaders.value = Array.from(keys).sort().map(k => ({ title: k, key: `data.${k}` }))
      // 默认选择所有列
      selectedColumns.value = historyHeaders.value.map(header => header.key)
    }
  } catch (e) {
    Message.error('获取历史数据失败: ' + e.message)
  } finally {
    historyLoading.value = false
  }
}

const handlePaginationChange = (page, pageSize) => {
  pagination.value.current = page
  pagination.value.pageSize = pageSize
  fetchHistory()
}

const copyToClipboard = (text) => {
  if (text) {
    navigator.clipboard.writeText(text).then(() => {
      Message.success('已复制到剪贴板')
    }).catch(() => {
      Message.error('复制失败')
    })
  }
}

const downloadHistoryCSV = () => {
  if (historyData.value.length === 0) {
    Message.warning('无数据可导出')
    return
  }

  const headers = ['时间', ...historyHeaders.value.map(h => h.title)]
  const keys = ['ts', ...historyHeaders.value.map(h => h.key)]

  const rows = historyData.value.map(row => {
    return keys.map(key => {
      if (key === 'ts') return formatFriendlyTime(row.ts)
      const prop = key.split('.')[1]
      return row.data ? (row.data[prop] ?? '') : ''
    })
  })

  const csvContent = [headers.join(','), ...rows.map(r => r.join(','))].join('\n')
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `${historyDevice.value?.name || 'device'}_history_${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
}
</script>

<style scoped>
.history-modal.industrial-style .arco-modal-content {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border-radius: 0;
}

.history-modal .history-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: #333;
  border-bottom: 1px solid #e5e7eb;
  margin-bottom: 12px;
  padding-bottom: 8px;
}

.history-modal .history-title {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  white-space: nowrap;
}

.history-modal .history-device {
  color: #666;
  font-size: 12px;
  white-space: nowrap;
}

.history-modal .history-table .arco-table {
  background: #ffffff;
  color: #333;
  border: 1px solid #e5e7eb;
  border-radius: 0;
  font-size: 12px;
}

.history-modal .history-table .arco-table thead th {
  background: #f8fafc;
  color: #333;
  border-bottom: 1px solid #e5e7eb;
  font-weight: 600;
  padding: 8px 12px;
  height: 32px;
  white-space: nowrap;
  font-size: 12px;
}

.history-modal .history-table .arco-table tbody tr:hover {
  background: #f9fafb;
}

.history-modal .history-table .arco-table tbody tr.arco-table-row-selected {
  background: #eff6ff;
  border: 1px solid #dbeafe;
}

.history-modal .history-table .arco-table tbody td {
  border-bottom: 1px solid #e5e7eb;
  padding: 8px 12px;
  height: 32px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 11px;
}

.history-modal .cell-content {
  position: relative;
  cursor: pointer;
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-modal .cell-content:hover {
  color: #1890ff;
}

.history-modal .cell-content:hover::after {
  content: attr(data-text);
  position: absolute;
  bottom: 100%;
  left: 0;
  background: rgba(0, 0, 0, 0.8);
  color: white;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  white-space: normal;
  max-width: 300px;
  z-index: 1000;
  margin-bottom: 8px;
  word-break: break-all;
}

.history-modal .history-table .arco-table-container {
  border-radius: 0;
}
</style>
