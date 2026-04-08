<template>
  <a-card class="target-editor-card" :bordered="true">
    <a-row :gutter="12">
      <a-col :span="24" :md="6">
        <a-form-item label="通道">
          <a-select 
            v-model="target.channel_id" 
            :options="channels" 
            :label-field="'name'" 
            :value-field="'id'" 
            placeholder="选择通道"
            class="rect-input"
            @change="onChannelChange"
          />
        </a-form-item>
      </a-col>
      <a-col :span="24" :md="6">
        <a-form-item label="设备">
          <a-select 
            v-model="target.device_id" 
            :options="deviceList" 
            :label-field="'name'" 
            :value-field="'id'" 
            placeholder="选择设备"
            class="rect-input"
            :disabled="!target.channel_id"
            @change="onDeviceChange"
          />
        </a-form-item>
      </a-col>
      <a-col :span="24" :md="6">
        <a-form-item label="点位">
          <a-select 
            v-model="target.point_id" 
            :options="pointList" 
            :label-field="'name'" 
            :value-field="'id'" 
            placeholder="选择点位"
            class="rect-input"
            :disabled="!target.device_id"
          />
        </a-form-item>
      </a-col>
      <a-col :span="24" :md="6">
        <a-form-item label="写入值">
          <a-input 
            v-model="target.value" 
            placeholder="固定值或表达式" 
            class="rect-input"
          />
        </a-form-item>
      </a-col>
    </a-row>
  </a-card>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import request from '@/utils/request'

const props = defineProps({
  target: {
    type: Object,
    required: true
  },
  channels: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:target'])

const localTarget = ref(props.target)
const deviceList = ref([])
const pointList = ref([])

// Sync props to local state
watch(() => props.target, (val) => {
  localTarget.value = val
  // Load devices/points if needed
  if (localTarget.value.channel_id) {
    loadDevices()
  }
}, { deep: true })

// Sync local state to props
watch(localTarget, (val) => {
  emit('update:target', val)
}, { deep: true })

const onChannelChange = async () => {
    localTarget.value.device_id = ''
    localTarget.value.point_id = ''
    deviceList.value = []
    pointList.value = []
    if (localTarget.value.channel_id) {
        const data = await request.get(`/api/channels/${localTarget.value.channel_id}/devices`)
        deviceList.value = Array.isArray(data) ? data : []
    }
}

const onDeviceChange = () => {
    localTarget.value.point_id = ''
    pointList.value = []
    if (localTarget.value.device_id && deviceList.value.length > 0) {
        const dev = deviceList.value.find(d => d.id === localTarget.value.device_id)
        if (dev && dev.points) {
            pointList.value = dev.points.filter(p => p.readwrite !== 'R')
        }
    }
}

const loadDevices = async () => {
    if (localTarget.value.channel_id && deviceList.value.length === 0) {
        const data = await request.get(`/api/channels/${localTarget.value.channel_id}/devices`)
        deviceList.value = Array.isArray(data) ? data : []
        if (localTarget.value.device_id) {
            onDeviceChange()
        }
    }
}

onMounted(() => {
    // Init device list loading
    if (localTarget.value.channel_id) {
        loadDevices()
    }
})
</script>

<style scoped>
.target-editor-card {
    border-left: 4px solid #165DFF;
    border-radius: 0;
    background: #f8fafc;
}
</style>