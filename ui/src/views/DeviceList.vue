<template>
  <div class="device-list-container">
    <div class="device-header">
      <div class="header-left">
        <a-button type="outline" size="small" @click="router.push('/channels')">
          <template #icon><IconArrowLeft /></template>
          返回通道
        </a-button>
        <div class="header-info">
          <span class="protocol-tag">{{ channelProtocol || 'UNKNOWN' }}</span>
          <h2 class="title-text">设备列表</h2>
        </div>
      </div>
      
      <div class="header-right">
        <a-space size="small">
          <a-button v-if="selected.length > 0" status="danger" type="outline" size="small" @click="confirmBatchDelete">
            <template #icon><IconDelete /></template>
            批量删除 ({{ selected.length }})
          </a-button>
          <a-button v-if="channelProtocol === 'bacnet-ip' || channelProtocol === 'opc-ua'" type="outline" status="success" size="small" @click="openScanDialog()">
            <template #icon><IconScan /></template>
            扫描设备
          </a-button>
          <a-button type="outline" status="primary" size="small" @click="openDialog()">
            <template #icon><IconPlus /></template>
            新增设备
          </a-button>
        </a-space>
      </div>
    </div>

    <a-spin :loading="loading" style="width: 100%">
      <a-card class="industrial-card borderless-card">
        <a-table
          :columns="tableColumns"
          :data="devices"
          :loading="loading"
          :row-selection="rowSelection"
          v-model:selected-keys="selected"
          row-key="id"
          size="small"
          :bordered="{ cell: true }"
          :pagination="{ showTotal: true, showPageSize: true }"
        >
          <template #enable="{ record }">
            <a-switch 
              v-model="record.enable" 
              size="small" 
              @change="toggleDeviceStatus(record)"
              :loading="record.statusLoading"
            />
          </template>

          <template #name="{ record }">
            <div class="device-name-cell">
              <span class="main-name">{{ record.name }}</span>
              <span class="sub-id">ID: {{ record.id }}</span>
            </div>
          </template>

          <template #interval="{ record }">
            <a-tag size="small" bordered>
              <IconClockCircle :size="12" style="margin-right: 4px" />
              {{ record.interval }}
            </a-tag>
          </template>

          <template #state="{ record }">
            <a-tag :color="getDeviceStateColor(record.state)" size="small">
              {{ getDeviceStateText(record.state) }}
            </a-tag>
          </template>

          <template #quality="{ record }">
            <a-tag 
              v-if="channelProtocol && (channelProtocol.includes('bacnet') || channelProtocol === 'bacnet-ip')"
              :color="getQualityColor(record.quality_score)" 
              size="small"
            >
              {{ record.quality_score !== undefined ? record.quality_score : '-' }} ({{ getQualityLabel(record.quality_score) }})
            </a-tag>
          </template>

          <template #actions="{ record }">
            <a-space size="mini">
              <a-tooltip content="查看点位">
                <a-button type="text" size="mini" @click="goToPoints(record)">
                  <IconEye :size="14" />
                </a-button>
              </a-tooltip>
              <a-tooltip content="规则链">
                <a-button type="text" size="mini" @click="showRuleUsage(record)">
                  <IconLink :size="14" />
                </a-button>
              </a-tooltip>
              <a-tooltip content="历史数据">
                <a-button type="text" size="mini" @click="openHistoryDialog(record)">
                  <IconClockCircle :size="14" />
                </a-button>
              </a-tooltip>
              <a-divider direction="vertical" />
              <a-button type="text" size="mini" @click="openDialog(record)">编辑</a-button>
              <a-button type="text" size="mini" status="danger" @click="confirmDelete(record)">删除</a-button>
            </a-space>
          </template>
        </a-table>
      </a-card>
    </a-spin>

    <div class="device-footer">
      <div class="terminal-info">
        <span class="terminal-dot"></span>
        <span class="monospace-text">CHANNEL_CONTEXT: {{ channelId }} | DEVICES_COUNT: {{ devices.length }}</span>
      </div>
    </div>

    <a-modal v-model:visible="dialog" :title="form.id && isEdit ? '编辑设备' : '新增设备'" width="800px" @ok="saveDevice" @cancel="closeDialog">
      <a-form :model="form" layout="horizontal" :label-col-props="{ span: 6 }" :wrapper-col-props="{ span: 18 }">
        <a-form-item field="id" label="设备ID" required>
          <a-input v-model="form.id" placeholder="设备唯一标识" :disabled="isEdit" />
        </a-form-item>
        <a-form-item field="name" label="设备名称" required>
          <a-input v-model="form.name" placeholder="例如: 智能电表_01" />
        </a-form-item>
        <a-form-item field="interval" label="采集间隔" required>
          <a-input v-model="form.interval" placeholder="例如: 5s, 1m" />
        </a-form-item>
        <a-form-item field="enable" label="启用状态">
          <a-switch v-model="form.enable" />
        </a-form-item>
        
        <a-divider orientation="left">通信配置</a-divider>
        
        <template v-if="channelProtocol === 'dlt645'">
          <a-form-item field="dlt645Address" label="设备地址" required>
            <a-input v-model="form.dlt645Address" placeholder="210220003011" />
          </a-form-item>
        </template>
        
        <template v-if="channelProtocol && channelProtocol.includes('modbus')">
          <a-form-item field="modbusSlaveId" label="从机ID" required>
            <a-input-number v-model="form.modbusSlaveId" :min="1" placeholder="1" />
          </a-form-item>
          <a-form-item field="startAddressMode" label="地址模式">
            <a-radio-group v-model="form.startAddressMode">
              <a-radio :value="0">0-based</a-radio>
              <a-radio :value="1">1-based</a-radio>
            </a-radio-group>
          </a-form-item>
        </template>
        
        <template v-if="channelProtocol === 'bacnet-ip'">
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item field="bacnetDeviceInstance" label="实例ID" required>
                <a-input-number v-model="form.bacnetDeviceInstance" placeholder="1001" />
              </a-form-item>
            </a-col>
            <a-col :span="10">
              <a-form-item field="bacnetIp" label="IP地址">
                <a-input v-model="form.bacnetIp" placeholder="192.168.1.100" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item field="bacnetPort" label="端口">
                <a-input-number v-model="form.bacnetPort" placeholder="47808" />
              </a-form-item>
            </a-col>
          </a-row>
        </template>
        
        <template v-if="channelProtocol === 'opc-ua'">
          <a-form-item field="endpoint" label="Endpoint URL">
            <a-input v-model="form.config.endpoint" placeholder="opc.tcp://192.168.1.10:4840" />
          </a-form-item>
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item field="security_policy" label="安全策略">
                <a-select v-model="form.config.security_policy" :options="[
                  { label: 'None', value: 'None' },
                  { label: 'Basic128Rsa15', value: 'Basic128Rsa15' },
                  { label: 'Basic256', value: 'Basic256' },
                  { label: 'Basic256Sha256', value: 'Basic256Sha256' }
                ]" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item field="security_mode" label="安全模式">
                <a-select v-model="form.config.security_mode" :options="[
                  { label: 'None', value: 'None' },
                  { label: 'Sign', value: 'Sign' },
                  { label: 'SignAndEncrypt', value: 'SignAndEncrypt' }
                ]" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item field="auth_method" label="认证方式">
                <a-select v-model="form.config.auth_method" :options="[
                  { label: 'Anonymous', value: 'Anonymous' },
                  { label: 'UserName', value: 'UserName' },
                  { label: 'Certificate', value: 'Certificate' }
                ]" />
              </a-form-item>
            </a-col>
          </a-row>
          
          <template v-if="form.config.auth_method === 'UserName'">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item field="username" label="用户名">
                  <a-input v-model="form.config.username" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item field="password" label="密码">
                  <a-input-password v-model="form.config.password" />
                </a-form-item>
              </a-col>
            </a-row>
          </template>
          
          <template v-if="form.config.auth_method === 'Certificate'">
            <a-form-item field="certificate_file" label="证书路径">
              <a-input v-model="form.config.certificate_file" />
            </a-form-item>
            <a-form-item field="private_key_file" label="私钥路径">
              <a-input v-model="form.config.private_key_file" />
            </a-form-item>
          </template>
        </template>
        
        <a-divider orientation="left">数据存储策略</a-divider>
        
        <a-form-item field="storageEnable" label="启用存储">
          <a-switch v-model="form.storageEnable" />
        </a-form-item>
        
        <template v-if="form.storageEnable">
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item field="storageStrategy" label="存储策略">
                <a-select v-model="form.storageStrategy" :options="[
                  { label: '实时 (每条)', value: 'realtime' },
                  { label: '定时间隔', value: 'interval' }
                ]" />
              </a-form-item>
            </a-col>
            <a-col :span="8" v-if="form.storageStrategy === 'interval'">
              <a-form-item field="storageInterval" label="存储间隔">
                <a-input-number v-model="form.storageInterval" :min="1" suffix="分钟" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item field="storageMaxRecords" label="最大记录数">
                <a-input-number v-model="form.storageMaxRecords" :min="1" placeholder="1000" />
              </a-form-item>
            </a-col>
          </a-row>
        </template>
        
        <a-divider orientation="left">高级配置</a-divider>
        
        <a-form-item field="configStr" label="JSON配置">
          <a-textarea v-model="form.configStr" placeholder='{"key": "value"}' :auto-size="{ minRows: 5, maxRows: 10 }" />
        </a-form-item>
      </a-form>
    </a-modal>

    <HistoryModal v-model:visible="historyModalVisible" :device="historyDevice" />

    <a-modal v-model:visible="deleteDialog" title="确认删除" @ok="executeDelete" @cancel="deleteDialog = false">
      <p>{{ itemToDelete ? '确定要删除该设备吗？' : `确定要删除选中的 ${selected.length} 个设备吗？` }}此操作无法撤销。</p>
    </a-modal>

    <a-modal v-model:visible="ruleUsageDialog.show" title="关联规则" width="80%" @cancel="ruleUsageDialog.show = false">
      <a-list v-if="ruleUsageDialog.rules.length > 0" :data="ruleUsageDialog.rules">
        <template #item="{ item }">
          <a-list-item>
            <a-list-item-meta :title="item.name" :description="item.id">
              <template #avatar>
                <icon-link :size="24" />
              </template>
            </a-list-item-meta>
            <template #actions>
              <a-button type="text" @click="goToRule(item.id)">查看配置</a-button>
            </template>
          </a-list-item>
        </template>
      </a-list>
      <a-empty v-else description="该设备未被任何规则引用" />
    </a-modal>

    <a-modal v-model:visible="scanDialog" title="扫描设备" width="1200px" @cancel="scanDialog = false" :mask-closable="false">
      <template #footer>
        <a-space>
          <a-button @click="scanDialog = false" :disabled="isScanning">取消</a-button>
          <a-button type="primary" :loading="isAddingDevices" :disabled="selectedScanDevices.length === 0" @click="addSelectedDevices">
            添加选定设备 ({{ selectedScanDevices.length }})
          </a-button>
        </a-space>
      </template>
      
      <a-space direction="vertical" :size="16" fill>
        <a-alert type="info" :closable="false">
          <template #icon>
            <icon-info-circle />
          </template>
          点击"开始扫描"以发现网络中的设备。扫描可能需要10秒左右，请耐心等待。
        </a-alert>
        
        <a-space>
          <a-button type="outline" status="primary" :loading="isScanning" :disabled="isScanning" @click="scanDevices">
            <template #icon>
              <icon-scan />
            </template>
            开始扫描
          </a-button>
          <a-text v-if="isScanning" type="secondary" style="line-height: 32px;">{{ scanStatus }}</a-text>
        </a-space>
        

        
        <a-table 
          :columns="scanColumns" 
          :data="scanResults" 
          :loading="isScanning" 
          :row-selection="scanRowSelection"
          v-model:selected-keys="selectedScanDevices"
          row-key="device_id"
          size="small"
          :bordered="{ cell: true }"
          :pagination="false"
        >
          <template #status="{ record }">
            <a-tag v-if="record.diff_status === 'new'" color="green">New</a-tag>
            <a-tag v-else-if="record.diff_status === 'existing'" color="orange">Existing</a-tag>
            <a-tag v-else-if="record.diff_status === 'removed'" color="red">Removed</a-tag>
          </template>
          <template #empty>
            <div v-if="isScanning" class="text-center py-8">
              <a-spin size="large" />
              <div class="mt-4 text-gray">{{ scanStatus }}</div>
              <div class="mt-2 text-gray text-sm">预计需要10秒左右</div>
            </div>
            <a-empty v-else description="暂无扫描结果" />
          </template>
        </a-table>
      </a-space>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import {
  IconArrowLeft, IconPlus, IconDelete, IconScan, IconList,
  IconSettings, IconHistory, IconSearch, IconEye, IconLink,
  IconClockCircle, IconInfoCircle, IconCheckCircle, IconCloseCircle
} from '@arco-design/web-vue/es/icon'
import request from '@/utils/request'
import { base64ToUint8Array, uint8ArrayToHex, detectFileType, downloadBytes } from '@/utils/decode'
import HistoryModal from '@/components/HistoryModal.vue'

const route = useRoute()
const router = useRouter()
const devices = ref([])
const channelInfo = ref(null)
const loading = ref(false)
const channelId = route.params.channelId
const channelProtocol = computed(() => channelInfo.value?.protocol || '')

const selected = ref([])
const selectAll = ref(false)
const dialog = ref(false)
const deleteDialog = ref(false)
const isEdit = ref(false)
const itemToDelete = ref(null)

const ruleUsageDialog = reactive({
  show: false,
  deviceName: '',
  rules: []
})
const allRules = ref([])

const fetchRules = async () => {
  try {
    const data = await request.get('/api/edge/rules')
    allRules.value = data
  } catch (e) {
    console.error('Failed to fetch rules', e)
  }
}

const showRuleUsage = (device) => {
  ruleUsageDialog.deviceName = device.name
  ruleUsageDialog.rules = allRules.value.filter(rule => {
    if (rule.source && rule.source.device_id === device.id) return true
    if (rule.sources && rule.sources.some(s => s.device_id === device.id)) return true
    
    if (rule.actions) {
      return rule.actions.some(a => {
        if (a.config && a.config.device_id === device.id) return true
        if (a.config && a.config.targets && a.config.targets.some(t => t.device_id === device.id)) return true
        return false
      })
    }
    return false
  })
  ruleUsageDialog.show = true
}

const goToRule = (ruleId) => {
  router.push({ path: '/edge-compute', query: { rule: ruleId } })
}

const getDeviceStateColor = (state) => {
  switch (state) {
    case 0: return 'green'
    case 1: return 'orange'
    case 2: return 'red'
    case 3: return 'gray'
    default: return 'gray'
  }
}

const getDeviceStateText = (state) => {
  switch (state) {
    case 0: return '在线'
    case 1: return '不稳定'
    case 2: return '离线'
    case 3: return '隔离'
    default: return '未知'
  }
}

const getQualityColor = (score) => {
  if (score === undefined || score === null) return 'gray'
  if (score === 100) return 'blue'
  if (score >= 90) return 'green'
  if (score >= 80) return 'cyan'
  if (score >= 60) return 'orange'
  return 'red'
}

const getQualityLabel = (score) => {
  if (score === undefined || score === null) return 'Unknown'
  if (score === 100) return 'Perfect'
  if (score >= 90) return 'Excellent'
  if (score >= 80) return 'Good'
  if (score >= 60) return 'Average'
  return 'Bad'
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

const getConfigValue = (device, field) => {
  if (!device || !device.config) return '-'
  let config = device.config
  if (typeof config === 'string') {
    try {
      config = JSON.parse(config)
    } catch (e) {
      return '-'
    }
  }
  if (field === 'instance_id') {
    return config.bacnetDeviceInstance || config.device_id || config.InstanceID || config.instance_id || '-'
  }
  if (field === 'ip') {
    return config.bacnetIp || config.ip || '-'
  }
  if (field === 'vendor_model') {
    const vendor = config.vendor_name || '-'
    const model = config.model_name || '-'
    return `${vendor} / ${model}`
  }
  return config[field] || '-'
}

const defaultForm = {
  id: '',
  name: '',
  interval: '1s',
  enable: true,
  configStr: '{}',
  dlt645Address: '',
  modbusSlaveId: 1,
  startAddressMode: 0,
  bacnetDeviceInstance: 0,
  bacnetIp: '',
  bacnetPort: 47808,
  config: {},
  storageEnable: false,
  storageStrategy: 'interval',
  storageInterval: 1,
  storageMaxRecords: 1000
}
const form = ref({ ...defaultForm })

const fetchDevices = async () => {
  loading.value = true
  try {
    const chanData = await request.get(`/api/channels/${channelId}`)
    channelInfo.value = chanData

    const devData = await request.get(`/api/channels/${channelId}/devices`)
    devices.value = devData
    
    selected.value = []
    selectAll.value = false
  } catch (e) {
    Message.error('获取设备失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

const toggleSelectAll = (val) => {
  if (val) {
    selected.value = devices.value.map(d => d.id)
  } else {
    selected.value = []
  }
}

const openDialog = (item = null) => {
  if (item) {
    isEdit.value = true
    let config = item.config || {}
    if (typeof config === 'string') {
      try {
        config = JSON.parse(config)
      } catch (e) {
        config = {}
      }
    }
    const storage = item.storage || {}
    form.value = {
      ...item,
      config: config,
      configStr: JSON.stringify(config, null, 2),
      dlt645Address: config.station_address || config.address || '',
      modbusSlaveId: config.slave_id || 1,
      startAddressMode: config.start_address || config.address_base || 0,
      bacnetDeviceInstance: config.bacnetDeviceInstance || config.device_id || config.InstanceID || config.instance_id || 0,
      bacnetIp: config.bacnetIp || config.ip || '',
      bacnetPort: config.bacnetPort || config.port || 47808,
      storageEnable: storage.enable || false,
      storageStrategy: storage.strategy || 'interval',
      storageInterval: storage.interval || 1,
      storageMaxRecords: storage.max_records || 1000
    }
  } else {
    isEdit.value = false
    form.value = { ...defaultForm }
    if (channelProtocol.value === 'opc-ua') {
      form.value.config = {
        endpoint: 'opc.tcp://127.0.0.1:4840',
        security_policy: 'None',
        security_mode: 'None',
        auth_method: 'Anonymous'
      }
    }
  }
  dialog.value = true
}

const closeDialog = () => {
  dialog.value = false
  form.value = { ...defaultForm }
}

const saveDevice = async () => {
  let config = {}
  try {
    config = JSON.parse(form.value.configStr)
  } catch (e) {
    Message.error('配置参数必须是有效的JSON格式')
    return
  }

  if (channelProtocol.value === 'dlt645') {
    config.station_address = form.value.dlt645Address
    config.address = form.value.dlt645Address 
  } else if (channelProtocol.value && channelProtocol.value.includes('modbus')) {
    config.slave_id = form.value.modbusSlaveId
    config.start_address = form.value.startAddressMode
  } else if (channelProtocol.value === 'bacnet-ip') {
    config.device_id = form.value.bacnetDeviceInstance
    config.bacnetDeviceInstance = form.value.bacnetDeviceInstance
    if (form.value.bacnetIp) {
      config.ip = form.value.bacnetIp
      config.bacnetIp = form.value.bacnetIp
    }
    if (form.value.bacnetPort) {
      config.port = form.value.bacnetPort
      config.bacnetPort = form.value.bacnetPort
    }
  } else if (channelProtocol.value === 'opc-ua') {
    Object.assign(config, form.value.config)
  }

  const payload = {
    id: form.value.id,
    name: form.value.name,
    interval: form.value.interval,
    enable: form.value.enable,
    config: config,
    storage: {
      enable: form.value.storageEnable,
      strategy: form.value.storageStrategy,
      interval: form.value.storageInterval,
      max_records: form.value.storageMaxRecords
    },
    points: isEdit.value ? undefined : [] 
  }
  
  if (isEdit.value) {
    const original = devices.value.find(d => d.id === form.value.id)
    if (original) {
      payload.points = original.points
    }
  }

  try {
    const url = `/api/channels/${channelId}/devices` + (isEdit.value ? `/${form.value.id}` : '')
    const method = isEdit.value ? 'put' : 'post'
    
    await request({
      url: url,
      method: method,
      data: payload
    })

    Message.success(isEdit.value ? '更新成功' : '创建成功')
    closeDialog()
    fetchDevices()
  } catch (e) {
    Message.error(e.message)
  }
}

const confirmDelete = (item) => {
  itemToDelete.value = item
  deleteDialog.value = true
}

const confirmBatchDelete = () => {
  itemToDelete.value = null
  deleteDialog.value = true
}

const executeDelete = async () => {
  try {
    if (itemToDelete.value) {
      await request.delete(`/api/channels/${channelId}/devices/${itemToDelete.value.id}`)
    } else {
      await request({
        url: `/api/channels/${channelId}/devices`,
        method: 'delete',
        data: selected.value
      })
    }
    
    Message.success('删除成功')
    deleteDialog.value = false
    fetchDevices()
  } catch (e) {
    Message.error(e.message)
  }
}

const historyModalVisible = ref(false)
const historyDevice = ref(null)

const openHistoryDialog = (device) => {
  historyDevice.value = device
  historyModalVisible.value = true
}

const goToPoints = (device) => {
  router.push(`/channels/${channelId}/devices/${device.id}/points`)
}

const scanDialog = ref(false)
const isScanning = ref(false)
const scanResults = ref([])
const selectedScanDevices = ref([])
const selectAllScan = ref(false)
const isAddingDevices = ref(false)
const interfaces = ref([])
const scanInterface = ref(null)

const scanStatus = ref('')
const scanTimeout = ref(null)

const fetchInterfaces = async () => {
  try {
    const res = await request.get('/api/system/network/interfaces')
    interfaces.value = res || []
  } catch (e) {
    console.error('Failed to fetch interfaces', e)
  }
}

const openScanDialog = () => {
  scanDialog.value = true
  scanResults.value = []
  selectedScanDevices.value = []
  selectAllScan.value = false
  
  if (channelProtocol.value === 'bacnet-ip') {
    const configuredIP = channelInfo.value?.config?.ip
    if (configuredIP && configuredIP !== '0.0.0.0') {
      scanInterface.value = configuredIP
    } else {
      scanInterface.value = null
    }
    fetchInterfaces()
  } else if (channelProtocol.value === 'opc-ua') {
    // For OPC-UA, we don't need network interfaces, but we might need to prompt for endpoint
    scanInterface.value = null
  }
}

const scanDevices = async () => {
  isScanning.value = true
  scanResults.value = []
  selectedScanDevices.value = []
  scanStatus.value = '正在准备扫描...'
  
  let stopMessage = null
  // 显示扫描开始提示
  stopMessage = Message.loading({
    content: '开始扫描设备，可能需要10秒左右，请耐心等待...',
    duration: 0
  })
  
  // 设置扫描开始时间，用于计算耗时
  const startTime = Date.now()
  
  // 清除之前的超时
  if (scanTimeout.value) {
    clearInterval(scanTimeout.value)
  }
  
  // 模拟状态更新
  scanTimeout.value = setInterval(() => {
    const elapsed = Math.round((Date.now() - startTime) / 1000)
    if (elapsed < 3) {
      scanStatus.value = '正在初始化扫描...'
    } else if (elapsed < 6) {
      scanStatus.value = '正在搜索网络设备...'
    } else if (elapsed < 9) {
      scanStatus.value = '正在识别设备信息...'
    } else {
      scanStatus.value = '正在整理扫描结果...'
    }
  }, 1000)
  
  try {
    console.log('开始扫描设备，channelId:', channelId)
    console.log('扫描接口:', scanInterface.value)
    
    // 构建扫描参数
    const scanParams = {}
    if (channelProtocol.value === 'bacnet-ip') {
      scanParams.interface_ip = scanInterface.value
    } else if (channelProtocol.value === 'opc-ua') {
      // For OPC-UA, we can add endpoint parameter if needed
      // For now, we'll use default endpoints
    }
    
    // 增加超时设置
    const res = await request.post(`/api/channels/${channelId}/scan`, scanParams, {
      timeout: 30000 // 30秒超时
    })
    
    console.log('扫描响应:', res)
    
    // 计算扫描耗时
    const scanTime = Math.round((Date.now() - startTime) / 1000)
    scanStatus.value = '扫描完成'
    
    // 处理后端响应格式 - 后端直接返回设备数组
    scanResults.value = Array.isArray(res) ? res : (res.devices || [])
    if (stopMessage && typeof stopMessage === 'object' && typeof stopMessage.close === 'function') {
      stopMessage.close()
      stopMessage = null
    }
    Message.success({
      content: `扫描完成 (耗时 ${scanTime} 秒)，发现 ${scanResults.value.length} 个设备，查看结果`,
      duration: 3000 // 3秒后自动消失
    })
  } catch (e) {
    console.error('扫描失败:', e)
    if (stopMessage && typeof stopMessage === 'object' && typeof stopMessage.close === 'function') {
      stopMessage.close()
      stopMessage = null
    }
    if (e.code === 'ECONNABORTED') {
      Message.error({
        content: '扫描超时，请检查网络连接或设备响应',
        duration: 3000
      })
    } else {
      Message.error({
        content: '扫描失败: ' + e.message,
        duration: 3000
      })
    }
  } finally {
    if (scanTimeout.value) {
      clearInterval(scanTimeout.value)
      scanTimeout.value = null
    }
    if (stopMessage && typeof stopMessage === 'object' && typeof stopMessage.close === 'function') {
      stopMessage.close()
      stopMessage = null
    }
    isScanning.value = false
    scanStatus.value = ''
  }
}

const addSelectedDevices = async () => {
  isAddingDevices.value = true
  
  try {
    for (const device of selectedScanDevices.value) {
      let config = {}
      if (channelProtocol.value === 'bacnet-ip') {
        config = {
          device_id: device.device_id,
          bacnetDeviceInstance: device.device_id,
          ip: device.ip,
          port: device.port,
          vendor_name: device.vendor_name,
          model_name: device.model_name
        }
      } else if (channelProtocol.value === 'opc-ua') {
        config = {
          endpoint: device.endpoint,
          name: device.name,
          vendor_name: device.vendor_name,
          model_name: device.model_name,
          version: device.version
        }
      }
      
      await request.post(`/api/channels/${channelId}/devices`, {
        id: device.device_id,
        name: device.name || device.device_id,
        interval: '10s',
        enable: true,
        config: config,
        points: []
      })
    }
    
    Message.success(`成功添加 ${selectedScanDevices.value.length} 个设备`)
    scanDialog.value = false
    fetchDevices()
  } catch (e) {
    Message.error('添加设备失败: ' + e.message)
  } finally {
    isAddingDevices.value = false
  }
}

const toggleDeviceStatus = async (record) => {
  record.statusLoading = true
  try {
    await request.put(`/api/channels/${channelId}/devices/${record.id}`, {
      ...record,
      enable: record.enable
    })
    Message.success('状态更新成功')
  } catch (e) {
    Message.error('状态更新失败: ' + e.message)
    record.enable = !record.enable
  } finally {
    record.statusLoading = false
  }
}

const tableColumns = computed(() => {
  const columns = [
    { title: '设备名称 / 标识', slotName: 'name', width: 220 },
    { title: '状态', slotName: 'enable', width: 100 },
    { title: '通信状态', slotName: 'state', width: 100 },
    { title: '采集间隔', slotName: 'interval', width: 120 },
  ]
  
  if (channelProtocol.value && (channelProtocol.value.includes('bacnet') || channelProtocol.value === 'bacnet-ip')) {
    columns.push({ title: '质量评分', slotName: 'quality', width: 120 })
  }
  
  columns.push({ title: '操作', slotName: 'actions', width: 240, fixed: 'right' })
  
  return columns
})

const rowSelection = reactive({
  type: 'checkbox',
  showCheckedAll: true,
  onlyCurrent: false,
})

const scanColumns = computed(() => {
  if (channelProtocol.value === 'opc-ua') {
    return [
      { title: 'Device ID', dataIndex: 'device_id', width: 150 },
      { title: 'Endpoint', dataIndex: 'endpoint', width: 300 },
      { title: '名称', dataIndex: 'name', width: 200 },
      { title: '厂商', dataIndex: 'vendor_name', width: 200 },
      { title: '型号', dataIndex: 'model_name', width: 150 },
      { title: '版本', dataIndex: 'version', width: 100 },
      { title: '状态', slotName: 'status', width: 100 },
    ]
  } else {
    return [
      { title: 'Device ID', dataIndex: 'device_id', width: 150 },
      { title: 'IP 地址', dataIndex: 'ip', width: 150 },
      { title: '端口', dataIndex: 'port', width: 100 },
      { title: '厂商', dataIndex: 'vendor_name', width: 200 },
      { title: '型号', dataIndex: 'model_name', width: 150 },
      { title: '对象名称', dataIndex: 'object_name', width: 200 },
      { title: '状态', slotName: 'status', width: 100 },
    ]
  }
})

const scanRowSelection = reactive({
  type: 'checkbox',
  showCheckedAll: true,
  onlyCurrent: false,
})

onMounted(() => {
  fetchDevices()
  fetchRules()
})
</script>

<style scoped>
.device-list-container {
  padding: 24px;
  background-color: #f1f5f9;
  min-height: calc(100vh - 56px);
}

.dark-theme .device-list-container {
  background-color: #0b1223 !important;
}

.dark-theme .device-header {
  border-color: #334155 !important;
}

.dark-theme .title-text,
.dark-theme .protocol-tag,
.dark-theme .main-name,
.dark-theme .sub-id,
.dark-theme .device-footer,
.dark-theme .terminal-info,
.dark-theme .terminal-dot,
.dark-theme .monospace-text,
.dark-theme .arco-table-th,
.dark-theme .arco-table-td,
.dark-theme .arco-form-item-label,
.dark-theme .arco-table-tr,
.dark-theme .arco-table-element {
  color: #f8fafc !important;
  background-color: #111827 !important;
  border-color: #334155 !important;
}

.dark-theme .industrial-card {
  border-color: #334155 !important;
  box-shadow: 6px 6px 0px #0f172a !important;
}

.dark-theme .arco-table-td.arco-table-td-row-select,
.dark-theme .arco-table-col-fixed-left .arco-table-th,
.dark-theme .arco-table-col-fixed-left .arco-table-td,
.dark-theme .arco-table-col-fixed-right .arco-table-th,
.dark-theme .arco-table-col-fixed-right .arco-table-td {
  background-color: #111827 !important;
  border-color: #334155 !important;
}

.device-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px dashed #cbd5e1;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-info {
  display: flex;
  flex-direction: column;
}

.protocol-tag {
  font-family: monospace;
  font-size: 10px;
  background: #0ea5e9;
  color: white;
  padding: 0 4px;
  width: fit-content;
  border-radius: 2px;
}

.title-text {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #1e293b;
  text-align: left;
}

.industrial-card {
  border: 1px solid #cbd5e1 !important;
  border-radius: 2px;
  box-shadow: 6px 6px 0px #e2e8f0;
}

/* 无边框卡片 */
.borderless-card {
  border: none !important;
  box-shadow: none !important;
}

/* 无边框卡片的内容区 */
.borderless-card :deep(.arco-card-body) {
  padding: 0 !important;
}

/* 分页组件背景颜色与页面背景一致 */
.borderless-card :deep(.arco-table-pagination) {
  background-color: #f1f5f9 !important;
  border-top: none !important;
  padding: 12px !important;
}

.dark-theme .borderless-card :deep(.arco-table-pagination),
.dark-theme .borderless-card :deep(.arco-pagination),
.dark-theme .borderless-card :deep(.arco-pagination .arco-pagination-list),
.dark-theme .borderless-card :deep(.arco-pagination .arco-pagination-item),
.dark-theme .borderless-card :deep(.arco-pagination .arco-pagination-total),
.dark-theme .borderless-card :deep(.arco-select-view) {
  background-color: #0f172a !important;
  border-top: 1px solid #334155 !important;
  color: #f8fafc !important;
}

.dark-theme .borderless-card :deep(.arco-pagination-item),
.dark-theme .borderless-card :deep(.arco-pagination-item-active),
.dark-theme .borderless-card :deep(.arco-pagination-item-disabled) {
  background-color: #1f2937 !important;
  color: #f8fafc !important;
}

.device-name-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.main-name {
  font-weight: 600;
  color: var(--color-text-1);
}

.sub-id {
  font-size: 11px;
  font-family: monospace;
  color: var(--color-text-3);
}

.device-footer {
  margin-top: 24px;
  padding: 12px;
  border: 1px dashed #cbd5e1;
  text-align: center;
  background-color: #f8fafc;
}

.terminal-info {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.terminal-dot {
  width: 6px;
  height: 6px;
  background: #0ea5e9;
  box-shadow: 0 0 4px #0ea5e9;
  border-radius: 50%;
}

.monospace-text {
  font-family: monospace;
  font-size: 11px;
  color: #1e293b;
  font-weight: 500;
}

:deep(.arco-form-item-label) {
  font-weight: 500;
  white-space: nowrap !important;
}

:deep(.arco-table-th) {
  background-color: #f8fafc !important;
  font-weight: bold !important;
  border-radius: 0 !important;
  padding: 12px !important;
  white-space: nowrap !important;
  overflow: visible !important;
}

:deep(.arco-table-td) {
  padding: 12px !important;
  border-radius: 0 !important;
}

:deep(.arco-table-tr) {
  height: 36px !important;
}

:deep(.arco-table-element) {
  border-radius: 0 !important;
  border: 1px solid #e5e7eb !important;
}

:deep(.arco-table-tr:hover) {
  background-color: #f9fafb !important;
}

:deep(.arco-table-td.arco-table-td-row-select) {
  background-color: #ffffff !important;
  border-right: 1px solid #e5e7eb !important;
}

:deep(.arco-table-col-fixed-left .arco-table-th) {
  background-color: #f8fafc !important;
  border-right: 1px solid #e5e7eb !important;
}

:deep(.arco-table-col-fixed-left .arco-table-td) {
  background-color: #ffffff !important;
  border-right: 1px solid #e5e7eb !important;
}

:deep(.arco-table-col-fixed-right .arco-table-th) {
  background-color: #f8fafc !important;
  border-left: 1px solid #e5e7eb !important;
}

:deep(.arco-table-col-fixed-right .arco-table-td) {
  background-color: #ffffff !important;
  border-left: 1px solid #e5e7eb !important;
  border-right: none !important;
}

:deep(.arco-table-container) {
  border-right: none !important;
  border-left: none !important;
}

@media (max-width: 768px) {
  .device-header {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .header-left {
    width: 100%;
  }
  
  .header-right {
    width: 100%;
    margin-top: 16px;
  }
  
  .industrial-card {
    overflow-x: auto;
  }
}
</style>
