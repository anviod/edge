<template>
  <a-modal
    v-model:visible="localVisible"
    :title="'OPC-UA 点位扫描'"
    :width="1200"
    :mask-closable="false"
    unmount-on-close
    modal-class="industrial-white-modal"
    :align-center="true"
    @cancel="onCancel"
  >
    <div class="scanner-content">
      <!-- 头部 Banner -->
      <div class="scanner-header-banner">
        <div class="endpoint-info">
          <span class="label">ENDPOINT</span>
          <span class="value font-mono">{{ deviceConfig?.endpoint || '-' }}</span>
        </div>
        <div class="protocol-badge">
          <span class="protocol-tag-simple">OPC-UA</span>
        </div>
      </div>

      <!-- 工具栏 - 水平对齐，扫描按钮靠右 -->
      <div class="scanner-toolbar">
        <div class="toolbar-left">
          <a-input
            v-model="filters.keyword"
            placeholder="搜索 NodeID / 名称"
            size="small"
            class="industrial-input"
            allow-clear
          >
            <template #prefix><IconSearch /></template>
          </a-input>

          <div class="toolbar-divider"></div>

          <!-- 状态筛选 - 极简符号点+文字 -->
          <div class="status-filters">
            <div 
              v-for="opt in diffStatusOptions" 
              :key="opt.value"
              class="status-filter-item"
              :class="{ active: filters.diffStatus === opt.value }"
              @click="filters.diffStatus = opt.value"
            >
              <span :class="['status-dot', opt.value === 'new' ? 'dot-new' : opt.value === 'existing' ? 'dot-existing' : 'dot-all']"></span>
              <span class="status-label">{{ opt.label }}</span>
            </div>
          </div>

          <div class="toolbar-divider"></div>

          <a-checkbox v-model="filters.varsOnly" class="industrial-checkbox">
            仅显示变量
          </a-checkbox>
        </div>

        <div class="toolbar-right">
          <a-button
            type="primary"
            size="small"
            :loading="loading"
            @click="startScan"
            :disabled="!deviceConfig?.endpoint"
            class="scan-btn"
          >
            <template #icon><IconScan /></template>
            重新扫描
          </a-button>
        </div>
      </div>

      <!-- 统计信息栏 -->
      <div v-if="filteredResults.length > 0" class="stats-bar">
        <div class="stats-item">
          <span class="stats-number">{{ filteredResults.length }}</span>
          <span class="stats-unit">个节点</span>
        </div>
        <div class="stats-divider"></div>
        <div class="stats-item">
          <span class="status-dot dot-new"></span>
          <span class="stats-number">{{ filteredResults.filter(r => r.diff_status === 'new').length }}</span>
          <span class="stats-unit">新增</span>
        </div>
        <div class="stats-item">
          <span class="status-dot dot-existing"></span>
          <span class="stats-number">{{ filteredResults.filter(r => r.diff_status === 'existing').length }}</span>
          <span class="stats-unit">存量</span>
        </div>
      </div>

      <!-- 表格区域 -->
      <div class="table-wrapper">
        <a-spin :loading="loading" class="w-full">
          <a-table
            v-if="results.length > 0"
            :columns="columns"
            :data="filteredResults"
            size="small"
            :bordered="{ cell: true, wrapper: true }"
            :pagination="false"
            :scroll="{ y: 500 }"
            row-key="node_id"
            class="industrial-table"
          >

            
            <template #selection="{ record }">
              <div class="row-checkbox-wrapper">
                <a-checkbox 
                  :model-value="selected.includes(record.node_id)"
                  @change="(checked) => onRowCheck(checked, record)"
                  :disabled="record.diff_status === 'existing'"
                  class="industrial-checkbox"
                />
              </div>
            </template>
            
            <template #status="{ record }">
              <div class="status-cell">
                <span :class="['status-dot', record.diff_status === 'new' ? 'dot-new' : 'dot-existing']" :title="record.diff_status === 'new' ? '新增' : '存量'"></span>
              </div>
            </template>

            <template #name="{ record }">
              <div class="node-name-cell" :style="{ paddingLeft: `${record.level * 20}px` }">
                <span class="node-icon">{{ record.type === 'Folder' ? '📁' : '◇' }}</span>
                <span class="node-name font-mono">{{ record.display_name }}</span>
              </div>
            </template>
          </a-table>
          
          <div v-else-if="!loading" class="empty-placeholder">
            <IconSearch size="28" class="empty-icon" />
            <div class="empty-text">点击"重新扫描"获取点位</div>
          </div>
        </a-spin>
      </div>
    </div>

    <!-- Footer 区域 -->
    <template #footer>
      <div class="modal-footer">
        <div class="footer-info">
          <span class="footer-label">已选择</span>
          <span class="footer-number">{{ selected.length }}</span>
          <span class="footer-unit">个点位</span>
        </div>
        <div class="footer-actions">
          <a-button @click="onCancel" size="small" class="cancel-btn">取消</a-button>
          <a-button 
            type="primary" 
            @click="addSelected" 
            :disabled="selected.length === 0" 
            :loading="adding" 
            size="small" 
            class="confirm-btn"
          >
            确认添加
          </a-button>
        </div>
      </div>
    </template>
  </a-modal>
</template>

<script setup>
import { ref, reactive, computed, watch, h, resolveComponent } from 'vue'
import { IconScan, IconSearch } from '@arco-design/web-vue/es/icon'
import request from '@/utils/request'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  channelId: {
    type: String,
    required: true
  },
  deviceId: {
    type: String,
    required: true
  },
  deviceConfig: {
    type: Object,
    default: () => ({})
  },
  existingPoints: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:visible', 'cancel', 'points-added'])

const loading = ref(false)
const adding = ref(false)
const results = ref([])
// 选中的节点（存储 node_id 数组）
const selected = ref([])
const localVisible = ref(props.visible)

const filters = reactive({
  keyword: '',
  varsOnly: true,
  diffStatus: 'all'
})

const diffStatusOptions = [
  { label: '全部', value: 'all' },
  { label: '新增', value: 'new' },
  { label: '存量', value: 'existing' }
]

watch(() => props.visible, (newValue) => {
  localVisible.value = newValue
})

watch(localVisible, (newValue) => {
  emit('update:visible', newValue)
})

const columns = [
  {
    title: () => {
      const Checkbox = resolveComponent('a-checkbox')
      return h('div', { class: 'selection-header' }, [
        h(Checkbox, {
          modelValue: isAllSelected.value,
          indeterminate: isIndeterminate.value,
          onChange: onToggleAll,
          disabled: selectableNodes.value.length === 0,
          class: 'industrial-checkbox'
        })
      ])
    },
    key: 'selection',
    width: 60,
    align: 'center',
    slotName: 'selection'
  },
  {
    title: '状态',
    slotName: 'status',
    width: 80,
    align: 'center'
  },
  {
    title: '节点名称',
    slotName: 'name',
    ellipsis: true,
    tooltip: true
  },
  {
    title: '节点类型',
    dataIndex: 'type',
    width: 120
  },
  {
    title: '数据类型',
    dataIndex: 'data_type',
    width: 120,
    ellipsis: true
  },
  {
    title: '访问权限',
    dataIndex: 'access_level',
    width: 100,
    ellipsis: true
  },
  {
    title: 'NodeID',
    dataIndex: 'node_id',
    ellipsis: true
  }
]

const existingAddresses = computed(() => {
  const set = new Set()
  for (const p of props.existingPoints) {
    if (p && p.address) set.add(p.address)
  }
  return set
})

const filteredResults = computed(() => {
  let list = results.value || []
  
  if (filters.varsOnly) {
    list = list.filter(r => r.type === 'Variable')
  }
  
  if (filters.diffStatus !== 'all') {
    list = list.filter(r => r.diff_status === filters.diffStatus)
  }
  
  if (filters.keyword) {
    const s = filters.keyword.trim().toLowerCase()
    list = list.filter(r =>
      (r.node_id && r.node_id.toLowerCase().includes(s)) ||
      (r.display_name && r.display_name.toLowerCase().includes(s))
    )
  }
  
  return list
})

// 获取可选择的节点（仅新增节点）
const selectableNodes = computed(() => {
  return filteredResults.value.filter(r => r.diff_status !== 'existing')
})

// 是否全选
const isAllSelected = computed(() => {
  const selectable = selectableNodes.value.map(n => n.node_id)
  return (
    selectable.length > 0 &&
    selectable.every(id => selected.value.includes(id))
  )
})

// 半选状态（关键）
const isIndeterminate = computed(() => {
  const selectable = selectableNodes.value.map(n => n.node_id)
  const selectedCount = selected.value.filter(id =>
    selectable.includes(id)
  ).length

  return selectedCount > 0 && selectedCount < selectable.length
})

// 点击表头切换逻辑（核心）
const onToggleAll = () => {
  const selectable = selectableNodes.value.map(n => n.node_id)

  if (isAllSelected.value) {
    // 👉 当前是全选 → 取消
    selected.value = selected.value.filter(
      id => !selectable.includes(id)
    )
  } else {
    // 👉 当前不是全选 → 全选
    const set = new Set(selected.value)
    selectable.forEach(id => set.add(id))
    selected.value = Array.from(set)
  }
}

const onCancel = () => {
  emit('update:visible', false)
  emit('cancel')
  resetState()
}

const resetState = () => {
  loading.value = false
  adding.value = false
  results.value = []
  selected.value = []
}

const startScan = async () => {
  if (!props.deviceConfig?.endpoint) {
    emit('error', 'OPC UA 设备未配置 endpoint，无法扫描')
    return
  }

  loading.value = true
  results.value = []
  selected.value = []

  try {
    const payload = { mode: 'fast' }
    const res = await request.post(
      `/api/channels/${props.channelId}/devices/${props.deviceId}/scan`,
      payload,
      { timeout: 180000 } // 增加到3分钟
    )

    if (Array.isArray(res)) {
      if (res.length === 0) {
        emit('info', '扫描结果为空')
      } else {
        results.value = flattenOpcNodes(res)
      }
    } else {
      emit('error', '扫描结果格式错误')
    }
  } catch (e) {
    emit('error', '扫描失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

const flattenOpcNodes = (nodes, level = 0) => {
  let result = []
  for (const node of nodes) {
    const item = {
      ...node,
      level: level,
      display_name: node.name || node.node_id,
      type: node.type || 'Unknown',
      data_type: node.data_type || '',
      access_level: node.access_level || ''
    }

    if (node.node_id) {
      item.diff_status = existingAddresses.value.has(node.node_id) ? 'existing' : 'new'
    }

    result.push(item)

    if (node.children && node.children.length > 0) {
      result = result.concat(flattenOpcNodes(node.children, level + 1))
    }
  }
  return result
}

// 行 checkbox 点击事件
const onRowCheck = (checked, record) => {
  const id = record.node_id
  if (!id) return

  if (checked) {
    if (!selected.value.includes(id)) {
      selected.value.push(id)
    }
  } else {
    selected.value = selected.value.filter(i => i !== id)
  }
}

const addSelected = async () => {
  if (selected.value.length === 0) return

  const selectedObjs = results.value.filter(r =>
    selected.value.includes(r.node_id)
  )

  adding.value = true
  let successCount = 0
  let failCount = 0

  for (const obj of selectedObjs) {
    if (obj.type !== 'Variable') continue

    try {
      let rw = obj.access_level && obj.access_level.includes('CurrentWrite') ? 'RW' : 'R'
      let dt = (obj.data_type || 'Float').toLowerCase()
      
      const dataTypeMap = {
        'bool': 'bool', 'boolean': 'bool',
        'int16': 'int16', 'short': 'int16',
        'uint16': 'uint16', 'unsignedshort': 'uint16',
        'int32': 'int32', 'int': 'int32',
        'uint32': 'uint32', 'unsignedint': 'uint32',
        'float': 'float32', 'double': 'float64',
        'string': 'string'
      }
      
      dt = dataTypeMap[dt] || 'float32'

      const pointPayload = {
        id: obj.node_id,
        name: obj.display_name || obj.node_id,
        address: obj.node_id,
        datatype: dt,
        readwrite: rw,
        unit: '',
        scale: 1.0,
        offset: 0.0
      }

      await request.post(
        `/api/channels/${props.channelId}/devices/${props.deviceId}/points`,
        pointPayload
      )
      successCount++
    } catch (e) {
      console.error(e)
      failCount++
    }
  }

  adding.value = false
  emit('points-added', { success: successCount, fail: failCount })
  onCancel()
}
</script>

<style scoped>
/* 白色工业风基础样式 */
.scanner-content {
  padding: 0;
}

/* 头部 Banner */
.scanner-header-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #fafbfc;
  border-bottom: 1px solid #e9ecef;
  margin-bottom: 16px;
}

.endpoint-info {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.scanner-header-banner .label {
  font-size: 11px;
  color: #6c757d;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.scanner-header-banner .value {
  font-size: 12px;
  color: #495057;
  font-family: 'JetBrains Mono', monospace;
}

.protocol-tag-simple {
  background: #e9ecef;
  color: #495057;
  padding: 4px 12px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

/* 工具栏 - 水平对齐，按钮靠右 */
.scanner-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #ffffff;
  border: 1px solid #e9ecef;
  margin-bottom: 12px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
}

.toolbar-right {
  display: flex;
  align-items: center;
}

.toolbar-divider {
  width: 1px;
  height: 20px;
  background: #e9ecef;
}

/* 搜索框样式 */
.industrial-input {
  width: 220px;
}

.industrial-input :deep(.arco-input-wrapper) {
  border-radius: 0;
  border-color: #dee2e6;
  background: #ffffff;
}

.industrial-input :deep(.arco-input-wrapper:hover) {
  border-color: #adb5bd;
}

.industrial-input :deep(.arco-input-wrapper:focus-within) {
  border-color: #495057;
  box-shadow: none;
}

/* 状态筛选 - 极简符号点+文字 */
.status-filters {
  display: flex;
  align-items: center;
  gap: 4px;
}

.status-filter-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  cursor: pointer;
  font-size: 12px;
  color: #6c757d;
  transition: all 0.2s ease;
  background: transparent;
}

.status-filter-item:hover {
  color: #495057;
  background: #f8f9fa;
}

.status-filter-item.active {
  color: #212529;
  background: #f8f9fa;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.dot-new {
  background: #2ecc71;
}

.dot-existing {
  background: #95a5a6;
}

.dot-all {
  background: #bdc3c7;
}

.status-label {
  font-size: 12px;
  font-weight: 500;
}

/* 工业风格复选框 */
.industrial-checkbox :deep(.arco-checkbox) {
  border-radius: 0;
}

.industrial-checkbox :deep(.arco-checkbox-icon) {
  border-radius: 0;
}

/* 扫描按钮 */
.scan-btn {
  background: #212529 !important;
  border: none;
  border-radius: 0;
  padding: 4px 16px;
  font-size: 12px;
}

.scan-btn:hover {
  background: #343a40 !important;
}

/* 统计信息栏 */
.stats-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px 16px;
  background: #fafbfc;
  border: 1px solid #e9ecef;
  margin-bottom: 12px;
}

.stats-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.stats-number {
  font-size: 13px;
  font-weight: 600;
  color: #212529;
  font-family: 'JetBrains Mono', monospace;
}

.stats-unit {
  font-size: 11px;
  color: #6c757d;
}

.stats-divider {
  width: 1px;
  height: 14px;
  background: #dee2e6;
}

/* 表格样式 */
.table-wrapper {
  border: 1px solid #e9ecef;
  background: #ffffff;
  display: flex;
  flex-direction: column;
}

.table-wrapper .arco-spin {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.table-wrapper .arco-spin .w-full {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.industrial-table :deep(.arco-table-th) {
  background: #fafbfc;
  border-bottom: 1px solid #e9ecef;
  color: #495057;
  font-size: 11px;
  font-weight: 600;
  padding: 12px 12px;
}

.industrial-table :deep(.arco-table-td) {
  padding: 10px 12px;
  font-size: 12px;
  border-bottom: 1px solid #f1f3f5;
}

.industrial-table :deep(.arco-table-tr:hover .arco-table-td) {
  background: #f8f9fa;
}

/* 选择框容器 - 水平居中对齐 */
.header-checkbox-wrapper,
.row-checkbox-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

/* 表头选择框容器 - 包含复选框和文字按钮 */
.selection-header {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
}

.selection-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: normal;
}

.select-all-text,
.clear-all-text {
  cursor: pointer;
  color: #6c757d;
  transition: color 0.2s ease;
  user-select: none;
}

.select-all-text:hover,
.clear-all-text:hover {
  color: #212529;
}

.select-divider {
  color: #dee2e6;
  margin: 0 2px;
}

/* 状态单元格 */
.status-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-text {
  font-size: 11px;
  color: #6c757d;
}

/* 节点名称单元格 */
.node-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.node-icon {
  font-size: 12px;
  color: #868e96;
  flex-shrink: 0;
}

.node-name {
  font-size: 12px;
  color: #212529;
  font-family: 'JetBrains Mono', monospace;
}

.font-mono {
  font-family: 'JetBrains Mono', monospace;
}

/* 空状态 */
.empty-placeholder {
  height: 360px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  background: #fafbfc;
  width: 100%;
}

.empty-icon {
  color: #ced4da;
}

.empty-text {
  margin-top: 12px;
  font-size: 12px;
  color: #adb5bd;
}

/* Footer 样式 */
.modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 8px;
}

.footer-info {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.footer-label {
  font-size: 11px;
  color: #6c757d;
}

.footer-number {
  font-size: 14px;
  font-weight: 600;
  color: #212529;
  font-family: 'JetBrains Mono', monospace;
}

.footer-unit {
  font-size: 11px;
  color: #868e96;
}

.footer-actions {
  display: flex;
  gap: 12px;
}

.cancel-btn {
  border-radius: 0;
  border-color: #dee2e6;
  color: #495057;
  font-size: 12px;
}

.cancel-btn:hover {
  border-color: #adb5bd;
  color: #212529;
}

.confirm-btn {
  background: #212529 !important;
  border: none;
  border-radius: 0;
  font-size: 12px;
}

.confirm-btn:hover:not(:disabled) {
  background: #343a40 !important;
}

.confirm-btn:disabled {
  background: #e9ecef !important;
  color: #adb5bd;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .scanner-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  
  .toolbar-left {
    flex-wrap: wrap;
    gap: 12px;
  }
  
  .toolbar-divider {
    display: none;
  }
  
  .status-filters {
    flex-wrap: wrap;
  }
  
  .industrial-input {
    width: 100%;
  }
  
  .modal-footer {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
  
  .footer-info {
    justify-content: center;
  }
  
  .footer-actions {
    justify-content: center;
  }
  
  .selection-header {
    flex-direction: column;
    gap: 4px;
  }
}
</style>

