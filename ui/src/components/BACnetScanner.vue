<template>
  <a-modal 
    :visible="props.visible" 
    @update:visible="handleClose" 
    :width="1200" 
    title-align="start" 
    :footer="false" 
    unmount-on-close 
    modal-class="industrial-modal"
  >
    <template #title>
      <div class="flex items-baseline gap-2">
        <span class="text-lg font-bold">点位扫描发现</span>
        <span class="text-xs font-mono text-slate-400">PROTOCOL: {{ props.channelProtocol.toUpperCase() }}</span>
      </div>
    </template>

    <div class="scan-container">
      <div class="scanner-toolbar">
        <div class="toolbar-left">
          <a-input-search 
            v-model="state.filterText" 
            placeholder="搜索 NodeID / 名称" 
            size="small" 
            style="width: 240px" 
            allow-clear 
            class="industrial-input"
          />
          <div class="toolbar-divider"></div>
          <!-- 状态筛选 - 极简符号点+文字 -->
          <div class="status-filters">
            <div 
              class="status-filter-item" 
              :class="{ active: state.filterStatus === 'all' }"
              @click="state.filterStatus = 'all'"
            >
              <span class="status-dot dot-all"></span>
              <span class="status-label">全部</span>
            </div>
            <div 
              class="status-filter-item" 
              :class="{ active: state.filterStatus === 'new' }"
              @click="state.filterStatus = 'new'"
            >
              <span class="status-dot dot-new"></span>
              <span class="status-label">新增</span>
            </div>
            <div 
              class="status-filter-item" 
              :class="{ active: state.filterStatus === 'exists' }"
              @click="state.filterStatus = 'exists'"
            >
              <span class="status-dot dot-existing"></span>
              <span class="status-label">存量</span>
            </div>
          </div>
        </div>
        <div class="toolbar-right">
          <a-button 
            @click="handleScan" 
            :loading="state.loading" 
            size="small" 
            class="scan-btn"
          >
            <template #icon><IconScan /></template> 重新扫描
          </a-button>
          <a-button 
            type="primary" 
            :disabled="!state.selectedKeys.length" 
            @click="handleAddSelected" 
            size="small" 
            class="scan-btn"
            style="margin-left: 8px"
          >
            导入选中点位 ({{ state.selectedKeys.length }})
          </a-button>
        </div>
      </div>

      <a-table 
        row-key="unique_id" 
        :loading="state.loading" 
        :columns="scanColumns" 
        :data="filteredScanResults" 
        :pagination="{ pageSize: 100, size: 'small' }" 
        :row-selection="{ type: 'checkbox', showCheckedAll: true }" 
        v-model:selectedKeys="state.selectedKeys" 
        :bordered="{ wrapper: true, cell: true }" 
        :scroll="{ y: 550 }" 
        class="industrial-table-fluid"
      >
        <template #status="{ record }">
          <a-tag v-if="record.is_exists" color="gray" size="mini" class="rect-tag">
            <template #icon><IconCheckCircle /></template>存量
          </a-tag>
          <a-tag v-else color="green" size="mini" class="rect-tag">
            <template #icon><IconPlus /></template>新增
          </a-tag>
        </template>

        <template #address="{ record }">
          <span class="font-mono text-[13px]">
            <template v-if="props.channelProtocol === 'bacnet-ip'">
              {{ record.type }}:{{ record.instance }}
            </template>
            <template v-else-if="props.channelProtocol === 'opc-ua'">
              {{ record.node_id }}
            </template>
          </span>
        </template>
      </a-table>
    </div>
  </a-modal>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { 
  IconScan, IconCheckCircle, IconPlus 
} from '@arco-design/web-vue/es/icon'
import request from '@/utils/request'
import { showMessage } from '../composables/useGlobalState'

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
  channelProtocol: {
    type: String,
    required: true
  },
  existingAddresses: {
    type: Set,
    required: true
  },
  deviceInfo: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:visible', 'refresh-points'])

const state = reactive({
  visible: false,
  loading: false,
  results: [],
  selectedKeys: [],
  filterText: '',
  filterStatus: 'all',
  mode: 'fast'
})

// 计算属性：实现多维度筛选
const filteredScanResults = computed(() => {
  return state.results.filter(item => {
    // 1. 文本搜索
    const matchText = !state.filterText || 
      (item.name && item.name.toLowerCase().includes(state.filterText.toLowerCase())) || 
      (item.object_name && item.object_name.toLowerCase().includes(state.filterText.toLowerCase())) ||
      (item.node_id && item.node_id.includes(state.filterText));
    
    // 2. 存量筛选 (基于 is_exists 属性)
    const matchStatus = 
      state.filterStatus === 'all' || 
      (state.filterStatus === 'new' && !item.is_exists) || 
      (state.filterStatus === 'exists' && item.is_exists);
      
    return matchText && matchStatus;
  });
});

// 计算新增点位数量
const countNewPoints = computed(() => 
  state.results.filter(i => !i.is_exists).length
);

// 统一的表格列定义
const scanColumns = computed(() => {
  const base = [
    { title: '状态', slotName: 'status', width: 90 },
    { title: '名称', dataIndex: 'name', ellipsis: true, tooltip: true },
    { title: '点位地址', slotName: 'address', width: 220 },
    { title: '类型', dataIndex: 'type', width: 120 },
    { title: '实例', dataIndex: 'instance', width: 80 },
    { title: '当前值', dataIndex: 'present_value', width: 100 },
    { title: '单位', dataIndex: 'units', width: 80 },
    { title: '描述/DataType', dataIndex: 'description', ellipsis: true, tooltip: true }
  ];

  return base;
});

// 处理扫描
const handleScan = async () => {
  state.loading = true
  state.results = []
  try {
    // 构建扫描请求参数
    const payload = {
      mode: state.mode
    }
    
    // 提取设备实例ID（如果有）
    let targetDeviceId = null
    if (props.deviceInfo?.config?.device_id) {
      targetDeviceId = props.deviceInfo.config.device_id
    } else if (props.deviceInfo?.config?.instance_id) {
      targetDeviceId = props.deviceInfo.config.instance_id
    } else {
      // 尝试从设备配置中提取
      const deviceIdParts = props.deviceId.split('-')
      if (deviceIdParts.length > 1) {
        const lastPart = deviceIdParts[deviceIdParts.length - 1]
        if (!isNaN(lastPart)) {
          targetDeviceId = lastPart
        }
      }
    }

    if (targetDeviceId === undefined || targetDeviceId === null || targetDeviceId === '') {
      showMessage('无法获取设备实例ID (config.instance_id 或 device_id)', 'error')
      return
    }

    // 如果提取到了 device_id (BACnet Instance ID)，则显式传递给后端
    if (targetDeviceId !== null) {
      payload.device_id = parseInt(targetDeviceId)
    }

    const res = await request.post(`/api/channels/${props.channelId}/devices/${props.deviceId}/scan`, payload, { timeout: 60000 })
    
    if (Array.isArray(res)) {
      if (res.length === 0) {
        showMessage('扫描结果为空', 'warning')
      }
      if (props.channelProtocol === 'opc-ua') {
        // Flatten OPC UA tree for display
        state.results = flattenOpcNodes(res)
      } else {
        // For BACnet (and others), process is_exists based on existing points in UI
        state.results = res.map(item => {
          if (props.channelProtocol === 'bacnet-ip') {
               const key = `${item.type}:${item.instance}`
               item.is_exists = props.existingAddresses.has(key)
               item.unique_id = key
               item.name = item.object_name || item.name
          }
          return item
        })
      }
    } else {
      showMessage('扫描结果格式错误', 'error')
    }
  } catch (e) {
    showMessage('扫描失败: ' + e.message, 'error')
  } finally {
    state.loading = false
  }
}

// 处理添加选定点位
const handleAddSelected = async () => {
  if (state.selectedKeys.length === 0) return
  
  state.loading = true
  let successCount = 0
  let failCount = 0
  
  // Find selected objects based on selectedKeys
  const selectedObjects = state.results.filter(obj => state.selectedKeys.includes(obj.unique_id))
  
  for (const obj of selectedObjects) {
    let pointPayload = {}

    if (props.channelProtocol === 'opc-ua') {
      // OPC UA Point Mapping
      // Skip non-variable nodes if desired, or let user decide (variables only usually)
      if (obj.type !== 'Variable') continue;
      
      let rw = 'R'
      if (obj.access_level && obj.access_level.includes('CurrentWrite')) {
        rw = 'RW'
      }
      
      // Map OPC UA DataType to System DataType
      let dt = (obj.data_type || 'Float').toLowerCase()
      if (dt.includes('bool')) dt = 'bool'
      else if (dt.includes('int16') || dt.includes('short')) dt = 'int16'
      else if (dt.includes('uint16') || dt.includes('unsignedshort')) dt = 'uint16'
      else if (dt.includes('int32') || dt.includes('int')) dt = 'int32'
      else if (dt.includes('uint32') || dt.includes('unsignedint')) dt = 'uint32'
      else if (dt.includes('float')) dt = 'float32'
      else if (dt.includes('double')) dt = 'float64'
      else if (dt.includes('string')) dt = 'string'
      else dt = 'float32' // Default fallback

      pointPayload = {
        id: obj.node_id, // Use NodeID as ID
        name: obj.display_name || obj.node_id,
        address: obj.node_id,
        datatype: dt,
        readwrite: rw,
        unit: '', // Units not always available in browse
        scale: 1.0,
        offset: 0.0
      }
    } else {
      // BACnet Point Mapping
      // Determine Datatype
      let datatype = 'float32'
      if (obj.type.includes('Binary') || obj.type.includes('Bit')) datatype = 'bool'
      if (obj.type.includes('MultiState')) datatype = 'uint16'
      
      // Determine RW
      let rw = 'R'
      if (obj.type.includes('Output') || obj.type.includes('Value')) rw = 'RW'
      
      pointPayload = {
        id: obj.name || `${obj.type}_${obj.instance}`.replace(/[\s:]+/g, '_'),
        name: obj.description || `${obj.type} ${obj.instance}`,
        address: `${obj.type}:${obj.instance}`,
        datatype: datatype,
        readwrite: rw,
        unit: obj.units || '',
        scale: 1.0,
        offset: 0.0
      }
    }

    try {
      await request.post(`/api/channels/${props.channelId}/devices/${props.deviceId}/points`, pointPayload)
      successCount++
    } catch (e) {
      console.error(e)
      failCount++
    }
  }
  
  state.loading = false
  showMessage(`已添加 ${successCount} 个点位${failCount > 0 ? `，${failCount} 个失败` : ''}`, failCount > 0 ? 'warning' : 'success')
  emit('update:visible', false)
  emit('refresh-points')
}

// 关闭弹窗
const handleClose = (value) => {
  if (!value) {
    emit('close')
    emit('refresh-points')
  }
}

// 扁平化OPC UA节点
const flattenOpcNodes = (nodes, level = 0) => {
  let result = []
  for (const node of nodes) {
    // Add current node
    const item = {
      ...node,
      level: level,
      isOpcNode: true,
      // Map to common fields for display
      device_id: node.node_id, // Use NodeID as ID
      object_name: node.name,
      name: node.name,
      type: node.type, // "Variable" or "Folder"
      description: node.node_id, // Show NodeID in description/extra
      unique_id: node.node_id
    }
    // Mark existing/new for sync status
    if (node.type === 'Variable' && node.node_id) {
      item.is_exists = props.existingAddresses.has(node.node_id)
    }
    result.push(item)
    
    // Process children
    if (node.children && node.children.length > 0) {
      result = result.concat(flattenOpcNodes(node.children, level + 1))
    }
  }
  return result
}
</script>

<style scoped>
/* --- 扫描工具栏样式 (与OPC-UA一致) --- */
.scanner-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background-color: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
  gap: 16px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
  min-width: 0;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-divider {
  width: 1px;
  height: 24px;
  background-color: #e2e8f0;
  margin: 0 8px;
}

.industrial-input {
  flex: 0 0 240px;
}

.status-filters {
  display: flex;
  align-items: center;
  gap: 16px;
}

.status-filter-item {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 0;
  transition: all 0.2s ease;
}

.status-filter-item:hover {
  background-color: #f1f5f9;
}

.status-filter-item.active {
  background-color: #e2e8f0;
  font-weight: 500;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.dot-all {
  background-color: #94a3b8;
}

.dot-new {
  background-color: #10b981;
}

.dot-existing {
  background-color: #6366f1;
}

.status-label {
  font-size: 14px;
  color: #475569;
}

.scan-btn {
  font-size: 14px;
}

.rect-tag {
  border-radius: 0;
  margin-right: 8px;
  white-space: nowrap;
}

/* --- 极简线构弹窗样式 (Industrial Modal) --- */

/* 1. 移除整体圆角，添加 1px 锐利边框和深色阴影 */
:deep(.industrial-modal) {
  border-radius: 0 !important;
  border: 1px solid #94a3b8 !important; /* Slate-400 */
  box-shadow: 4px 4px 0px rgba(0, 0, 0, 0.05) !important; /* 工业风硬阴影 */
}

/* 2. 标题栏 - 紧凑、左对齐、无背景 */
:deep(.industrial-modal .arco-modal-title) {
  padding: 12px 16px !important;
  margin: 0 !important;
  font-size: 16px !important;
  font-weight: 600 !important;
  color: #1e293b !important; /* Slate-800 */
  background: transparent !important;
  border-bottom: 1px solid #e2e8f0 !important; /* Slate-200 */
}

/* 3. 内容区 - 无内边距，与表格/卡片自然衔接 */
:deep(.industrial-modal .arco-modal-content) {
  padding: 0 !important;
  overflow: hidden !important;
  background: #ffffff !important;
}

/* 4. 移除默认关闭按钮的圆角，改为锐利边框 */
:deep(.industrial-modal .arco-btn-icon-only) {
  border-radius: 0 !important;
  border: 1px solid #e2e8f0 !important;
  width: 28px !important;
  height: 28px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
}

:deep(.industrial-modal .arco-btn-icon-only:hover) {
  background: #f1f5f9 !important; /* Slate-100 */
  border-color: #cbd5e1 !important; /* Slate-300 */
}
</style>
