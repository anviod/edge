<template>
  <div class="opcua-point-config">
    <a-row :gutter="16">
      <a-col :span="12">
        <a-form-item field="nodeId" label="NodeId">
          <a-input
            v-model="form.node_id"
            placeholder="例如: ns=2;s=Demo.Static.Scalar.Double"
            :tooltip="{ title: 'OPC-UA节点标识符', placement: 'top' }"
          ></a-input>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item field="namespaceIndex" label="命名空间索引">
          <a-input
            v-model.number="form.namespace_index"
            type="number"
            min="0"
            placeholder="默认: 0"
            :tooltip="{ title: 'NodeId的命名空间索引', placement: 'top' }"
          ></a-input>
        </a-form-item>
      </a-col>
    </a-row>
    
    <a-row :gutter="16">
      <a-col :span="12">
        <a-form-item field="dataType" label="数据类型">
          <a-select
            v-model="form.datatype"
            :options="dataTypes"
            placeholder="选择数据类型"
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

const dataTypes = [
  { label: 'Boolean', value: 'bool' },
  { label: 'Integer', value: 'int32' },
  { label: 'Float', value: 'float32' },
  { label: 'Double', value: 'float64' },
  { label: 'String', value: 'string' }
]
</script>

<style scoped>
.opcua-point-config {
  padding: 8px 0;
}
</style>