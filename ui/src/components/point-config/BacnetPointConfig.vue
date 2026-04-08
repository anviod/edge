<template>
  <div class="bacnet-point-config">
    <a-row :gutter="16">
      <a-col :span="12">
        <a-form-item field="objectType" label="对象类型">
          <a-select
            v-model="form.object_type"
            :options="objectTypes"
            placeholder="选择对象类型"
          ></a-select>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item field="instance" label="实例号">
          <a-input
            v-model.number="form.instance"
            type="number"
            min="0"
            placeholder="例如: 1"
            :tooltip="{ title: 'BACnet对象实例号', placement: 'top' }"
          ></a-input>
        </a-form-item>
      </a-col>
    </a-row>
    
    <a-row :gutter="16">
      <a-col :span="12">
        <a-form-item field="propertyId" label="属性ID">
          <a-input
            v-model.number="form.property_id"
            type="number"
            min="0"
            placeholder="例如: 85"
            :tooltip="{ title: 'BACnet属性标识符', placement: 'top' }"
          ></a-input>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item field="priority" label="写入优先级">
          <a-select
            v-model.number="form.priority"
            :options="priorities"
            placeholder="选择优先级"
            :tooltip="{ title: 'BACnet写入优先级', placement: 'top' }"
          ></a-select>
        </a-form-item>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  form: {
    type: Object,
    required: true
  },
  deviceInfo: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:form'])

const objectTypes = [
  { label: 'Analog Input', value: 'analog-input' },
  { label: 'Analog Output', value: 'analog-output' },
  { label: 'Analog Value', value: 'analog-value' },
  { label: 'Binary Input', value: 'binary-input' },
  { label: 'Binary Output', value: 'binary-output' },
  { label: 'Binary Value', value: 'binary-value' },
  { label: 'Multi-state Input', value: 'multi-state-input' },
  { label: 'Multi-state Output', value: 'multi-state-output' },
  { label: 'Multi-state Value', value: 'multi-state-value' }
]

const priorities = [
  { label: '1 (最高)', value: 1 },
  { label: '8 (手动)', value: 8 },
  { label: '16 (最低)', value: 16 },
  { label: 'NULL (释放)', value: null }
]
</script>

<style scoped>
.bacnet-point-config {
  padding: 8px 0;
}
</style>