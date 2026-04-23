<template>
  <a-card class="target-editor-card" :bordered="true">
    <a-row :gutter="12">
      <a-col :span="24" :md="6">
        <a-form-item label="通道">
          <a-select
            v-model="localTarget.channel_id"
            :options="channels"
            placeholder="选择通道"
            class="rect-input"
            @change="onChannelChange"
          />
        </a-form-item>
      </a-col>
      <a-col :span="24" :md="6">
        <a-form-item label="设备">
          <a-select
            v-model="localTarget.device_id"
            :options="deviceList"
            placeholder="选择设备"
            class="rect-input"
            :disabled="!localTarget.channel_id"
            @change="onDeviceChange"
          />
        </a-form-item>
      </a-col>
      <a-col :span="24" :md="6">
        <a-form-item label="点位">
          <a-select
            v-model="localTarget.point_id"
            :options="pointList"
            placeholder="选择点位"
            class="rect-input"
            :disabled="!localTarget.device_id"
          />
        </a-form-item>
      </a-col>
      <a-col :span="24" :md="6">
        <a-form-item label="写入值">
          <a-input
            v-model="localTarget.value"
            placeholder="固定值或表达式"
            class="rect-input"
          />
        </a-form-item>
      </a-col>
    </a-row>
  </a-card>
</template>

<script setup>
import { ref, watch } from 'vue'
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

const localTarget = ref(props.target || {})
const deviceList = ref([])
const pointList = ref([])

const toOptionLabel = (item, fallbackText) => {
  if (typeof item === 'string' || typeof item === 'number') return String(item)

  const candidates = [
    item?.name,
    item?.device_name,
    item?.point_name,
    item?.label,
    item?.id,
    item?.value
  ]

  const candidate = candidates.find((value) => value != null && String(value).trim() !== '')
  return candidate == null ? fallbackText : String(candidate)
}

const normalizeDeviceOptions = (data) => {
  return (Array.isArray(data) ? data : []).map((d) => ({
    label: toOptionLabel(d, 'Unnamed Device'),
    value: typeof d === 'string' || typeof d === 'number'
      ? String(d)
      : String(d?.id ?? d?.value ?? toOptionLabel(d, 'Unnamed Device')),
    raw: d
  }))
}

const normalizePointOptions = (points) => {
  return (Array.isArray(points) ? points : [])
    .filter((p) => p?.readwrite !== 'R')
    .map((p) => ({
      label: toOptionLabel(p, 'Unnamed Point'),
      value: typeof p === 'string' || typeof p === 'number'
        ? String(p)
        : String(p?.id ?? p?.value ?? toOptionLabel(p, 'Unnamed Point')),
      raw: p
    }))
}

watch(() => props.target, (val) => {
  if (val === localTarget.value) return

  localTarget.value = val || {}

  if (localTarget.value?.channel_id) {
    loadDevices()
  }
}, { immediate: true })

watch(localTarget, (val) => {
  emit('update:target', val)
}, { deep: true })

const onChannelChange = async () => {
  localTarget.value.device_id = ''
  localTarget.value.point_id = ''
  deviceList.value = []
  pointList.value = []

  if (!localTarget.value.channel_id) return

  const data = await request.get(`/api/channels/${localTarget.value.channel_id}/devices`)
  deviceList.value = normalizeDeviceOptions(data)
}

const onDeviceChange = () => {
  localTarget.value.point_id = ''
  pointList.value = []

  if (!localTarget.value.device_id || deviceList.value.length === 0) return

  const dev = deviceList.value.find((d) => String(d.value) === String(localTarget.value.device_id))
  const points = dev?.raw?.points || dev?.points || []
  pointList.value = normalizePointOptions(points)
}

const loadDevices = async () => {
  if (!localTarget.value?.channel_id || deviceList.value.length > 0) return

  const data = await request.get(`/api/channels/${localTarget.value.channel_id}/devices`)
  deviceList.value = normalizeDeviceOptions(data)

  if (localTarget.value.device_id) {
    onDeviceChange()
  }
}
</script>

<style scoped>
.target-editor-card {
  border-left: 4px solid #165dff;
  border-radius: 0;
  background: #f8fafc;
}
</style>


