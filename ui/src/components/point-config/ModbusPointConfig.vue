<template>
  <div class="modbus-point-config">
    <a-row :gutter="16">
      <a-col :span="12">
        <a-form-item field="registerType" label="寄存器类型">
          <a-select
            v-model="registerType"
            :options="registerTypes"
            @update:value="updateAddress"
          ></a-select>
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item field="registerIndex" label="寄存器索引">
          <a-input
            v-model.number="registerIndex"
            type="number"
            :min="getRegisterIndexMin()"
            :max="getRegisterIndexMax()"
            :error-message="registerIndexError"
            @input="validateRegisterIndex; updateAddress"
          ></a-input>
        </a-form-item>
      </a-col>
    </a-row>
    
    <a-row :gutter="16">
      <a-col :span="12">
        <a-form-item field="functionCode" label="功能码" id="functionCode">
          <a-input
            v-model.number="functionCode"
            type="number"
            min="1"
            max="255"
            :tooltip="{ title: '默认: 根据寄存器类型自动确定', placement: 'top' }"
          />
        </a-form-item>
      </a-col>
      <a-col :span="12">
        <a-form-item field="registerOffset" label="起始偏移量">
          <a-input
            v-model.number="registerOffset"
            type="number"
            min="0"
            max="9999"
            :error-message="registerOffsetError"
            @input="validateRegisterOffset"
          ></a-input>
        </a-form-item>
      </a-col>
    </a-row>
    
    <a-row :gutter="16">
      <a-col :span="12">
        <a-form-item field="address" label="Modbus 地址" id="address">
          <a-input
            v-model="form.address"
            :disabled="true"
            :tooltip="{ title: '自动计算的 PDU 0-based 地址', placement: 'top' }"
          ></a-input>
        </a-form-item>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

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

const registerType = ref(props.form.register_type || 'holding')
const registerIndex = ref(parseInt(props.form.address) || 0)
const registerOffset = ref(0)
const functionCode = ref(props.form.function_code || 3)
const registerIndexError = ref('')
const registerOffsetError = ref('')

const registerTypes = [
  { label: 'COIL (输出线圈)', value: 'coil' },
  { label: 'DISCRETE_INPUT (离散输入)', value: 'discrete' },
  { label: 'INPUT_REGISTER (输入寄存器)', value: 'input' },
  { label: 'HOLDING_REGISTER (保持寄存器)', value: 'holding' }
]

const getRegisterIndexMin = () => {
  const startAddress = props.deviceInfo?.config?.start_address || props.deviceInfo?.config?.address_base || 0
  return startAddress
}

const getRegisterIndexMax = () => {
  const startAddress = props.deviceInfo?.config?.start_address || props.deviceInfo?.config?.address_base || 0
  return startAddress + 65535
}

const validateRegisterIndex = () => {
  const idx = parseInt(registerIndex.value) || 0
  const min = getRegisterIndexMin()
  const max = getRegisterIndexMax()
  
  if (idx < min || idx > max) {
    registerIndexError.value = `寄存器索引必须在 ${min} 到 ${max} 之间`
  } else {
    registerIndexError.value = ''
  }
}

const validateRegisterOffset = () => {
  const offset = parseInt(registerOffset.value) || 0
  if (offset < 0 || offset > 9999) {
    registerOffsetError.value = '起始偏移量必须在 0 到 9999 之间'
  } else {
    registerOffsetError.value = ''
  }
}

const updateAddress = () => {
  const idx = parseInt(registerIndex.value) || 0
  const offset = parseInt(registerOffset.value) || 0
  let address = 0
  
  const startAddress = props.deviceInfo?.config?.start_address || props.deviceInfo?.config?.address_base || 0
  
  if (idx < startAddress) {
    registerIndexError.value = `地址不能小于基准地址 ${startAddress}`
    return
  }
  
  address = idx - startAddress + offset
  
  if (address < 0 || address > 65535) {
    registerIndexError.value = 'PDU地址必须在 0 到 65535 之间'
    return
  }
  
  registerIndexError.value = ''
  
  const functionCodeMap = {
    'coil': 1,
    'discrete': 2,
    'input': 4,
    'holding': 3
  }
  
  if (functionCodeMap[registerType.value]) {
    functionCode.value = functionCodeMap[registerType.value]
  }
  
  const updatedForm = {
    ...props.form,
    register_type: registerType.value,
    address: address.toString(),
    function_code: functionCode.value
  }
  
  emit('update:form', updatedForm)
}

watch(registerType, () => {
  updateAddress()
})

watch(registerOffset, () => {
  validateRegisterOffset()
  updateAddress()
})
</script>

<style scoped>
.modbus-point-config {
  padding: 8px 0;
}
</style>