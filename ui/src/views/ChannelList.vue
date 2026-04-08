<template>
  <div class="channel-list-container">
    <div class="channel-header">
      <div class="header-title">
        <h2 class="title-text">采集通道</h2>
        <div class="title-subtitle">管理工业设备通信通道及协议配置</div>
      </div>
      <div class="header-actions">
        <a-space size="medium">
          <a-radio-group v-model="viewMode" type="button" size="small">
            <a-radio value="card">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="7" height="7"/>
                <rect x="14" y="3" width="7" height="7"/>
                <rect x="14" y="14" width="7" height="7"/>
                <rect x="3" y="14" width="7" height="7"/>
              </svg>
            </a-radio>
            <a-radio value="list">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="8" y1="6" x2="21" y2="6"/>
                <line x1="8" y1="12" x2="21" y2="12"/>
                <line x1="8" y1="18" x2="21" y2="18"/>
                <line x1="3" y1="6" x2="3.01" y2="6"/>
                <line x1="3" y1="12" x2="3.01" y2="12"/>
                <line x1="3" y1="18" x2="3.01" y2="18"/>
              </svg>
            </a-radio>
          </a-radio-group>
          <a-divider direction="vertical" />
          <a-space size="small">
            <a-button v-if="selectionMode && selectedChannels.length > 0" status="warning" size="small" @click="openBatchConfig">
              <template #icon>
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="3"/>
                  <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"/>
                </svg>
              </template>批量配置
            </a-button>
            <a-button :type="selectionMode ? 'secondary' : 'outline'" size="small" @click="toggleSelectionMode">
              <template #icon>
                <svg v-if="selectionMode" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
                <svg v-else viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                  <polyline points="9 11 12 14 22 4"/>
                </svg>
              </template>
              {{ selectionMode ? '取消选择' : '批量操作' }}
            </a-button>
            <a-button type="outline" size="small" :loading="loading" @click="fetchChannels">
              <template #icon>
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="23 4 23 10 17 10"/>
                  <polyline points="1 20 1 14 7 14"/>
                  <path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
                </svg>
              </template>刷新
            </a-button>
            <a-button type="primary" size="small" @click="openAddDialog">
              <template #icon>
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
              </template>添加通道
            </a-button>
          </a-space>
        </a-space>
      </div>
    </div>

    <a-spin :loading="loading" tip="数据同步中..." style="width: 100%">
      <div v-if="channels.length > 0">
        <a-row v-if="viewMode === 'card'" :gutter="[16, 16]">
          <a-col v-for="item in channels" :key="item.id" :xs="24" :sm="12" :md="12" :lg="8">
            <a-card 
              class="minimal-line-card" 
              :class="{ 'is-selected': isSelected(item.id) }" 
              hoverable 
              @click="handleCardClick(item)"
            >
              <template #title>
                <div class="card-title-content">
                  <span class="protocol-tag">{{ item.protocol }}</span>
                  <span class="name-text text-truncate">{{ item.name }}</span>
                </div>
              </template>
              <template #extra>
                <a-tag :color="item.enableColor" size="small" bordered>{{ item.enableText }}</a-tag>
              </template>

              <div class="card-info-body">
                <div class="info-item">
                  <span class="label">通道 ID</span>
                  <span class="value">{{ item.id }}</span>
                </div>
                <div class="info-item">
                  <span class="label">关联设备</span>
                  <span class="value-highlight">{{ item.deviceCount }} <small>台</small></span>
                </div>
                <div class="info-item">
                  <span class="label">运行状态</span>
                  <a-badge :status="item.runtimeArcoStatus" :text="item.runtimeText" />
                </div>
              </div>

              <template #actions>
                <a-tooltip content="监控指标">
                  <a-button type="text" size="small" @click.stop="openMetricsDialog(item)">
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="18" y1="20" x2="18" y2="10"/>
                      <line x1="12" y1="20" x2="12" y2="4"/>
                      <line x1="6" y1="20" x2="6" y2="14"/>
                    </svg>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="编辑">
                  <a-button type="text" size="small" @click.stop="openEditDialog(item)">
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                      <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                  </a-button>
                </a-tooltip>
                <a-tooltip v-if="item.protocol === 'bacnet-ip'" content="扫描设备">
                  <a-button type="text" size="small" @click.stop="scanChannel(item)">
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M12 2a10 10 0 1010 10 10 10 0 00-10-10z"/>
                      <path d="M12 6v6l4 2"/>
                    </svg>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="删除">
                  <a-button type="text" size="small" status="danger" @click.stop="deleteChannel(item)">
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="3 6 5 6 21 6"/>
                      <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                    </svg>
                  </a-button>
                </a-tooltip>
              </template>
            </a-card>
          </a-col>
        </a-row>

        <a-table 
          v-else 
          :columns="tableColumns" 
          :data="channels" 
          :row-selection="selectionMode ? rowSelection : undefined"
          row-key="id"
          size="small"
          :bordered="{ cell: true }"
          :pagination="false"
        >
          <template #name="{ record }">
            <a-link @click="goToDevices(record)" icon>{{ record.name }}</a-link>
          </template>

          <template #enable="{ record }">
            <a-tag :color="record.enableColor" size="small" bordered>{{ record.enableText }}</a-tag>
          </template>

          <template #runtime="{ record }">
            <a-badge :status="record.runtimeArcoStatus" :text="record.runtimeText" />
          </template>

          <template #deviceCount="{ record }">
            <span style="font-weight: 500">{{ record.deviceCount }}</span>
          </template>

          <template #actions="{ record }">
            <a-space>
              <a-tooltip content="监控">
                <a-button type="text" size="mini" @click="openMetricsDialog(record)">
                  <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="20" x2="18" y2="10"/>
                    <line x1="12" y1="20" x2="12" y2="4"/>
                    <line x1="6" y1="20" x2="6" y2="14"/>
                  </svg>
                </a-button>
              </a-tooltip>
              <a-tooltip content="编辑">
                <a-button type="text" size="mini" @click="openEditDialog(record)">
                  <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </a-button>
              </a-tooltip>
              <a-tooltip v-if="record.protocol === 'bacnet-ip'" content="扫描">
                <a-button type="text" size="mini" @click="scanChannel(record)">
                  <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 2a10 10 0 1010 10 10 10 0 00-10-10z"/>
                    <path d="M12 6v6l4 2"/>
                  </svg>
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除">
                <a-button type="text" size="mini" status="danger" @click="deleteChannel(record)">
                  <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                  </svg>
                </a-button>
              </a-tooltip>
            </a-space>
          </template>
        </a-table>
      </div>
      <a-empty v-else class="empty-placeholder" />
    </a-spin>

    <!-- Add/Edit Dialog -->
    <a-modal 
      v-model:visible="dialog.show" 
      :title="dialog.isEdit ? '编辑通道' : '添加通道'"
      :width="900"
      @ok="saveChannel"
    >
      <a-form :model="dialog.form" layout="horizontal" :label-col-props="{ span: 6 }" :wrapper-col-props="{ span: 18 }">
        <a-form-item field="id" label="ID" required>
          <a-input v-model="dialog.form.id" :disabled="dialog.isEdit" placeholder="请输入ID">
            <template #append v-if="!dialog.isEdit">
              <a-button @click="generateId">
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="23 4 23 10 17 10"/>
                  <polyline points="1 20 1 14 7 14"/>
                  <path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
                </svg>
              </a-button>
            </template>
          </a-input>
        </a-form-item>
        
        <a-form-item field="name" label="名称" required>
          <a-input v-model="dialog.form.name" placeholder="给通道起一个易于识别的名称" />
        </a-form-item>

        <a-form-item field="protocol" label="协议" required>
          <a-select v-model="dialog.form.protocol" :options="protocols" placeholder="请选择协议" />
        </a-form-item>

        <a-form-item field="enable" label="启用">
          <a-switch v-model="dialog.form.enable" />
        </a-form-item>

        <!-- Protocol specific config -->
        <a-divider orientation="left">协议配置</a-divider>

        <!-- Modbus TCP & Modbus RTU Over TCP Config -->
        <div v-if="dialog.form.protocol === 'modbus-tcp' || dialog.form.protocol === 'modbus-rtu-over-tcp'" class="config-section">
          <a-form-item field="config.url" :label="dialog.form.protocol === 'modbus-rtu-over-tcp' ? 'URL (tcp+rtu://ip:port)' : 'URL (tcp://ip:port)'">
            <a-input v-model="dialog.form.config.url" />
          </a-form-item>
          <a-form-item field="config.timeout" label="超时时间 (ms)">
            <a-input-number v-model="dialog.form.config.timeout" :min="100" :max="60000" placeholder="2000" />
          </a-form-item>

          <a-divider class="my-4"></a-divider>
          <div class="section-header">
            <span class="section-title">高级配置</span>
          </div>
          <a-row :gutter="12">
            <a-col :span="12">
              <a-form-item field="config.max_retries" label="最大重试次数">
                <a-input-number v-model="dialog.form.config.max_retries" :min="0" :max="10" placeholder="3" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="config.retry_interval" label="重试间隔 (ms)">
                <a-input-number v-model="dialog.form.config.retry_interval" :min="10" :max="1000" placeholder="100" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="config.instruction_interval" label="指令间隔 (ms)">
                <a-input-number v-model="dialog.form.config.instruction_interval" :min="1" :max="100" placeholder="10" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="config.start_address" label="起始地址">
                <a-select v-model="dialog.form.config.start_address" placeholder="默认 1 (40001)">
                  <a-option :value="0">0 (40000)</a-option>
                  <a-option :value="1">1 (40001)</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="config.byte_order_4" label="4字节字节序">
                <a-select v-model="dialog.form.config.byte_order_4" placeholder="默认 ABCD (Big Endian)">
                  <a-option value="ABCD">ABCD</a-option>
                  <a-option value="CDAB">CDAB</a-option>
                  <a-option value="BADC">BADC</a-option>
                  <a-option value="DCBA">DCBA</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="config.enableSmartProbe" label="启用智能地址探测">
                <a-switch v-model="dialog.form.config.enableSmartProbe" />
              </a-form-item>
            </a-col>
            <a-col :span="12" v-if="dialog.form.config.enableSmartProbe">
              <a-form-item field="config.probeMaxDepth" label="探测深度">
                <a-input-number v-model="dialog.form.config.probeMaxDepth" :min="1" :max="10" placeholder="6" />
              </a-form-item>
            </a-col>
            <a-col :span="12" v-if="dialog.form.config.enableSmartProbe">
              <a-form-item field="config.probeTimeout" label="探测超时 (ms)">
                <a-input-number v-model="dialog.form.config.probeTimeout" :min="100" :max="10000" placeholder="3000" />
              </a-form-item>
            </a-col>
            <a-col :span="12" v-if="dialog.form.config.enableSmartProbe">
              <a-form-item field="config.probeMaxConsecutive" label="最大连续失败">
                <a-input-number v-model="dialog.form.config.probeMaxConsecutive" :min="1" :max="100" placeholder="20" />
              </a-form-item>
            </a-col>
            <a-col :span="12" v-if="dialog.form.config.enableSmartProbe">
              <a-form-item field="config.probeEnableMTU" label="启用MTU探测">
                <a-switch v-model="dialog.form.config.probeEnableMTU" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <!-- DLT645 Config -->
        <div v-if="dialog.form.protocol === 'dlt645'" class="config-section">
          <a-form-item field="config.connectionType" label="连接方式">
            <a-select v-model="dialog.form.config.connectionType" placeholder="请选择连接方式">
              <a-option value="serial">串口 (Serial)</a-option>
              <a-option value="tcp">网络 (TCP)</a-option>
            </a-select>
          </a-form-item>
          <div v-if="dialog.form.config.connectionType === 'tcp'">
            <a-form-item field="config.ip" label="设备 IP 地址">
              <a-input v-model="dialog.form.config.ip" placeholder="192.168.1.100" />
            </a-form-item>
            <a-form-item field="config.port" label="端口">
              <a-input-number v-model="dialog.form.config.port" :min="1" :max="65535" placeholder="8001" />
            </a-form-item>
            <a-form-item field="config.timeout" label="超时时间 (ms)">
              <a-input-number v-model="dialog.form.config.timeout" :min="100" :max="60000" placeholder="2000" />
            </a-form-item>
          </div>
        </div>

        <!-- Modbus RTU & DLT645 Serial Config -->
        <div v-if="dialog.form.protocol === 'modbus-rtu' || (dialog.form.protocol === 'dlt645' && dialog.form.config.connectionType === 'serial')" class="config-section">
          <a-form-item field="config.port" label="串口设备">
            <a-input v-model="dialog.form.config.port" placeholder="/dev/ttyS1" />
          </a-form-item>
          <a-row :gutter="12">
            <a-col :span="6">
              <a-form-item field="config.baudRate" label="波特率">
                <a-select v-model="dialog.form.config.baudRate" placeholder="9600">
                  <a-option :value="1200">1200</a-option>
                  <a-option :value="2400">2400</a-option>
                  <a-option :value="4800">4800</a-option>
                  <a-option :value="9600">9600</a-option>
                  <a-option :value="19200">19200</a-option>
                  <a-option :value="38400">38400</a-option>
                  <a-option :value="57600">57600</a-option>
                  <a-option :value="115200">115200</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item field="config.dataBits" label="数据位">
                <a-select v-model="dialog.form.config.dataBits" placeholder="8">
                  <a-option :value="5">5</a-option>
                  <a-option :value="6">6</a-option>
                  <a-option :value="7">7</a-option>
                  <a-option :value="8">8</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item field="config.stopBits" label="停止位">
                <a-select v-model="dialog.form.config.stopBits" placeholder="1">
                  <a-option :value="1">1</a-option>
                  <a-option :value="2">2</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item field="config.parity" label="校验位">
                <a-select v-model="dialog.form.config.parity" placeholder="无校验">
                  <a-option value="N">无校验 (None)</a-option>
                  <a-option value="E">偶校验 (Even)</a-option>
                  <a-option value="O">奇校验 (Odd)</a-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item field="config.timeout" label="超时时间 (ms)">
            <a-input-number v-model="dialog.form.config.timeout" :min="100" :max="60000" placeholder="2000" />
          </a-form-item>

          <!-- Modbus RTU Advanced Config -->
          <div v-if="dialog.form.protocol === 'modbus-rtu'">
            <a-divider class="my-4"></a-divider>
            <div class="section-header">
              <span class="section-title">高级配置</span>
            </div>
            <a-row :gutter="12">
              <a-col :span="12">
                <a-form-item field="config.max_retries" label="最大重试次数">
                  <a-input-number v-model="dialog.form.config.max_retries" :min="0" :max="10" placeholder="3" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item field="config.retry_interval" label="重试间隔 (ms)">
                  <a-input-number v-model="dialog.form.config.retry_interval" :min="10" :max="1000" placeholder="100" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item field="config.instruction_interval" label="指令间隔 (ms)">
                  <a-input-number v-model="dialog.form.config.instruction_interval" :min="1" :max="100" placeholder="10" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item field="config.start_address" label="起始地址">
                  <a-select v-model="dialog.form.config.start_address" placeholder="默认 1 (40001)">
                    <a-option :value="0">0 (40000)</a-option>
                    <a-option :value="1">1 (40001)</a-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item field="config.byte_order_4" label="4字节字节序">
                  <a-select v-model="dialog.form.config.byte_order_4" placeholder="默认 ABCD (Big Endian)">
                    <a-option value="ABCD">ABCD</a-option>
                    <a-option value="CDAB">CDAB</a-option>
                    <a-option value="BADC">BADC</a-option>
                    <a-option value="DCBA">DCBA</a-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
          </div>
        </div>

        <!-- BACnet IP Config -->
        <div v-if="dialog.form.protocol === 'bacnet-ip'" class="config-section">
          <a-form-item field="config.ip" label="IP地址">
            <a-input v-model="dialog.form.config.ip" placeholder="0.0.0.0 (默认)" />
          </a-form-item>
          <a-form-item field="config.port" label="端口">
            <a-input-number v-model="dialog.form.config.port" :min="1" :max="65535" placeholder="47808 (默认)" />
          </a-form-item>
          <a-divider class="my-4"></a-divider>
          <div class="section-header">
            <span class="section-title">加密参数 (可选)</span>
          </div>
          <a-form-item field="config.key" label="密钥">
            <a-input v-model="dialog.form.config.key" type="password" />
          </a-form-item>
          <a-form-item field="config.cert" label="证书路径">
            <a-input v-model="dialog.form.config.cert" />
          </a-form-item>
          <a-form-item field="config.ca" label="CA证书路径">
            <a-input v-model="dialog.form.config.ca" />
          </a-form-item>
        </div>

        <!-- OPC UA Config -->
        <div v-if="dialog.form.protocol === 'opc-ua'" class="config-section">
          <a-form-item field="config.url" label="Endpoint URL">
            <a-input v-model="dialog.form.config.url" placeholder="opc.tcp://localhost:4840" />
          </a-form-item>
        </div>

        <!-- S7 Config -->
        <div v-if="dialog.form.protocol === 's7'" class="config-section">
          <a-form-item field="config.ip" label="PLC IP 地址" required>
            <a-input v-model="dialog.form.config.ip" />
          </a-form-item>
          <a-form-item field="config.port" label="PLC 端口">
            <a-input-number v-model="dialog.form.config.port" :min="1" :max="65535" placeholder="102 (默认)" />
          </a-form-item>
          <a-row :gutter="12">
            <a-col :span="12">
              <a-form-item field="config.rack" label="CPU 机架号 (Rack)">
                <a-input-number v-model="dialog.form.config.rack" :min="0" :max="10" placeholder="0 (默认)" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="config.slot" label="CPU 槽号 (Slot)">
                <a-input-number v-model="dialog.form.config.slot" :min="0" :max="10" placeholder="1 (默认)" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="config.plcType" label="PLC 型号">
                <a-select v-model="dialog.form.config.plcType" placeholder="请选择 PLC 型号">
                  <a-option value="S7-200Smart">S7-200Smart</a-option>
                  <a-option value="S7-1200">S7-1200</a-option>
                  <a-option value="S7-1500">S7-1500</a-option>
                  <a-option value="S7-300">S7-300</a-option>
                  <a-option value="S7-400">S7-400</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item field="config.startup" label="启动方式">
                <a-select v-model="dialog.form.config.startup" placeholder="请选择启动方式">
                  <a-option value="cold">冷启动</a-option>
                  <a-option value="warm">热启动</a-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <!-- Help button -->
        <div class="mt-4 d-flex justify-end">
          <a-button type="text" status="info" @click="showHelp = true" v-if="!dialog.isEdit">
            <template #icon>
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <path d="M9.09 9a3 3 0 015.83 1c0 2-3 3-3 3"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
            </template>
            查看帮助说明
          </a-button>
        </div>
      </a-form>
    </a-modal>

    <!-- Smart Probe Help Modal -->
    <a-modal
      v-model:visible="smartProbeHelpDialog.show"
      title="智能地址探测帮助"
      :width="1000"
      :footer="false"
      unmount-on-close
    >
      <div class="smart-probe-help">
        <div class="help-section">
          <h3>什么是智能地址探测？</h3>
          <p>智能地址探测是一种自动扫描和识别Modbus设备有效寄存器地址的功能，它能够：</p>
          <ul>
            <li>自动扫描设备的有效寄存器地址范围</li>
            <li>检测设备的MTU（最大传输单元）大小</li>
            <li>优化寄存器分组策略，提高读取效率</li>
            <li>减少手动配置错误，提高系统稳定性</li>
          </ul>
        </div>
        <div class="help-section">
          <h3>工作原理</h3>
          <div class="principle-section">
            <h4>1. 分层扫描策略</h4>
            <p>系统采用分层扫描算法，从粗到细逐步定位有效地址：</p>
            <ul>
              <li><strong>第一层：</strong>按1000地址为间隔进行快速扫描</li>
              <li><strong>第二层：</strong>对包含有效地址的区间按100地址间隔扫描</li>
              <li><strong>第三层：</strong>对包含有效地址的区间按10地址间隔扫描</li>
              <li><strong>第四层：</strong>对包含有效地址的区间进行逐地址扫描</li>
            </ul>
            <h4>2. MTU检测</h4>
            <p>系统会自动检测设备的最大传输单元大小，以确定单次可读取的最大寄存器数量，从而优化读取效率。</p>
            <h4>3. 分组优化</h4>
            <p>扫描完成后，系统会对连续的有效寄存器地址进行分组，生成最优的读取指令序列，减少通信次数。</p>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <a-button type="primary" @click="smartProbeHelpDialog.show = false">关闭</a-button>
      </div>
    </a-modal>

    <!-- Channel Metrics Modal -->
    <a-modal
      v-model:visible="metricsDialog.show"
      :title="`通道监控指标 - ${metricsDialog.channel?.name || ''}`"
      :width="800"
      :footer="false"
      unmount-on-close
    >
      <div v-if="metricsDialog.loading" class="metrics-loading">
        <a-spin tip="加载监控指标中..." size="large" />
      </div>
      <div v-else-if="metricsDialog.error" class="metrics-error">
        <a-alert type="error" :title="metricsDialog.error" show-icon />
      </div>
      <div v-else-if="metricsDialog.metrics" class="metrics-content">
        <!-- 质量评分 -->
        <div class="metrics-header">
          <div class="quality-score-section">
            <div class="quality-score-circle" :class="getQualityClass(qualityScore)">
              <div class="quality-score-value">{{ qualityScore }}</div>
              <div class="quality-score-label">质量评分</div>
              <div class="quality-score-level">{{ getQualityLabel(qualityScore) }}</div>
            </div>
            <div class="quality-info">
              <div class="info-item">
                <a-badge :status="getQualityStatus(qualityScore)" :text="`通道状态: ${getQualityLabel(qualityScore)}`" />
              </div>
              <div v-if="metricsDialog.metrics.reconnectCount > 0" class="info-item">
                <a-tag status="warning">重连 {{ metricsDialog.metrics.reconnectCount }} 次</a-tag>
              </div>
              <div class="info-item">
                <span class="info-label">连接时长：</span>
                <span class="info-value">{{ connectionDuration }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">网络连接：</span>
                <span class="info-value">
                  {{ getNetworkConnectionText() }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 核心指标 -->
        <div class="metrics-core">
          <div class="metrics-grid">
            <div class="metric-card">
              <div class="metric-label">成功率</div>
              <div class="metric-value" :class="getSuccessRateClass(metricsDialog.metrics?.successRate)">
                {{ formatPercent(metricsDialog.metrics?.successRate) }}
              </div>
            </div>
            <div class="metric-card">
              <div class="metric-label">平均 RTT</div>
              <div class="metric-value">
                {{ formatDuration(metricsDialog.metrics?.avgRtt) }}
              </div>
            </div>
            <div class="metric-card">
              <div class="metric-label">丢包率</div>
              <div class="metric-value" :class="getPacketLossClass(metricsDialog.metrics?.packetLoss)">
                {{ formatPercent(metricsDialog.metrics?.packetLoss) }}
              </div>
            </div>
          </div>
        </div>

        <!-- 详细指标 -->
        <div class="metrics-details">
          <a-collapse v-model="metricsDialog.expandedDetails" class="metrics-collapse">
            <a-collapse-item name="details" title="详细指标">
              <a-descriptions :column="2" size="small" bordered>
                <a-descriptions-item label="CRC错误率">
                  {{ formatPercent(metricsDialog.metrics?.crcErrorRate) }}
                </a-descriptions-item>
                <a-descriptions-item label="重试率">
                  {{ formatPercent(metricsDialog.metrics?.retryRate) }}
                </a-descriptions-item>
                <a-descriptions-item label="总请求数">
                  {{ metricsDialog.metrics?.totalRequests || 0 }}
                </a-descriptions-item>
                <a-descriptions-item label="成功请求数">
                  {{ metricsDialog.metrics?.successRequests || 0 }}
                </a-descriptions-item>
                <a-descriptions-item label="失败请求数">
                  {{ metricsDialog.metrics?.failedRequests || 0 }}
                </a-descriptions-item>
                <a-descriptions-item label="重连次数">
                  {{ metricsDialog.metrics?.reconnectCount || 0 }}
                </a-descriptions-item>
              </a-descriptions>
            </a-collapse-item>
          </a-collapse>
        </div>
      </div>
      <div v-else class="metrics-empty">
        <a-empty description="暂无监控指标数据" />
      </div>
      <div class="modal-footer">
        <a-button type="primary" @click="metricsDialog.show = false">关闭</a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed, h } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import request from '@/utils/request'

// 恢复使用 SVG 图标以避免导入问题

const router = useRouter()
const loading = ref(false)
const viewMode = ref('card')
const selectionMode = ref(false)
const selectedChannels = ref([])
const channels = ref([])

const dialog = reactive({
  show: false,
  isEdit: false,
  form: {
    id: '',
    name: '',
    protocol: 'modbus-tcp',
    enable: true,
    config: {},
    devices: []
  }
})

const smartProbeHelpDialog = reactive({
  show: false
})

const metricsDialog = reactive({
  show: false,
  loading: false,
  error: null,
  channel: null,
  metrics: null,
  expandedDetails: ['details']
})

const protocols = [
  { label: 'Modbus TCP', value: 'modbus-tcp' },
  { label: 'Modbus RTU Over TCP', value: 'modbus-rtu-over-tcp' },
  { label: 'Modbus RTU', value: 'modbus-rtu' },
  { label: 'BACnet IP', value: 'bacnet-ip' },
  { label: 'OPC UA', value: 'opc-ua' },
  { label: 'S7', value: 's7' },
  { label: 'DLT645', value: 'dlt645' }
]

const tableColumns = [
  { title: '通道名称', slotName: 'name', width: 200 },
  { title: '协议类型', dataIndex: 'protocol', width: 140 },
  { title: '启用状态', slotName: 'enable', width: 100 },
  { title: '运行状态', slotName: 'runtime', width: 120 },
  { title: '关联设备', slotName: 'deviceCount', width: 100, align: 'center' },
  { title: '操作', slotName: 'actions', width: 220, fixed: 'right' },
]

const rowSelection = {
  selectedRowKeys: selectedChannels,
  onChange: (keys) => {
    selectedChannels.value = keys
  }
}

const isSelected = (id) => selectedChannels.value.includes(id)

const toggleSelectionMode = () => {
  selectionMode.value = !selectionMode.value
  selectedChannels.value = []
}

const toggleChannelSelection = (id) => {
  const idx = selectedChannels.value.indexOf(id)
  if (idx === -1) {
    selectedChannels.value.push(id)
  } else {
    selectedChannels.value.splice(idx, 1)
  }
}

const handleCardClick = (channel) => {
  if (selectionMode.value) {
    toggleChannelSelection(channel.id)
  } else {
    goToDevices(channel)
  }
}

const goToDevices = (channel) => {
  router.push(`/channels/${channel.id}/devices`)
}

const openAddDialog = () => {
  dialog.isEdit = false
  dialog.form = {
    id: '',
    name: '',
    protocol: 'modbus-tcp',
    enable: true,
    config: {},
    devices: []
  }
  dialog.show = true
}

const openEditDialog = (channel) => {
  dialog.isEdit = true
  dialog.form = JSON.parse(JSON.stringify(channel))
  dialog.show = true
}

const generateId = () => {
  if (dialog.isEdit) return
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < 16; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  dialog.form.id = result
}

const saveChannel = async () => {
  try {
    if (!dialog.form.id || !dialog.form.name) {
      Message.error('请填写完整信息')
      return
    }
    
    if (dialog.isEdit) {
      await request({
        url: `/api/channels/${dialog.form.id}`,
        method: 'put',
        data: dialog.form
      })
      Message.success('通道更新成功')
    } else {
      await request({
        url: '/api/channels',
        method: 'post',
        data: dialog.form
      })
      Message.success('通道添加成功')
    }
    
    dialog.show = false
    fetchChannels()
  } catch (e) {
    Message.error('操作失败: ' + e.message)
  }
}

const deleteChannel = async (channel) => {
  if (!confirm(`确定要删除通道 "${channel.name}" 吗？`)) return
  
  try {
    await request({
      url: `/api/channels/${channel.id}`,
      method: 'delete'
    })
    Message.success('通道删除成功')
    fetchChannels()
  } catch (e) {
    Message.error('删除失败: ' + e.message)
  }
}

const openMetricsDialog = async (channel) => {
  metricsDialog.channel = channel
  metricsDialog.error = null
  
  // 检查通道是否已有缓存的指标数据
  if (channel.metrics && Object.keys(channel.metrics).length > 0) {
    // 直接使用缓存数据，不显示加载状态
    metricsDialog.metrics = channel.metrics
    metricsDialog.loading = false
    metricsDialog.show = true
  } else {
    // 需要加载指标数据
    metricsDialog.loading = true
    metricsDialog.show = true
    
    try {
      // 设置2秒超时
      const timeoutPromise = new Promise((_, reject) => 
        setTimeout(() => reject(new Error('加载超时')), 2000)
      )
      
      const metricsResPromise = request({
        url: `/api/channels/${channel.id}/metrics`,
        method: 'get'
      })
      
      const metricsRes = await Promise.race([metricsResPromise, timeoutPromise])
      console.log('Metrics response:', metricsRes)
      metricsDialog.metrics = metricsRes
    } catch (error) {
      metricsDialog.error = `获取监控指标失败: ${error.message}`
      console.error('Failed to get channel metrics:', error)
    } finally {
      metricsDialog.loading = false
    }
  }
}

const scanChannel = (channel) => {
  // 实现设备扫描功能
  console.log('Scan devices for channel:', channel)
}

const openBatchConfig = () => {
  // 实现批量配置功能
  console.log('Open batch config for channels:', selectedChannels.value)
}

const getRuntimeColor = (state) => {
  const colorMap = {
    'running': 'success',
    'error': 'danger',
    'offline': 'gray'
  }
  return colorMap[state] || 'gray'
}

const getRuntimeText = (state) => {
  const textMap = {
    'running': '运行中',
    'error': '错误',
    'offline': '离线'
  }
  return textMap[state] || '未知'
}

// 监控指标相关计算属性和函数
const qualityScore = computed(() => {
  if (!metricsDialog.metrics) return 0
  
  let score = 100
  const m = metricsDialog.metrics
  
  if (m.successRate !== undefined)
    score -= (1 - m.successRate) * 40
  
  if (m.crcErrorRate !== undefined)
    score -= m.crcErrorRate * 20
  
  if (m.retryRate !== undefined)
    score -= m.retryRate * 20
  
  if (m.avgRtt > 100)
    score -= Math.min(10, (m.avgRtt - 100) / 50)
  
  return Math.max(0, Math.round(score))
})

const connectionDuration = computed(() => {
  if (!metricsDialog.metrics) return '暂无连接信息'
  const seconds = metricsDialog.metrics.connectionSeconds || 0
  if (seconds === 0) return '已连接 0s（刚建立连接）'
  if (seconds < 60) return `已连接 ${seconds}s`
  if (seconds < 3600) return `已连接 ${Math.floor(seconds / 60)}m`
  return `已连接 ${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
})

// 解析网络地址信息
const networkInfo = computed(() => {
  if (!metricsDialog.metrics) {
    return { localIp: '-', localPort: '-', remoteIp: '-', remotePort: '-' }
  }
  
  // 优先使用分开的字段
  let localIp = metricsDialog.metrics.localIp || metricsDialog.metrics.local_ip
  let localPort = metricsDialog.metrics.localPort || metricsDialog.metrics.local_port
  let remoteIp = metricsDialog.metrics.remoteIp || metricsDialog.metrics.remote_ip
  let remotePort = metricsDialog.metrics.remotePort || metricsDialog.metrics.remote_port
  
  // 辅助函数：解析地址字符串（处理IP:Port格式）
  const parseAddressString = (addrStr) => {
    if (!addrStr) return { ip: '-', port: '-' }
    
    // 去掉协议前缀
    let addr = addrStr
    if (addr.includes('://')) {
      addr = addr.split('://')[1] || addr
    }
    
    // 提取IP和端口
    // 处理IPv6 [::1]:8080 这种格式
    if (addr.startsWith('[')) {
      const bracketIdx = addr.indexOf(']')
      if (bracketIdx > 0) {
        const ip = addr.substring(1, bracketIdx)
        const rest = addr.substring(bracketIdx + 1)
        if (rest.startsWith(':')) {
          return { ip, port: rest.substring(1).split('/')[0] }
        }
        return { ip, port: '-' }
      }
    }
    
    // 处理普通IPv4格式 IP:Port 或 包含路径的格式 IP:Port/Path
    const colonIdx = addr.lastIndexOf(':')
    if (colonIdx > 0) {
      const ip = addr.substring(0, colonIdx)
      let port = addr.substring(colonIdx + 1)
      
      // 处理包含路径的情况
      const slashIdx = port.indexOf('/')
      if (slashIdx > 0) {
        port = port.substring(0, slashIdx)
      }
      
      return { ip, port }
    }
    
    // 如果找不到冒号，整个字符串作为IP
    return { ip: addr, port: '-' }
  }
  
  // 如果没有分开的字段，尝试解析 localAddr 和 remoteAddr
  if (!localIp && metricsDialog.metrics.localAddr) {
    const parsed = parseAddressString(metricsDialog.metrics.localAddr)
    localIp = parsed.ip
    localPort = parsed.port
  }
  
  if (!remoteIp && metricsDialog.metrics.remoteAddr) {
    const parsed = parseAddressString(metricsDialog.metrics.remoteAddr)
    remoteIp = parsed.ip
    remotePort = parsed.port
  }
  
  return {
    localIp: localIp || '-',
    localPort: localPort || '-',
    remoteIp: remoteIp || '-',
    remotePort: remotePort || '-'
  }
})

const getQualityLabel = (score) => {
  if (score >= 90) return '优秀'
  if (score >= 75) return '良好'
  if (score >= 60) return '一般'
  return '较差'
}

const getQualityClass = (score) => {
  if (score >= 90) return 'quality-excellent'
  if (score >= 75) return 'quality-good'
  if (score >= 60) return 'quality-warning'
  return 'quality-poor'
}

const getNetworkConnectionText = () => {
  if (!metricsDialog.metrics) {
    return '暂无网络连接信息'
  }
  
  // 获取网络信息，确保没有undefined
  const info = networkInfo
  const localIp = info.localIp || '-'
  const localPort = info.localPort || '-'
  const remoteIp = info.remoteIp || '-'
  const remotePort = info.remotePort || '-'
  
  const local = `${localIp}:${localPort}`
  
  // 检查remoteAddr是否是IP:Port格式
  const remoteAddr = metricsDialog.metrics.remoteAddr
  if (remoteAddr && remoteAddr.includes(':')) {
    const parts = remoteAddr.split(':')
    if (parts.length >= 2 && !isNaN(parts[1])) {
      // 是IP:Port格式
      return `本地 ${local} → 目标 ${remoteIp}:${remotePort}`
    }
  }
  
  // 不是标准IP:Port格式，直接显示remoteAddr作为描述
  if (remoteAddr && remoteAddr !== '') {
    return `本地 ${local} → ${remoteAddr}`
  }
  
  return `本地 ${local} → 目标 -:-`
}

const getQualityStatus = (score) => {
  if (score >= 90) return 'success'
  if (score >= 75) return 'info'
  if (score >= 60) return 'warning'
  return 'danger'
}

const getSuccessRateClass = (rate) => {
  if (rate >= 0.99) return 'metric-success'
  if (rate >= 0.95) return 'metric-warning'
  return 'metric-danger'
}

const getPacketLossClass = (rate) => {
  if (rate < 0.01) return 'metric-success'
  if (rate < 0.05) return 'metric-warning'
  return 'metric-danger'
}

const formatPercent = (val) => {
  if (val === undefined || val === null) return '-'
  return (val * 100).toFixed(1) + '%'
}

const formatDuration = (ms) => {
  if (ms === undefined || ms === null) return '-'
  if (ms < 1) return '<1ms'
  if (ms < 1000) return ms.toFixed(2) + 'ms'
  return (ms / 1000).toFixed(2) + 's'
}

const fetchChannels = async () => {
  loading.value = true
  try {
    const res = await request({ url: '/api/channels', method: 'get' })
    const rawData = Array.isArray(res) ? res : (res.data || [])
    
    // 第一步：快速展示通道列表（不等待指标）
    const channelsWithoutMetrics = rawData.map((item) => {
      const count = Array.isArray(item.devices) ? item.devices.length : 0
      const enableText = item.enable ? '已启用' : '已禁用'
      const enableColor = item.enable ? 'green' : 'gray'
      
      // 初始状态：如果enabled则默认在线，否则离线
      let runtimeText = '离线'
      let runtimeArcoStatus = 'normal'
      
      if (item.enable) {
        runtimeText = '运行中'
        runtimeArcoStatus = 'success'
      }
      
      return {
        ...item,
        deviceCount: count,
        enableText,
        enableColor,
        runtimeText,
        runtimeArcoStatus,
        metrics: null // 先不加载指标
      }
    })
    
    channels.value = channelsWithoutMetrics
    loading.value = false // 立即关闭loading
    
    // 第二步：异步加载指标数据
    Promise.all(
      rawData.map(async (item) => {
        try {
          const metricsRes = await request({
            url: `/api/channels/${item.id}/metrics`,
            method: 'get'
          })
          return { channelId: item.id, metrics: metricsRes }
        } catch (metricsError) {
          console.warn(`获取通道 ${item.id} 指标失败:`, metricsError)
          return { channelId: item.id, metrics: null }
        }
      })
    ).then((metricsResults) => {
      // 更新通道指标
      metricsResults.forEach((result) => {
        const channelIndex = channels.value.findIndex(ch => ch.id === result.channelId)
        if (channelIndex >= 0) {
          const metrics = result.metrics
          
          // 基于监控指标更新状态
          if (metrics && metrics.qualityScore !== undefined && metrics.qualityScore !== null) {
            const qualityScore = metrics.qualityScore
            let runtimeText = '运行中'
            let runtimeArcoStatus = 'success'
            
            if (qualityScore >= 90) {
              runtimeText = '运行中 (优秀)'
              runtimeArcoStatus = 'success'
            } else if (qualityScore >= 75) {
              runtimeText = '运行中 (良好)'
              runtimeArcoStatus = 'success'
            } else if (qualityScore >= 60) {
              runtimeText = '运行中 (一般)'
              runtimeArcoStatus = 'warning'
            } else if (qualityScore > 0) {
              runtimeText = '运行中 (较差)'
              runtimeArcoStatus = 'danger'
            } else {
              runtimeText = '离线'
              runtimeArcoStatus = 'normal'
            }
            
            channels.value[channelIndex] = {
              ...channels.value[channelIndex],
              metrics,
              runtimeText,
              runtimeArcoStatus
            }
          }
        }
      })
    }).catch((err) => {
      console.error('批量加载指标失败:', err)
    })
  } catch (e) {
    Message.error('加载通道列表失败: ' + e.message)
    loading.value = false
  }
}

onMounted(() => {
  fetchChannels()
})
</script>

<style scoped>
.channel-list-container {
  padding: 24px;
  min-height: calc(100vh - 56px);
  background: #f1f5f9;
}

.dark-theme .channel-list-container {
  background: #070f1f !important;
}

.dark-theme .channel-header {
  background: #0f172a !important;
  border-color: #334155 !important;
}

.dark-theme .title-text,
.dark-theme .title-subtitle,
.dark-theme .header-actions,
.dark-theme .protocol-tag,
.dark-theme .name-text,
.dark-theme .info-item .label,
.dark-theme .info-item .value,
.dark-theme .stat-label,
.dark-theme .stat-value,
.dark-theme .channel-meta,
.dark-theme .status-text,
.dark-theme .quality-score {
  color: #f8fafc !important;
}

.dark-theme .minimal-line-card,
.dark-theme .card-info-body,
.dark-theme .config-section,
.dark-theme .section-header,
.dark-theme .section-title {
  background: #111827 !important;
  border-color: #334155 !important;
}

.dark-theme .info-item {
  border-color: #334155 !important;
}

.dark-theme .arco-table-small .arco-table-th,
.dark-theme .arco-table-small .arco-table-td {
  background: #0f172a !important;
  color: #f8fafc !important;
  border-color: #334155 !important;
}

.channel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding: 20px 24px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid var(--arco-border, #e2e8f0);
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  backdrop-filter: blur(10px);
}

.header-title {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.title-text {
  font-size: 20px;
  font-weight: 600;
  color: #0f172a;
  margin: 0;
  letter-spacing: 0.5px;
}

.title-subtitle {
  font-size: 14px;
  color: #64748b;
  margin: 0;
  font-weight: 400;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* 卡片样式 */
.minimal-line-card {
  border: 1px solid #e2e8f0;
  border-radius: 2px;
  transition: all 0.2s ease;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.9);
  overflow: hidden;
}

.minimal-line-card:hover {
  border-color: var(--arco-primary, #0ea5e9);
  box-shadow: 0 1px 3px rgba(14, 165, 233, 0.1);
}

.minimal-line-card.is-selected {
  border-color: var(--arco-primary, #0ea5e9);
  background: rgba(14, 165, 233, 0.05);
  box-shadow: 0 0 0 1px rgba(14, 165, 233, 0.2);
}

.card-title-content {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.protocol-tag {
  padding: 2px 8px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  color: #475569;
  white-space: nowrap;
}

.name-text {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  flex: 1;
  min-width: 0;
}

.card-info-body {
  padding: 16px 0;
  border-top: 1px solid #f1f5f9;
  margin-top: 8px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 13px;
}

.info-item:last-child {
  margin-bottom: 0;
}

.info-item .label {
  color: #64748b;
  font-weight: 400;
}

.info-item .value {
  color: #334155;
  font-weight: 500;
}

.info-item .value-highlight {
  color: var(--arco-primary, #0ea5e9);
  font-weight: 600;
  font-size: 14px;
}

/* 表格样式 */
:deep(.arco-table-small) {
  font-size: 13px;
}

:deep(.arco-table-small .arco-table-th) {
  font-weight: 600;
  color: #334155;
  background: #f8fafc;
  border-bottom: 2px solid #e2e8f0;
}

:deep(.arco-table-small .arco-table-td) {
  padding: 10px 12px;
  border-bottom: 1px solid #f1f5f9;
}

:deep(.arco-table-small .arco-table-tr:hover) {
  background: #f8fafc;
}

:deep(.arco-table-small .arco-table-tr.arco-table-tr-selected) {
  background: rgba(14, 165, 233, 0.05);
  border-left: 3px solid var(--arco-primary, #0ea5e9);
}

/* 空状态 */
.empty-placeholder {
  margin: 60px 0;
  text-align: center;
}

:deep(.arco-empty-icon) {
  font-size: 48px;
  color: #cbd5e1;
}

:deep(.arco-empty-description) {
  color: #94a3b8;
  font-size: 14px;
  margin-top: 16px;
}

/* 配置区域 */
.config-section {
  margin-top: 16px;
  padding: 20px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
}

.section-header {
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e8f0;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}

/* 帮助对话框 */
.smart-probe-help {
  padding: 24px 0;
}

.help-section {
  margin-bottom: 32px;
}

.help-section h3 {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 12px;
}

.help-section p {
  margin-bottom: 12px;
  line-height: 1.6;
  color: #475569;
}

.help-section ul {
  margin-left: 20px;
}

/* 监控指标对话框样式 */
.metrics-loading {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 300px;
}

.metrics-error {
  margin: 20px 0;
}

.metrics-content {
  padding: 20px 0;
}

.metrics-header {
  margin-bottom: 24px;
}

.quality-score-section {
  display: flex;
  align-items: flex-start;
  gap: 24px;
}

.quality-score-circle {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  position: relative;
  background: linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.quality-score-circle::before {
  content: '';
  position: absolute;
  inset: 4px;
  border-radius: 50%;
  background: rgba(255,255,255,0.05);
  z-index: 1;
}

.quality-score-value {
  font-size: 32px;
  font-weight: 700;
  z-index: 2;
  position: relative;
}

.quality-score-label {
  font-size: 12px;
  opacity: 0.8;
  margin-top: 4px;
  z-index: 2;
  position: relative;
}

.quality-score-level {
  font-size: 14px;
  font-weight: 600;
  margin-top: 4px;
  z-index: 2;
  position: relative;
}

.quality-excellent {
  border: 3px solid var(--arco-success-6, #52c41a);
}

.quality-excellent .quality-score-value,
.quality-excellent .quality-score-level {
  color: var(--arco-success-6, #52c41a);
}

.quality-good {
  border: 3px solid var(--arco-info-6, #1890ff);
}

.quality-good .quality-score-value,
.quality-good .quality-score-level {
  color: var(--arco-info-6, #1890ff);
}

.quality-warning {
  border: 3px solid var(--arco-warning-6, #faad14);
}

.quality-warning .quality-score-value,
.quality-warning .quality-score-level {
  color: var(--arco-warning-6, #faad14);
}

.quality-poor {
  border: 3px solid var(--arco-danger-6, #ff4d4f);
}

.quality-poor .quality-score-value,
.quality-poor .quality-score-level {
  color: var(--arco-danger-6, #ff4d4f);
}

.quality-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-label {
  color: #64748b;
  font-size: 14px;
}

.info-value {
  color: #1e293b;
  font-size: 14px;
  font-weight: 500;
}

.metrics-core {
  margin-bottom: 24px;
}

.metrics-grid {
  display: flex;
  gap: 16px;
  width: 100%;
}

.metric-card {
  flex: 1;
  min-width: 0;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 20px;
  text-align: center;
  transition: all 0.2s ease;
}

.metric-card:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  transform: translateY(-2px);
}

.metric-label {
  font-size: 14px;
  color: #64748b;
  margin-bottom: 8px;
}

.metric-value {
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
}

.metric-success {
  color: var(--arco-success-6, #52c41a);
}

.metric-warning {
  color: var(--arco-warning-6, #faad14);
}

.metric-danger {
  color: var(--arco-danger-6, #ff4d4f);
}

.metrics-details {
  margin-top: 24px;
}

.metrics-collapse :deep(.arco-collapse-item-content) {
  padding: 16px;
}

.metrics-empty {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 300px;
}

/* 暗黑主题适配 */
.dark-theme .metric-card {
  background: #1e293b;
  border-color: #334155;
  flex: 1;
  min-width: 0;
}

.dark-theme .metric-label {
  color: #94a3b8;
}

.dark-theme .metric-value {
  color: #f8fafc;
}

.dark-theme .info-label {
  color: #94a3b8;
}

.dark-theme .info-value {
  color: #f8fafc;
}

.dark-theme .quality-score-circle {
  background: linear-gradient(135deg, rgba(255,255,255,0.05) 0%, rgba(255,255,255,0.02) 100%);
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
}

.dark-theme .quality-score-circle::before {
  background: rgba(255,255,255,0.02);
}

.help-section li {
  margin-bottom: 8px;
  color: #475569;
}

.principle-section h4 {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  margin-top: 16px;
  margin-bottom: 8px;
}

.modal-footer {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  padding-top: 20px;
  border-top: 1px solid #e2e8f0;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .channel-list-container {
    padding: 12px;
  }
  
  .channel-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
    padding: 16px;
  }
  
  .header-actions {
    width: 100%;
    flex-wrap: wrap;
  }
  
  .header-actions > * {
    flex: 1;
    min-width: 120px;
  }
  
  .title-text {
    font-size: 18px;
  }
  
  .card-title-content {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
  
  .protocol-tag {
    align-self: flex-start;
  }
  
  .info-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
    text-align: left;
  }
  
  .info-item .value {
    align-self: flex-start;
  }
}

/* 工业风增强 */
.channel-list-container::before {
  content: '';
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: repeating-linear-gradient(
    90deg,
    rgba(0, 0, 0, 0.02),
    rgba(0, 0, 0, 0.02) 1px,
    transparent 1px,
    transparent 20px
  );
  pointer-events: none;
  z-index: 0;
}

.channel-list-container {
  position: relative;
  z-index: 1;
}

/* 按钮增强 */
:deep(.arco-btn) {
  transition: all 0.2s ease;
}

:deep(.arco-btn:hover) {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

:deep(.arco-btn:active) {
  transform: translateY(0);
}

/* 标签增强 */
:deep(.arco-tag) {
  font-weight: 500;
  letter-spacing: 0.5px;
}

/* 徽章增强 */
:deep(.arco-badge-status) {
  font-weight: 500;
  letter-spacing: 0.5px;
}

/* 表单标签 */
:deep(.arco-form-item-label) {
  font-weight: 500;
  color: #334155;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 140px;
}

/* 确保图标居中 */
:deep(.arco-btn-icon) {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 工具提示增强 */
:deep(.arco-tooltip-content) {
  font-size: 12px;
  padding: 6px 10px;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
</style>