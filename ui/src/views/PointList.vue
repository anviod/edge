<template>
    <div>
        <v-card class="glass-card no-hover">
            <v-card-title class="d-flex align-center py-4 px-6 border-b">
                <v-btn 
                    prepend-icon="mdi-arrow-left" 
                    variant="flat" 
                    color="white" 
                    class="mr-4 text-primary font-weight-bold"
                    elevation="2"
                    @click="$router.back()"
                >
                    返回设备
                </v-btn>

                <v-spacer></v-spacer>
                
                <!-- Point Filtering -->
                <div class="d-flex align-center mr-4" style="max-width: 400px; width: 100%;">
                    <v-text-field
                        v-model="filters.search"
                        label="搜索点位 (ID/名称/地址)"
                        variant="outlined"
                        density="compact"
                        prepend-inner-icon="mdi-magnify"
                        hide-details
                        class="mr-2"
                        clearable
                    ></v-text-field>
                    
                    <v-select
                        v-model="filters.quality"
                        :items="['Good', 'Bad']"
                        label="质量过滤"
                        multiple
                        chips
                        variant="outlined"
                        density="compact"
                        hide-details
                        style="max-width: 180px;"
                    >
                        <template v-slot:selection="{ item, index }">
                            <v-chip v-if="index === 0" size="x-small">{{ item.title }}</v-chip>
                            <span v-if="index === 1" class="text-grey text-caption ml-1">(+{{ filters.quality.length - 1 }})</span>
                        </template>
                    </v-select>
                </div>

                <v-btn 
                    v-if="selection.selectedIds.length > 0"
                    color="error"
                    variant="elevated"
                    prepend-icon="mdi-delete-sweep"
                    class="mr-2"
                    @click="confirmBatchDelete"
                >
                    批量删除 ({{ selection.selectedIds.length }})
                </v-btn>

                <v-btn 
                    color="success" 
                    variant="tonal" 
                    prepend-icon="mdi-plus" 
                    class="mr-2"
                    @click="openAddDialog"
                >
                    新增点位
                </v-btn>
                <v-btn
                    v-if="channelProtocol === 'bacnet-ip' || channelProtocol === 'opc-ua'"
                    color="info"
                    variant="tonal"
                    prepend-icon="mdi-radar"
                    class="mr-2"
                    @click="openDiscoverDialog"
                >
                    扫描点位
                </v-btn>
                <v-btn 
                    color="primary" 
                    variant="tonal" 
                    prepend-icon="mdi-refresh" 
                    @click="fetchPoints"
                    :loading="loading"
                >
                    刷新
                </v-btn>
            </v-card-title>

            <v-progress-linear v-if="loading" indeterminate color="primary"></v-progress-linear>

            <v-card-text class="pa-0">
                <!-- Device Connection Metrics -->
                <div v-if="deviceInfo" class="px-6 py-2 bg-grey-lighten-4 border-b d-flex align-center text-caption text-grey-darken-2">
                    <template v-if="deviceInfo.state === 0 || deviceInfo.state === 1">
                        <v-icon size="small" :icon="deviceInfo.state === 0 ? 'mdi-lan-connect' : 'mdi-lan-pending'" class="mr-2" :color="deviceInfo.state === 0 ? 'success' : 'warning'"></v-icon>
                        <span class="mr-4">
                            <strong>连接状态:</strong> 
                            <v-chip size="x-small" :color="deviceInfo.state === 0 ? 'success' : 'warning'" variant="flat" class="ml-1">
                                {{ deviceInfo.state === 0 ? '已连接' : '不稳定' }}
                            </v-chip>
                        </span>
                        <span class="mr-4"><strong>协议:</strong> {{ getProtocolTransport(channelProtocol) }}</span>
                        <span v-if="metrics.localAddr" class="mr-4"><strong>本地端口:</strong> {{ metrics.localAddr.split(':').pop() }}</span>
                        <span v-if="metrics.remoteAddr" class="mr-4"><strong>远程地址:</strong> {{ metrics.remoteAddr }}</span>
                        <!-- 使用 runtime.success_count 和 last_fail_time 计算更准确的连接信息 -->
                        <span v-if="deviceInfo.runtime" class="mr-4" title="连续成功通信的次数">
                            <strong>连续通信:</strong> 
                            <v-chip size="x-small" color="success" variant="tonal" class="ml-1 font-weight-bold">
                                {{ deviceInfo.runtime.success_count }} 次
                            </v-chip>
                        </span>
                        <span v-if="deviceInfo.runtime?.last_fail_time && new Date(deviceInfo.runtime.last_fail_time).getFullYear() > 1" class="mr-4">
                            <strong>最近失败:</strong> 
                            <span class="text-grey-darken-1 ml-1">{{ formatDate(deviceInfo.runtime.last_fail_time) }}</span>
                        </span>
                    </template>
                    <template v-else>
                        <v-icon size="small" icon="mdi-lan-disconnect" class="mr-2" color="error"></v-icon>
                        <span class="mr-4"><strong>连接状态:</strong> <v-chip size="x-small" color="error" variant="flat" class="ml-1">已断开</v-chip></span>
                        <span class="mr-4"><strong>协议:</strong> {{ getProtocolTransport(channelProtocol) }}</span>
                        <span v-if="deviceInfo.runtime?.last_fail_time && new Date(deviceInfo.runtime.last_fail_time).getFullYear() > 1" class="mr-4"><strong>断开时间:</strong> {{ formatDate(deviceInfo.runtime.last_fail_time) }}</span>
                        <span v-if="deviceInfo.runtime?.last_fail_time && new Date(deviceInfo.runtime.last_fail_time).getFullYear() > 1" class="mr-4"><strong>离线时长:</strong> {{ formatDuration(Math.floor((Date.now() - new Date(deviceInfo.runtime.last_fail_time).getTime()) / 1000)) }}</span>
                    </template>
                    <span v-if="metrics.reconnectCount > 0" class="mr-4"><strong>重连次数:</strong> {{ metrics.reconnectCount }}</span>
                    <span v-if="deviceInfo.quality_score !== undefined" class="mr-4"><strong>质量评分:</strong> {{ deviceInfo.quality_score }}</span>
                </div>

                <v-table hover>
                    <thead>
                        <tr>
                            <th class="text-left" style="width: 40px;">
                                <v-checkbox
                                    v-model="selection.selectAll"
                                    hide-details
                                    density="compact"
                                    @update:model-value="toggleSelectAll"
                                ></v-checkbox>
                            </th>
                            <th class="text-left">点位ID</th>
                            <th class="text-left">点位名称</th>
                            <th class="text-left">读写权限</th>
                            <th class="text-left">数值</th>
                            <th class="text-left">质量</th>
                            <th class="text-left">时间戳</th>
                            <th class="text-left">操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="point in filteredPoints" :key="point.id" :class="{'bg-blue-lighten-5': selection.selectedIds.includes(point.id)}">
                            <td>
                                <v-checkbox
                                    v-model="selection.selectedIds"
                                    :value="point.id"
                                    hide-details
                                    density="compact"
                                ></v-checkbox>
                            </td>
                            <td class="font-weight-medium">{{ point.id }}</td>
                            <td>{{ point.name }}</td>
                            <td>
                                <v-chip
                                    size="x-small"
                                    :color="point.readwrite === 'RW' ? 'success' : 'info'"
                                    variant="outlined"
                                    class="font-weight-medium"
                                >
                                    {{ point.readwrite }}
                                </v-chip>
                            </td>
                            <td style="max-width: 260px;">
                                <div 
                                    class="d-flex align-center cursor-pointer"
                                    @click="showFullValue(point)"
                                    title="点击查看完整值"
                                >
                                    <span class="text-h6 font-weight-bold text-primary text-truncate d-block" style="max-width: 100%;">
                                        {{ formatValue(point.value) }}
                                    </span>
                                    <span v-if="point.unit" class="text-caption ml-1 flex-shrink-0">{{ point.unit }}</span>
                                </div>
                                <div class="text-caption text-grey-darken-1 mt-1">
                                    {{ getRegisterHint(point) }}
                                </div>
                            </td>
                            <td>
                                <v-chip 
                                    size="small" 
                                    :color="isQualityGood(point.quality) ? 'success' : 'error'" 
                                    variant="flat"
                                >
                                    {{ point.quality }}
                                </v-chip>
                            </td>
                            <td class="text-body-2">{{ formatDate(point.timestamp) }}</td>
                            <td>
                                <v-btn 
                                    v-if="point.readwrite === 'RW' || point.readwrite === 'W'"
                                    color="secondary" 
                                    size="x-small" 
                                    variant="tonal"
                                    icon="mdi-pencil"
                                    class="mr-1"
                                    @click="openWriteDialog(point)"
                                    title="写入数值"
                                ></v-btn>
                                <v-btn 
                                    icon="mdi-file-edit-outline"
                                    size="x-small"
                                    variant="tonal"
                                    color="primary"
                                    class="mr-1"
                                    @click="openEditDialog(point)"
                                    title="编辑点位配置"
                                ></v-btn>
                                <v-btn 
                                    icon="mdi-delete-outline"
                                    size="x-small"
                                    variant="tonal"
                                    color="error"
                                    @click="confirmDelete(point)"
                                    title="删除点位"
                                ></v-btn>
                                <v-btn
                                    icon="mdi-bug-outline"
                                    size="x-small"
                                    variant="tonal"
                                    color="info"
                                    class="ml-1"
                                    @click="openDebug(point)"
                                    title="调试点位"
                                ></v-btn>
                            </td>
                        </tr>
                        <tr v-if="!loading && filteredPoints.length === 0">
                            <td colspan="8" class="text-center pa-8 text-grey">
                                暂无匹配的点位数据
                                <v-btn v-if="points.length === 0" color="primary" variant="text" class="ml-2" @click="openCloneDialog" prepend-icon="mdi-content-copy">
                                    复制其它设备点位
                                </v-btn>
                                <v-btn v-else color="secondary" variant="text" class="ml-2" @click="filters.search = ''; filters.quality = []">
                                    清除过滤器
                                </v-btn>
                            </td>
                        </tr>
                    </tbody>
                </v-table>
            </v-card-text>
        </v-card>

        <v-dialog v-model="cloneDialog.visible" width="100%" max-width="100%" persistent class="clone-dialog-full">
            <v-card class="rounded-xl">
                <v-card-title class="text-h6 d-flex align-center">
                    <v-icon icon="mdi-content-copy" class="mr-2"></v-icon>
                    克隆其它设备点位
                </v-card-title>
                <v-card-text class="pt-3 pb-2">
                    <v-row class="mb-2" align="center">
                        <v-col cols="12" md="4" class="pb-2 pb-md-0 pr-md-2">
                            <v-select
                                v-model="cloneDialog.selectedChannel"
                                :items="cloneDialog.channels"
                                item-title="name"
                                item-value="id"
                                label="选择通道"
                                variant="outlined"
                                density="compact"
                                :loading="cloneDialog.loading"
                                @update:model-value="onCloneChannelChange"
                            ></v-select>
                        </v-col>
                        <v-col cols="12" md="4" class="pb-2 pb-md-0 pr-md-2">
                            <v-select
                                v-model="cloneDialog.selectedDevice"
                                :items="cloneDialog.devices"
                                item-title="name"
                                item-value="id"
                                label="选择设备"
                                variant="outlined"
                                density="compact"
                                :loading="cloneDialog.loading"
                                @update:model-value="onCloneDeviceChange"
                            ></v-select>
                        </v-col>
                        <v-col cols="12" md="4">
                            <v-text-field
                                v-model="cloneDialog.search"
                                label="按名称或地址过滤"
                                variant="outlined"
                                density="compact"
                                prepend-inner-icon="mdi-magnify"
                                clearable
                                hide-details="auto"
                            ></v-text-field>
                        </v-col>
                    </v-row>
                    <v-row class="mb-2" align="center" v-if="cloneDialog.points && cloneDialog.points.length > 0">
                        <v-col cols="12" class="d-flex align-center">
                            <v-checkbox
                                v-model="cloneDialog.selectAll"
                                density="compact"
                                hide-details
                                label="全选"
                                class="mr-4"
                                @change="toggleCloneSelectAll"
                            />
                            <div class="text-caption text-grey-darken-1">
                                已选择 {{ cloneDialog.selected.length }} / {{ cloneDialog.points.length }}
                            </div>
                        </v-col>
                    </v-row>
                    <v-table fixed-header height="360">
                        <thead>
                            <tr>
                                <th class="text-left" style="width:40px"></th>
                                <th class="text-left">名称</th>
                                <th class="text-left">地址</th>
                                <th class="text-left">数据类型</th>
                                <th class="text-left">单位</th>
                                <th class="text-left">读写</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="p in filteredClonePoints" :key="p.id">
                                <td>
                                    <v-checkbox
                                        v-model="cloneDialog.selected"
                                        :value="p"
                                        density="compact"
                                        hide-details
                                    />
                                </td>
                                <td>{{ p.name }}</td>
                                <td class="text-no-wrap">{{ p.address }}</td>
                                <td>{{ p.datatype }}</td>
                                <td>{{ p.unit }}</td>
                                <td>{{ p.readwrite }}</td>
                            </tr>
                            <tr v-if="!cloneDialog.loading && cloneDialog.points.length === 0">
                                <td colspan="6" class="text-center pa-6 text-grey">请选择通道与设备以加载点位</td>
                            </tr>
                        </tbody>
                    </v-table>
                </v-card-text>
                <v-card-actions class="pa-4">
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="cloneDialog.visible = false">取消</v-btn>
                    <v-btn color="primary" :loading="cloneDialog.loading" :disabled="cloneDialog.selected.length === 0" @click="executeClone">
                        克隆所选点位
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <!-- Point Config Dialog (Add/Edit) -->
        <v-dialog v-model="pointDialog.visible" max-width="80%" persistent>
            <v-card class="rounded-xl">
                <v-card-title class="text-h5 pa-4 bg-primary text-white d-flex align-center">
                    <div class="d-flex align-center">
                        <v-icon :icon="pointDialog.isEdit ? 'mdi-file-edit' : 'mdi-plus-circle'" class="mr-2"></v-icon>
                        <span>{{ pointDialog.isEdit ? '编辑点位' : '新增点位' }}</span>
                    </div>
                    <v-spacer></v-spacer>
                    <v-btn
                        v-if="recentFormatIds.length > 0"
                        class="mr-2"
                        color="white"
                        variant="outlined"
                        size="small"
                        prepend-icon="mdi-swap-horizontal"
                        :disabled="recentFormatIds.length < 2"
                        @click="toggleRecentFormats"
                        title="Toggle recent formats / 在最近两种格式间切换"
                    >
                        最近格式切换
                    </v-btn>
                    <v-btn
                        icon="mdi-help-circle-outline"
                        variant="text"
                        color="white"
                        @click="helpDialog.visible = true"
                        title="点位公式与解码帮助"
                    ></v-btn>
                </v-card-title>
                <v-card-text class="pa-4 pt-6">
                    <v-form ref="pointForm" @submit.prevent="submitPoint">
                        <v-row>
                            <v-col cols="6">
                                <v-text-field
                                    v-model="pointDialog.form.id"
                                    label="点位ID"
                                    variant="outlined"
                                    density="compact"
                                    :readonly="pointDialog.isEdit"
                                    hint="唯一标识符"
                                    persistent-hint
                                ></v-text-field>
                            </v-col>
                            <v-col cols="6">
                                <v-text-field
                                    v-model="pointDialog.form.name"
                                    label="点位名称"
                                    variant="outlined"
                                    density="compact"
                                ></v-text-field>
                            </v-col>

                            <!-- Modbus Specific -->
                            <template v-if="channelProtocol.startsWith('modbus')">
                                <v-col cols="6">
                                    <v-select
                                        v-model="pointDialog.registerType"
                                        label="寄存器类型"
                                        :items="registerTypes"
                                        item-title="title"
                                        item-value="value"
                                        variant="outlined"
                                        density="compact"
                                        @update:model-value="updateAddress"
                                    ></v-select>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model.number="pointDialog.registerIndex"
                                        label="寄存器索引"
                                        type="number"
                                        :min="getRegisterIndexMin()"
                                        :max="getRegisterIndexMax()"
                                        :error-messages="registerIndexError"
                                        variant="outlined"
                                        density="compact"
                                        @input="validateRegisterIndex; updateAddress"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model.number="pointDialog.registerOffset"
                                        label="起始偏移量"
                                        type="number"
                                        min="0"
                                        max="9999"
                                        :error-messages="registerOffsetError"
                                        variant="outlined"
                                        density="compact"
                                        hint="数据读取起始偏移量 (默认: 0)"
                                        persistent-hint
                                        @input="validateRegisterOffset; updateAddress"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="Modbus 地址"
                                        variant="outlined"
                                        density="compact"
                                        hint="自动生成 (例如: 40001)"
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model.number="pointDialog.functionCode"
                                        label="功能码"
                                        type="number"
                                        min="1"
                                        max="255"
                                        variant="outlined"
                                        density="compact"
                                        hint="默认: 根据寄存器类型自动确定"
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                            </template>

                            <!-- BACnet Specific -->
                            <template v-else-if="channelProtocol === 'bacnet-ip'">
                                <v-col cols="6">
                                    <v-select
                                        v-model="pointDialog.bacnetType"
                                        label="对象类型"
                                        :items="bacnetObjectTypes"
                                        variant="outlined"
                                        density="compact"
                                        @update:model-value="updateBACnetAddress"
                                    ></v-select>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model.number="pointDialog.bacnetInstance"
                                        label="实例 ID"
                                        type="number"
                                        min="0"
                                        variant="outlined"
                                        density="compact"
                                        @input="updateBACnetAddress"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="12">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="BACnet 地址"
                                        variant="outlined"
                                        density="compact"
                                        readonly
                                        hint="格式: Type:Instance"
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                            </template>

                            <!-- OPC UA Specific -->
                            <template v-else-if="channelProtocol === 'opc-ua'">
                                <v-col cols="12">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="Node ID"
                                        placeholder="ns=2;s=Demo.Static.Scalar.Double"
                                        variant="outlined"
                                        density="compact"
                                        hint="例如: ns=2;s=Demo..."
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                            </template>

                            <!-- S7 Specific -->
                            <template v-else-if="channelProtocol === 's7'">
                                <v-col cols="12">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="S7 地址"
                                        placeholder="DB1.DBD0"
                                        variant="outlined"
                                        density="compact"
                                        hint="例如: DB1.DBD0, M0.0"
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                            </template>

                            <!-- EtherNet/IP Specific -->
                            <template v-else-if="channelProtocol === 'ethernet-ip'">
                                <v-col cols="12">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="Tag 名称"
                                        placeholder="Program:Main.MyTag"
                                        variant="outlined"
                                        density="compact"
                                        hint="例如: Program:Main.MyTag"
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                            </template>

                            <!-- Mitsubishi Specific -->
                            <template v-else-if="channelProtocol === 'mitsubishi-slmp'">
                                <v-col cols="12">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="地址"
                                        placeholder="D100"
                                        variant="outlined"
                                        density="compact"
                                        hint="格式: D100, M0, X0, D20.2, D100.16L"
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                            </template>

                            <!-- Omron FINS Specific -->
                            <template v-else-if="channelProtocol === 'omron-fins'">
                                <v-col cols="12">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="地址"
                                        placeholder="D100"
                                        variant="outlined"
                                        density="compact"
                                        hint="格式: CIO1.2, D100, W3.4, EM10.100"
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                            </template>

                            <!-- DL/T645 Specific -->
                            <template v-else-if="channelProtocol === 'dlt645'">
                                <v-col cols="6">
                                    <v-text-field
                                        v-model="pointDialog.dlt645DeviceAddr"
                                        label="设备地址"
                                        variant="outlined"
                                        density="compact"
                                        hint="通常与设备配置一致"
                                        persistent-hint
                                        @input="updateDLT645Address"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model="pointDialog.dlt645DataID"
                                        label="数据标识 (DI)"
                                        placeholder="02-01-01-00"
                                        variant="outlined"
                                        density="compact"
                                        hint="格式: XX-XX-XX-XX"
                                        persistent-hint
                                        @input="updateDLT645Address"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="12">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="完整地址"
                                        variant="outlined"
                                        density="compact"
                                        readonly
                                        hint="格式: 设备地址#数据标识"
                                        persistent-hint
                                    ></v-text-field>
                                </v-col>
                            </template>

                            <!-- Fallback -->
                            <template v-else>
                                <v-col cols="12">
                                    <v-text-field
                                        v-model="pointDialog.form.address"
                                        label="地址"
                                        variant="outlined"
                                        density="compact"
                                    ></v-text-field>
                                </v-col>
                            </template>
                            <template v-if="channelProtocol.startsWith('modbus')">
                                <v-col cols="12">
                                    <v-expansion-panels>
                                        <v-expansion-panel>
                                            <v-expansion-panel-title>高级设置</v-expansion-panel-title>
                                            <v-expansion-panel-text>
                                                <v-row>
                                                    <v-col cols="12">
                                                        <v-select
                                                            v-model="formatPresetSelected"
                                                            :items="filteredFormatPresets"
                                                            item-title="label"
                                                            item-value="id"
                                                            label="数据格式"
                                                            variant="outlined"
                                                            density="compact"
                                                            clearable
                                                            @update:model-value="onSelectFormatPreset"
                                                        ></v-select>
                                                    </v-col>
                                                    <v-col cols="4">
                                                        <v-select
                                                            v-model="pointDialog.parseType"
                                                            :items="filteredParseTypes"
                                                            item-title="label"
                                                            item-value="value"
                                                            label="解析类型"
                                                            variant="outlined"
                                                            density="compact"
                                                        ></v-select>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-select
                                                            v-model="pointDialog.form.datatype"
                                                            label="数据类型(存储)"
                                                            :items="datatypeOptions"
                                                            variant="outlined"
                                                            density="compact"
                                                        ></v-select>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-select
                                                            v-model="pointDialog.form.read_formula_template"
                                                            :items="formulaTemplates"
                                                            item-title="label"
                                                            item-value="expr"
                                                            label="读公式模板"
                                                            variant="outlined"
                                                            density="compact"
                                                            clearable
                                                            @update:model-value="onSelectFormulaTemplate('read')"
                                                        ></v-select>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-text-field
                                                            v-model="pointDialog.form.read_formula"
                                                            label="读公式 (使用变量 v)"
                                                            variant="outlined"
                                                            density="compact"
                                                            :error-messages="formulaErrors.read"
                                                            @input="validateFormula('read')"
                                                        ></v-text-field>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-select
                                                            v-model="pointDialog.form.readwrite"
                                                            label="读写权限"
                                                            :items="['R', 'RW']"
                                                            variant="outlined"
                                                            density="compact"
                                                        ></v-select>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-select
                                                            v-model="pointDialog.form.write_formula_template"
                                                            :items="formulaTemplates"
                                                            item-title="label"
                                                            item-value="expr"
                                                            label="写公式模板"
                                                            variant="outlined"
                                                            density="compact"
                                                            clearable
                                                            @update:model-value="onSelectFormulaTemplate('write')"
                                                        ></v-select>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-text-field
                                                            v-model="pointDialog.form.write_formula"
                                                            label="写公式 (使用变量 v)"
                                                            variant="outlined"
                                                            density="compact"
                                                            :error-messages="formulaErrors.write"
                                                            @input="validateFormula('write')"
                                                        ></v-text-field>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-text-field
                                                            v-model="pointDialog.form.unit"
                                                            label="单位"
                                                            variant="outlined"
                                                            density="compact"
                                                        ></v-text-field>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-text-field
                                                            v-model.number="pointDialog.form.scale"
                                                            label="缩放比例"
                                                            type="number"
                                                            step="0.01"
                                                            variant="outlined"
                                                            density="compact"
                                                            hint="默认为 1.0"
                                                        ></v-text-field>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-text-field
                                                            v-model.number="pointDialog.form.offset"
                                                            label="偏移量"
                                                            type="number"
                                                            step="0.01"
                                                            variant="outlined"
                                                            density="compact"
                                                            hint="默认为 0"
                                                        ></v-text-field>
                                                    </v-col>
                                                    <v-col cols="6">
                                                        <v-text-field
                                                            v-model="pointDialog.defaultValue"
                                                            label="默认值"
                                                            variant="outlined"
                                                            density="compact"
                                                        ></v-text-field>
                                                    </v-col>
                                                    <v-col cols="6" class="d-flex align-center">
                                                        <v-btn
                                                            color="primary"
                                                            variant="tonal"
                                                            prepend-icon="mdi-flash"
                                                            @click="openQuickValidate"
                                                        >
                                                            快速验证
                                                        </v-btn>
                                                        <v-btn
                                                            class="ml-2"
                                                            color="secondary"
                                                            variant="text"
                                                            prepend-icon="mdi-clipboard-text"
                                                            @click="openTemplateDialog"
                                                        >
                                                            协议模板
                                                        </v-btn>
                                                    </v-col>
                                                </v-row>
                                            </v-expansion-panel-text>
                                        </v-expansion-panel>
                                    </v-expansion-panels>
                                </v-col>
                            </template>
                            <template v-else>
                                <v-col cols="12">
                                    <v-select
                                        v-model="formatPresetSelected"
                                        :items="filteredFormatPresets"
                                        item-title="label"
                                        item-value="id"
                                        label="数据格式"
                                        variant="outlined"
                                        density="compact"
                                        clearable
                                        @update:model-value="onSelectFormatPreset"
                                    ></v-select>
                                </v-col>
                                <v-col cols="4">
                                    <v-select
                                        v-model="pointDialog.byteLength"
                                        label="字节数"
                                        :items="[1, 2, 4, 8]"
                                        variant="outlined"
                                        density="compact"
                                    ></v-select>
                                </v-col>
                                <v-col cols="4">
                                    <v-select
                                        v-model="pointDialog.wordOrderOption"
                                        :items="wordOrderOptionsForBytes"
                                        item-title="label"
                                        item-value="value"
                                        label="WordOrder(字序)"
                                        :disabled="pointDialog.byteLength === 1"
                                        variant="outlined"
                                        density="compact"
                                    ></v-select>
                                </v-col>
                                <v-col cols="4">
                                    <v-select
                                        v-model="pointDialog.parseType"
                                        :items="filteredParseTypes"
                                        item-title="label"
                                        item-value="value"
                                        label="解析类型"
                                        variant="outlined"
                                        density="compact"
                                    ></v-select>
                                </v-col>
                                <v-col cols="6">
                                    <v-select
                                        v-model="pointDialog.form.datatype"
                                        label="数据类型(存储)"
                                        :items="datatypeOptions"
                                        variant="outlined"
                                        density="compact"
                                    ></v-select>
                                </v-col>
                                <v-col cols="6">
                                    <v-select
                                        v-model="pointDialog.form.read_formula_template"
                                        :items="formulaTemplates"
                                        item-title="label"
                                        item-value="expr"
                                        label="读公式模板"
                                        variant="outlined"
                                        density="compact"
                                        clearable
                                        @update:model-value="onSelectFormulaTemplate('read')"
                                    ></v-select>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model="pointDialog.form.read_formula"
                                        label="读公式 (使用变量 v)"
                                        variant="outlined"
                                        density="compact"
                                        :error-messages="formulaErrors.read"
                                        @input="validateFormula('read')"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-select
                                        v-model="pointDialog.form.readwrite"
                                        label="读写权限"
                                        :items="['R', 'RW']"
                                        variant="outlined"
                                        density="compact"
                                    ></v-select>
                                </v-col>
                                <v-col cols="6">
                                    <v-select
                                        v-model="pointDialog.form.write_formula_template"
                                        :items="formulaTemplates"
                                        item-title="label"
                                        item-value="expr"
                                        label="写公式模板"
                                        variant="outlined"
                                        density="compact"
                                        clearable
                                        @update:model-value="onSelectFormulaTemplate('write')"
                                    ></v-select>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model="pointDialog.form.write_formula"
                                        label="写公式 (使用变量 v)"
                                        variant="outlined"
                                        density="compact"
                                        :error-messages="formulaErrors.write"
                                        @input="validateFormula('write')"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model="pointDialog.form.unit"
                                        label="单位"
                                        variant="outlined"
                                        density="compact"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model.number="pointDialog.form.scale"
                                        label="缩放比例"
                                        type="number"
                                        step="0.01"
                                        variant="outlined"
                                        density="compact"
                                        hint="默认为 1.0"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model.number="pointDialog.form.offset"
                                        label="偏移量"
                                        type="number"
                                        step="0.01"
                                        variant="outlined"
                                        density="compact"
                                        hint="默认为 0"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6">
                                    <v-text-field
                                        v-model="pointDialog.defaultValue"
                                        label="默认值"
                                        variant="outlined"
                                        density="compact"
                                    ></v-text-field>
                                </v-col>
                                <v-col cols="6" class="d-flex align-center">
                                    <v-btn
                                        color="primary"
                                        variant="tonal"
                                        prepend-icon="mdi-flash"
                                        @click="openQuickValidate"
                                    >
                                        快速验证
                                    </v-btn>
                                    <v-btn
                                        class="ml-2"
                                        color="secondary"
                                        variant="text"
                                        prepend-icon="mdi-clipboard-text"
                                        @click="openTemplateDialog"
                                    >
                                        协议模板
                                    </v-btn>
                                </v-col>
                            </template>
                        </v-row>
                    </v-form>
                </v-card-text>
                <v-card-actions class="pa-4 pt-0">
                    <v-spacer></v-spacer>
                    <v-btn color="grey-darken-1" variant="text" @click="pointDialog.visible = false">取消</v-btn>
                    <v-btn color="primary" variant="elevated" @click="submitPoint" :loading="pointDialog.loading">保存</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="quickValidate.visible" max-width="640">
            <v-card>
                <v-card-title class="d-flex align-center">
                    <span class="text-h6">快速验证当前解析配置</span>
                    <v-spacer></v-spacer>
                    <v-chip
                        v-if="quickValidate.status"
                        :color="quickValidate.status === 'pass' ? 'green' : 'red'"
                        size="small"
                        class="mr-2"
                    >
                        {{ quickValidate.status === 'pass' ? '验证通过' : '未通过' }}
                    </v-chip>
                    <v-btn icon="mdi-close" variant="text" @click="quickValidate.visible = false"></v-btn>
                </v-card-title>
                <v-card-text>
                    <v-row dense>
                        <v-col cols="12">
                            <v-text-field
                                v-model="quickValidate.rawHex"
                                label="原始十六进制报文"
                                placeholder="例如: 01 0A FF 00"
                                variant="outlined"
                                density="compact"
                            ></v-text-field>
                        </v-col>
                        <v-col cols="12">
                            <v-text-field
                                v-model="quickValidate.registerValues"
                                label="寄存器值列表(可选, 以空格或逗号分隔, 支持0x前缀)"
                                placeholder="例如: 0x1234 0x5678 或 4660 22136"
                                variant="outlined"
                                density="compact"
                            ></v-text-field>
                        </v-col>
                        <v-col cols="12" sm="6">
                            <v-text-field
                                v-model="quickValidate.registerBaseAddress"
                                label="起始寄存器地址(仅用于标注)"
                                placeholder="例如: 40001"
                                variant="outlined"
                                density="compact"
                            ></v-text-field>
                        </v-col>
                        <v-col cols="12">
                            <v-text-field
                                v-model="quickValidate.expected"
                                label="期望工程值(可选)"
                                placeholder="例如: 230.1 或 LongABCD 11112222"
                                variant="outlined"
                                density="compact"
                            ></v-text-field>
                        </v-col>
                        <v-col cols="12">
                            <div class="text-caption mb-1">解析结果预览</div>
                            <div class="pa-3 rounded bg-grey-lighten-4 font-mono text-body-2">
                                <span v-html="quickValidate.previewHtml"></span>
                            </div>
                        </v-col>
                    </v-row>
                </v-card-text>
                <v-card-actions class="pa-4 pt-0">
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="quickValidate.visible = false">关闭</v-btn>
                    <v-btn color="primary" variant="elevated" @click="runQuickValidate">
                        立即验证
                    </v-btn>
                    <v-btn
                        color="secondary"
                        variant="tonal"
                        :disabled="quickValidate.status !== 'pass'"
                        @click="saveCurrentAsTemplate"
                    >
                        保存为模板
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="templateDialog.visible" max-width="900">
            <v-card>
                <v-card-title class="d-flex align-center">
                    <span class="text-h6">协议模板示例</span>
                    <v-spacer></v-spacer>
                    <v-text-field
                        v-model="templateDialog.search"
                        prepend-inner-icon="mdi-magnify"
                        label="搜索模板(名称/描述)"
                        variant="outlined"
                        density="compact"
                        hide-details
                        style="max-width: 260px"
                    ></v-text-field>
                    <v-btn icon="mdi-close" variant="text" @click="templateDialog.visible = false"></v-btn>
                </v-card-title>
                <v-card-text>
                    <v-row dense>
                        <v-col
                            v-for="tpl in filteredPointTemplates"
                            :key="tpl.id"
                            cols="12"
                            md="6"
                        >
                            <v-card variant="outlined" class="pa-3">
                                <div class="d-flex align-center mb-1">
                                    <span class="font-weight-medium">{{ tpl.name }}</span>
                                    <v-spacer></v-spacer>
                                    <v-chip size="x-small" color="primary" class="mr-1">
                                        {{ tpl.protocol }}
                                    </v-chip>
                                </div>
                                <div class="text-caption text-grey-darken-1 mb-2">
                                    {{ tpl.description }}
                                </div>
                                <div class="text-caption mb-1">
                                    类型: {{ tpl.parseType }} / {{ tpl.datatype }},
                                    字节数: {{ tpl.byteLength || 'N/A' }},
                                    字序: {{ tpl.wordOrder || 'N/A' }},
                                    单位: {{ tpl.unit || '-' }},
                                    默认值: {{ tpl.defaultValue === '' ? '-' : tpl.defaultValue }},
                                    权限: {{ tpl.readwrite }}
                                </div>
                                <div class="mt-2 d-flex">
                                    <v-btn
                                        color="primary"
                                        size="small"
                                        variant="elevated"
                                        prepend-icon="mdi-clipboard-arrow-right"
                                        @click="applyTemplate(tpl)"
                                    >
                                        套用模板
                                    </v-btn>
                                    <v-btn
                                        class="ml-2"
                                        color="secondary"
                                        size="small"
                                        variant="text"
                                        prepend-icon="mdi-content-copy"
                                        @click="copyTemplate(tpl)"
                                    >
                                        复制配置
                                    </v-btn>
                                </div>
                            </v-card>
                        </v-col>
                    </v-row>
                </v-card-text>
                <v-card-actions class="pa-4 pt-0">
                    <v-spacer></v-spacer>
                    <v-btn color="primary" variant="elevated" @click="templateDialog.visible = false">关闭</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="helpDialog.visible" max-width="900">
            <v-card>
                <v-card-title class="d-flex align-center">
                    <span class="text-h6">点位解码与公式使用帮助</span>
                    <v-spacer></v-spacer>
                    <v-btn icon="mdi-close" variant="text" @click="helpDialog.visible = false"></v-btn>
                </v-card-title>
                <v-card-text class="pt-2">
                    <v-text-field
                        v-model="helpDialog.search"
                        prepend-inner-icon="mdi-magnify"
                        label="搜索关键字 (协议 / 函数 / 示例)"
                        variant="outlined"
                        density="compact"
                        class="mb-4"
                        clearable
                    ></v-text-field>
                    <v-expansion-panels multiple>
                        <v-expansion-panel
                            v-for="section in filteredHelpSections"
                            :key="section.id"
                        >
                            <v-expansion-panel-title>
                                {{ section.title }}
                            </v-expansion-panel-title>
                            <v-expansion-panel-text>
                                <div v-for="item in section.items" :key="item.title" class="mb-4">
                                    <div class="d-flex align-center mb-1">
                                        <span class="font-weight-medium">{{ item.title }}</span>
                                        <v-spacer></v-spacer>
                                        <v-btn
                                            v-if="item.snippet"
                                            size="x-small"
                                            variant="text"
                                            color="primary"
                                            @click="copySnippet(item.snippet)"
                                        >
                                            复制示例
                                        </v-btn>
                                    </div>
                                    <div class="text-body-2 mb-1">{{ item.desc }}</div>
                                    <div v-if="item.snippet" class="pa-2 rounded bg-grey-lighten-4 font-mono text-body-2">
                                        {{ item.snippet }}
                                    </div>
                                </div>
                            </v-expansion-panel-text>
                        </v-expansion-panel>
                    </v-expansion-panels>
                </v-card-text>
                <v-card-actions class="pa-4">
                    <v-spacer></v-spacer>
                    <v-btn color="primary" variant="elevated" @click="helpDialog.visible = false">关闭</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <!-- Scan Dialog -->
        <v-dialog v-model="scanDialog.visible" max-width="1200px" persistent>
            <v-card>
                <v-card-title class="d-flex align-center bg-info text-white">
                    <v-icon icon="mdi-radar" class="mr-2"></v-icon>
                    扫描点位 (对象发现)
                    <v-spacer></v-spacer>
                    <v-btn icon="mdi-close" variant="text" @click="scanDialog.visible = false"></v-btn>
                </v-card-title>
                <v-card-text class="pa-4">
                    <v-row class="mb-2" align="center">
                        <v-col cols="12" sm="4">
                            <div class="text-caption text-grey-darken-1">
                                正在扫描设备 (ID: {{ deviceInfo?.config?.device_id }}) 的对象列表...
                            </div>
                        </v-col>
                        <v-col cols="12" sm="9" class="d-flex align-center justify-end scan-toolbar">
                            <v-btn-toggle v-model="scanDialog.mode" color="primary" density="compact" class="mr-3" mandatory>
                                <v-btn value="fast" variant="tonal" size="small" title="结构 + 元数据">快速扫描</v-btn>
                                <v-btn value="deep" variant="tonal" size="small" title="含实时值">深度扫描</v-btn>
                            </v-btn-toggle>
                            <v-btn color="primary" :loading="scanDialog.loading" prepend-icon="mdi-radar" @click="scanPoints">
                                开始扫描
                            </v-btn>
                            <v-switch
                                class="ml-4"
                                hide-details
                                color="primary"
                                density="compact"
                                v-model="scanDialog.varsOnly"
                                :label="scanDialog.varsOnly ? '仅显示变量' : '显示全部'"
                            ></v-switch>
                        </v-col>
                    </v-row>
                    
                    <v-divider class="mb-4"></v-divider>
                    
                    <v-table hover density="compact">
                        <thead>
                            <tr>
                                <th style="width: 50px">
                                    <v-checkbox-btn
                                        v-model="scanDialog.selectAll"
                                        @update:model-value="toggleSelectAllScan"
                                        density="compact"
                                        hide-details
                                    ></v-checkbox-btn>
                                </th>
                                <th class="text-left">状态</th>
                                <th class="text-left">对象名称/NodeID</th>
                                <th class="text-left">类型</th>
                                <th class="text-left" v-if="channelProtocol !== 'opc-ua'">实例</th>
                                <th class="text-left" v-if="channelProtocol !== 'opc-ua'">当前值</th>
                                <th class="text-left" v-if="channelProtocol !== 'opc-ua'">单位</th>
                                <th class="text-left">描述/DataType</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-if="scanDialog.results.length === 0 && !scanDialog.loading">
                                <td colspan="7" class="text-center text-grey py-8">
                                    <v-icon icon="mdi-magnify" size="large" class="mb-2"></v-icon>
                                    <div>点击"开始扫描"获取设备对象列表</div>
                                </td>
                            </tr>
                            <tr v-for="obj in scanFilteredResults" :key="obj.isOpcNode ? obj.node_id : (obj.type + ':' + obj.instance)">
                                <td>
                                    <v-checkbox-btn
                                        v-model="scanDialog.selected"
                                        :value="obj"
                                        :disabled="obj.diff_status === 'existing'"
                                        density="compact"
                                        hide-details
                                    ></v-checkbox-btn>
                                </td>
                                <td>
                                    <v-chip
                                        v-if="obj.diff_status"
                                        size="x-small"
                                        :color="getStatusColor(obj.diff_status)"
                                        class="font-weight-bold"
                                    >
                                        {{ getStatusText(obj.diff_status) }}
                                    </v-chip>
                                    <span v-else>-</span>
                                </td>
                                <!-- Object Name with Indentation for OPC UA -->
                                <td :style="obj.isOpcNode ? { paddingLeft: (obj.level * 20 + 16) + 'px' } : {}">
                                    <v-icon v-if="obj.isOpcNode" :icon="obj.type === 'Variable' ? 'mdi-tag-outline' : 'mdi-folder-outline'" size="small" class="mr-1"></v-icon>
                                    {{ obj.object_name || obj.name || '-' }}
                                    <div v-if="obj.isOpcNode" class="text-caption text-grey">{{ obj.node_id }}</div>
                                </td>
                                <td>{{ obj.type }}</td>
                                <td v-if="channelProtocol !== 'opc-ua'">{{ obj.instance }}</td>
                                <td v-if="channelProtocol !== 'opc-ua'">{{ obj.present_value }}</td>
                                <td v-if="channelProtocol !== 'opc-ua'">{{ obj.units || '-' }}</td>
                                <td>{{ obj.isOpcNode ? (obj.data_type || '-') : (obj.description || '-') }}</td>
                            </tr>
                        </tbody>
                    </v-table>
                </v-card-text>
                <v-card-actions class="pa-4 border-t">
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="scanDialog.visible = false">取消</v-btn>
                    <v-btn 
                        color="primary" 
                        variant="elevated"
                        @click="addSelectedPoints" 
                        :disabled="scanDialog.selected.length === 0 || scanDialog.loading"
                        :loading="scanDialog.loading"
                    >
                        添加选定点位 ({{ scanDialog.selected.length }})
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <!-- Delete Confirmation Dialog -->
        <v-dialog v-model="deleteDialog.visible" max-width="400">
            <v-card class="rounded-xl">
                <v-card-title class="text-h5 bg-error text-white pa-4">
                    <v-icon icon="mdi-alert" class="mr-2"></v-icon>
                    确认删除
                </v-card-title>
                <v-card-text class="pa-6 text-center">
                    <template v-if="deleteDialog.isBatch">
                        确定要批量删除选中的 <span class="text-error font-weight-bold">{{ deleteDialog.batchCount }}</span> 个点位吗？
                    </template>
                    <template v-else>
                        确定要删除点位 <span class="text-error font-weight-bold">{{ deleteDialog.point?.name || deleteDialog.point?.id }}</span> 吗？
                    </template>
                    <div class="mt-2 text-grey text-caption">此操作不可撤销。</div>
                </v-card-text>
                <v-card-actions class="pa-4 pt-0">
                    <v-spacer></v-spacer>
                    <v-btn color="grey-darken-1" variant="text" @click="deleteDialog.visible = false">取消</v-btn>
                    <v-btn color="error" variant="elevated" @click="executeDelete" :loading="deleteDialog.loading">确认删除</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <!-- Write Dialog -->
        <v-dialog v-model="writeDialog.visible" max-width="400" persistent>
            <v-card class="rounded-xl bg-white elevation-10">
                <v-card-title class="text-h5 pa-4 bg-primary text-white">
                    <v-icon icon="mdi-pencil" class="mr-2"></v-icon>
                    写入数值
                </v-card-title>
                <v-card-text class="pa-4 pt-6">
                    <v-form @submit.prevent="submitWrite">
                        <v-text-field
                            v-model="writeDialog.deviceID"
                            label="设备ID"
                            variant="outlined"
                            readonly
                            density="compact"
                            prepend-inner-icon="mdi-devices"
                            class="mb-2"
                        ></v-text-field>
                        <v-text-field
                            v-model="writeDialog.pointID"
                            label="点位ID"
                            variant="outlined"
                            readonly
                            density="compact"
                            prepend-inner-icon="mdi-tag"
                            class="mb-2"
                        ></v-text-field>
                        <template v-if="isBoolType(writeDialog.dataType)">
                            <v-switch
                                v-model="writeDialog.valueBool"
                                inset
                                color="primary"
                                class="mt-2"
                                :label="writeDialog.valueBool ? 'TRUE' : 'FALSE'"
                            ></v-switch>
                        </template>
                        <template v-else-if="isStringType(writeDialog.dataType)">
                            <v-text-field
                                v-model="writeDialog.valueStr"
                                label="新数值"
                                variant="outlined"
                                density="comfortable"
                                prepend-inner-icon="mdi-cog"
                                placeholder="请输入要写入的字符串"
                                autofocus
                            ></v-text-field>
                        </template>
                        <template v-else>
                            <v-text-field
                                v-model.number="writeDialog.valueNum"
                                type="number"
                                step="0.01"
                                label="新数值"
                                variant="outlined"
                                density="comfortable"
                                prepend-inner-icon="mdi-cog"
                                placeholder="请输入要写入的数值"
                                autofocus
                            ></v-text-field>
                        </template>
                        
                        <!-- BACnet Priority Selection -->
                        <v-select
                            v-if="channelProtocol === 'bacnet-ip'"
                            v-model.number="writeDialog.priority"
                            :items="[
                                { title: '1 (最高)', value: 1 },
                                { title: '8 (手动)', value: 8 },
                                { title: '16 (最低)', value: 16 },
                                { title: 'NULL (释放)', value: null }
                            ]"
                            item-title="title"
                            item-value="value"
                            label="BACnet 写入优先级 (1-16)"
                            variant="outlined"
                            density="compact"
                            prepend-inner-icon="mdi-order-numeric-ascending"
                            class="mt-2"
                            hint="优先级 1-16, NULL 表示释放该点位"
                            persistent-hint
                        ></v-select>
                    </v-form>
                </v-card-text>
                <v-card-actions class="pa-4 pt-0">
                    <v-spacer></v-spacer>
                    <v-btn color="grey-darken-1" variant="text" @click="writeDialog.visible = false">取消</v-btn>
                    <v-btn color="primary" variant="elevated" @click="submitWrite" :loading="writeDialog.loading">确认写入</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <!-- Value Detail Dialog -->
        <v-dialog v-model="valueDialog.visible" max-width="700">
            <v-card>
                <v-card-title class="text-h5 bg-primary text-white">
                    完整数值
                </v-card-title>
                <v-card-text class="pt-4">
                    <v-textarea
                        label="原始值"
                        v-model="valueDialog.value"
                        readonly
                        auto-grow
                        rows="3"
                        variant="outlined"
                        class="mb-4"
                    ></v-textarea>

                    <div v-if="valueDialog.isBase64">
                        <div class="text-subtitle-1 mb-2 font-weight-bold">Base64 解码</div>
                        <v-btn-toggle v-model="valueDialog.decodeType" color="primary" mandatory class="mb-4" @update:model-value="tryDecode">
                            <v-btn value="text">Text (UTF-8)</v-btn>
                            <v-btn value="hex">Hex</v-btn>
                            <v-btn value="json">JSON</v-btn>
                        </v-btn-toggle>
                        
                        <v-textarea
                            label="解码结果"
                            v-model="valueDialog.decodedValue"
                            readonly
                            auto-grow
                            rows="5"
                            variant="outlined"
                            style="font-family: monospace;"
                        ></v-textarea>
                    </div>
                    <div v-else-if="numericFormats">
                        <div class="text-subtitle-1 mb-3 font-weight-bold">数值格式转换</div>
                        <v-row class="mb-3" align="center">
                            <v-col cols="12" sm="6">
                                <div class="text-body-2 text-grey-darken-1 mb-1">字节数</div>
                                <v-btn-toggle
                                    v-model="valueDialog.byteLength"
                                    color="primary"
                                    density="compact"
                                    divided
                                >
                                    <v-btn :value="1" size="small">1 字节</v-btn>
                                    <v-btn :value="2" size="small">2 字节</v-btn>
                                    <v-btn :value="4" size="small">4 字节</v-btn>
                                    <v-btn :value="8" size="small">8 字节</v-btn>
                                </v-btn-toggle>
                            </v-col>
                            <v-col cols="12" sm="6" v-if="valueWordOrderOptions.length">
                                <div class="text-body-2 text-grey-darken-1 mb-1">字节 / 字序</div>
                                <v-select
                                    v-model="valueDialog.wordOrder"
                                    :items="valueWordOrderOptions"
                                    item-title="label"
                                    item-value="value"
                                    density="compact"
                                    variant="outlined"
                                    hide-details
                                ></v-select>
                            </v-col>
                        </v-row>
                        <v-table density="compact">
                            <tbody>
                                <tr>
                                    <td class="font-weight-medium" style="width: 140px;">有符号整型</td>
                                    <td style="font-family: monospace;">{{ numericFormats.signed }}</td>
                                </tr>
                                <tr>
                                    <td class="font-weight-medium">无符号整型</td>
                                    <td style="font-family: monospace;">{{ numericFormats.unsigned }}</td>
                                </tr>
                                <tr>
                                    <td class="font-weight-medium">十六进制</td>
                                    <td style="font-family: monospace;">{{ numericFormats.hex }}</td>
                                </tr>
                                <tr>
                                    <td class="font-weight-medium">二进制</td>
                                    <td style="font-family: monospace; word-break: break-all;">
                                        {{ numericFormats.binary }}
                                    </td>
                                </tr>
                            </tbody>
                        </v-table>
                    </div>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn color="primary" @click="valueDialog.visible = false">关闭</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { globalState, showMessage } from '../composables/useGlobalState'
import request from '@/utils/request'
import basePointTemplates from '@/utils/pointTemplates.json'
import { sanitizeHtml } from '@/utils/sanitizeHtml'
import {
    baseWordOrderOptions,
    baseParseTypeOptions,
    getWordOrderOptionsForBytes,
    filterParseTypesByBytes,
    wordOrderToBackend,
    reorderBytes,
    parseByType,
    applyFormula,
    registersToBytes
} from '@/utils/pointDecodeHelper'

const { locale } = useI18n()
const currentLang = computed(() => (locale.value || 'zh').toString())

const route = useRoute()
const points = ref([])
const deviceInfo = ref(null)
const loading = ref(false)

const channelId = computed(() => route.params.channelId)
const deviceId = computed(() => route.params.deviceId)

// Watch for route changes to refresh data
watch([channelId, deviceId], () => {
    fetchPoints()
    fetchChannel()
    fetchMetrics()
})

// Point Filtering & Selection
const filters = reactive({
    search: '',
    quality: []
})

const selection = reactive({
    selectedIds: [],
    selectAll: false
})

const filteredPoints = computed(() => {
    let result = points.value || []
    
    // Search filter
    if (filters.search) {
        const s = filters.search.toLowerCase()
        result = result.filter(p => 
            (p.id && p.id.toLowerCase().includes(s)) ||
            (p.name && p.name.toLowerCase().includes(s)) ||
            (p.address && String(p.address).toLowerCase().includes(s))
        )
    }
    
    // Quality filter
    if (filters.quality && filters.quality.length > 0) {
        result = result.filter(p => filters.quality.includes(p.quality))
    }
    
    return result
})

const toggleSelectAll = (val) => {
    if (val) {
        selection.selectedIds = filteredPoints.value.map(p => p.id)
    } else {
        selection.selectedIds = []
    }
}

// Watch filteredPoints to update selectAll state if points change
watch(filteredPoints, (newPoints) => {
    if (newPoints.length === 0) {
        selection.selectAll = false
        selection.selectedIds = []
    } else {
        // Update selection to only include visible points
        selection.selectedIds = selection.selectedIds.filter(id => 
            newPoints.some(p => p.id === id)
        )
        selection.selectAll = newPoints.length > 0 && selection.selectedIds.length === newPoints.length
    }
}, { deep: true })

// Watch selectedIds to update selectAll state
watch(() => selection.selectedIds, (newIds) => {
    if (filteredPoints.value.length === 0) {
        selection.selectAll = false
    } else {
        selection.selectAll = newIds.length === filteredPoints.value.length
    }
}, { deep: true })

const confirmBatchDelete = () => {
    if (selection.selectedIds.length === 0) return
    
    deleteDialog.isBatch = true
    deleteDialog.batchCount = selection.selectedIds.length
    deleteDialog.visible = true
}

// Write Dialog State
const writeDialog = reactive({
    visible: false,
    deviceID: '',
    pointID: '',
    dataType: '',
    valueNum: 0,
    valueStr: '',
    valueBool: false,
    priority: 16, // Default priority
    loading: false
})

// Point Dialog State (Add/Edit)
// Load register offset from localStorage if it exists
const loadRegisterOffset = () => {
    try {
        const savedOffset = localStorage.getItem('modbus_register_offset')
        return savedOffset ? parseInt(savedOffset) : 0
    } catch (e) {
        console.error('Error loading register offset from localStorage:', e)
        return 0
    }
}

// Save register offset to localStorage
const saveRegisterOffset = (offset) => {
    try {
        localStorage.setItem('modbus_register_offset', offset.toString())
    } catch (e) {
        console.error('Error saving register offset to localStorage:', e)
    }
}

const pointDialog = reactive({
    visible: false,
    isEdit: false,
    loading: false,
    registerType: 'holding',
    registerIndex: 1,
    functionCode: 3,
    bacnetType: 'AnalogInput',
    bacnetInstance: 1,
    dlt645DeviceAddr: '',
    dlt645DataID: '',
    byteLength: 4,
    wordOrderOption: 'ABCD',
    parseType: 'FLOAT32',
    defaultValue: '',
    registerOffset: loadRegisterOffset(),
    form: {
        id: '',
        name: '',
        address: '',
        format: '',
        datatype: 'float32',
        readwrite: 'R',
        unit: '',
        scale: 1.0,
        offset: 0.0,
        read_formula: '',
        write_formula: '',
        read_formula_template: null,
        write_formula_template: null
    }
})

const datatypeOptions = [
    'int16',
    'uint16',
    'int32',
    'uint32',
    'float32',
    'float64',
    'bool',
    'string',
    'WORD',
    'DWORD',
    'LWORD'
]

const formatPresets = [
    {
        id: 'Signed',
        label: 'Signed (2 字节 / 1 寄存器)',
        bytes: 2,
        parseType: 'INT16',
        wordOrder: 'AB',
        datatype: 'int16'
    },
    {
        id: 'Unsigned',
        label: 'Unsigned (2 字节 / 1 寄存器)',
        bytes: 2,
        parseType: 'UINT16',
        wordOrder: 'AB',
        datatype: 'uint16'
    },
    {
        id: 'Hex',
        label: 'Hex (2 字节 / 1 寄存器)',
        bytes: 2,
        parseType: 'UINT16',
        wordOrder: 'AB',
        datatype: 'uint16'
    },
    {
        id: 'Binary',
        label: 'Binary (2 字节 / 1 寄存器)',
        bytes: 2,
        parseType: 'UINT16',
        wordOrder: 'AB',
        datatype: 'uint16'
    },
    {
        id: 'LongABCD',
        label: 'LongABCD (4 字节 / 2 寄存器)',
        bytes: 4,
        parseType: 'INT32',
        wordOrder: 'ABCD',
        datatype: 'int32'
    },
    {
        id: 'LongCDAB',
        label: 'LongCDAB (4 字节 / 2 寄存器)',
        bytes: 4,
        parseType: 'INT32',
        wordOrder: 'CDAB',
        datatype: 'int32'
    },
    {
        id: 'LongBADC',
        label: 'LongBADC (4 字节 / 2 寄存器)',
        bytes: 4,
        parseType: 'INT32',
        wordOrder: 'BADC',
        datatype: 'int32'
    },
    {
        id: 'LongDCBA',
        label: 'LongDCBA (4 字节 / 2 寄存器)',
        bytes: 4,
        parseType: 'INT32',
        wordOrder: 'DCBA',
        datatype: 'int32'
    },
    {
        id: 'FloatABCD',
        label: 'FloatABCD (4 字节 / 2 寄存器)',
        bytes: 4,
        parseType: 'FLOAT32',
        wordOrder: 'ABCD',
        datatype: 'float32'
    },
    {
        id: 'FloatCDAB',
        label: 'FloatCDAB (4 字节 / 2 寄存器)',
        bytes: 4,
        parseType: 'FLOAT32',
        wordOrder: 'CDAB',
        datatype: 'float32'
    },
    {
        id: 'FloatBADC',
        label: 'FloatBADC (4 字节 / 2 寄存器)',
        bytes: 4,
        parseType: 'FLOAT32',
        wordOrder: 'BADC',
        datatype: 'float32'
    },
    {
        id: 'FloatDCBA',
        label: 'FloatDCBA (4 字节 / 2 寄存器)',
        bytes: 4,
        parseType: 'FLOAT32',
        wordOrder: 'DCBA',
        datatype: 'float32'
    },
    {
        id: 'DoubleABCDEFGH',
        label: 'DoubleABCDEFGH (8 字节 / 4 寄存器)',
        bytes: 8,
        parseType: 'FLOAT64',
        wordOrder: 'ABCD',
        datatype: 'float64'
    },
    {
        id: 'DoubleGHEFCDAB',
        label: 'DoubleGHEFCDAB (8 字节 / 4 寄存器)',
        bytes: 8,
        parseType: 'FLOAT64',
        wordOrder: 'CDAB',
        datatype: 'float64'
    },
    {
        id: 'DoubleBADCFEHG',
        label: 'DoubleBADCFEHG (8 字节 / 4 寄存器)',
        bytes: 8,
        parseType: 'FLOAT64',
        wordOrder: 'BADC',
        datatype: 'float64'
    },
    {
        id: 'DoubleHGFEDCBA',
        label: 'DoubleHGFEDCBA (8 字节 / 4 寄存器)',
        bytes: 8,
        parseType: 'FLOAT64',
        wordOrder: 'DCBA',
        datatype: 'float64'
    }
]

const formatPresetSelected = ref(null)
const recentFormatIds = ref([])

const wordOrderOptions = baseWordOrderOptions
const parseTypeOptions = baseParseTypeOptions

const wordOrderOptionsForBytes = computed(() => getWordOrderOptionsForBytes(pointDialog.byteLength))

const filteredParseTypes = computed(() => filterParseTypesByBytes(pointDialog.byteLength))

const filteredFormatPresets = computed(() => formatPresets)

const quickValidate = reactive({
    visible: false,
    rawHex: '',
    expected: '',
    previewHtml: '',
    status: '',
    registerValues: '',
    registerBaseAddress: ''
})

const templateDialog = reactive({
    visible: false,
    search: '',
    templates: [...basePointTemplates],
    runtimeTemplates: []
})

const allPointTemplates = computed(() => {
    return [...templateDialog.templates, ...templateDialog.runtimeTemplates]
})

const filteredPointTemplates = computed(() => {
    const proto = channelProtocol.value
    const key = (templateDialog.search || '').trim().toLowerCase()
    return allPointTemplates.value.filter(tpl => {
        if (tpl.protocol && tpl.protocol !== proto) {
            return false
        }
        if (!key) return true
        const text = `${tpl.name || ''} ${tpl.description || ''}`.toLowerCase()
        return text.includes(key)
    })
})

const formulaTemplates = [
    { label: '线性缩放: v * 0.1', expr: 'v * 0.1' },
    { label: '线性缩放: v / 10', expr: 'v / 10' },
    { label: '温度转换: 摄氏转华氏 (v * 1.8 + 32)', expr: 'v * 1.8 + 32' },
    { label: '位运算: 取第0位 (bit0)', expr: 'bitand(v,1) != 0' },
    { label: '位运算: 右移2位 (v >> 2)', expr: 'v >> 2' },
    { label: '高低字交换: 16位 (v >> 8 | (v & 0xFF) << 8)', expr: '(v >> 8) | ((v & 255) << 8)' }
]

const formulaErrors = reactive({
    read: '',
    write: ''
})

const helpDialog = reactive({
    visible: false,
    search: ''
})

const helpSections = [
    {
        id: 'protocol',
        title: '协议解码示例',
        items: [
            {
                title: 'Modbus 电压值 (寄存器值 * 0.1)',
                desc: '寄存器保存的为整数 0.1V 步进的电压值，例如 2301 表示 230.1V，可通过读公式 v * 0.1 得到工程值。',
                snippet: '读公式: v * 0.1'
            },
            {
                title: 'BACnet 多状态量 (右移去掉状态位)',
                desc: '某些多状态点高位为状态标志，低位为实际值，可以通过读公式 v >> 2 提取实际值。',
                snippet: '读公式: v >> 2'
            }
        ]
    },
    {
        id: 'syntax',
        title: '公式语法说明',
        items: [
            {
                title: '基本运算符',
                desc: '支持 +, -, *, /, %, 括号 ()，以及按位运算符 & | ^ << >>。变量名统一为 v。',
                snippet: '示例: (v - 4) * 0.5'
            },
            {
                title: '比较与三元运算',
                desc: '支持 >, >=, <, <=, ==, != 以及条件表达式 condition ? a : b，用于告警或状态映射。',
                snippet: '示例: v > 0 ? 1 : 0'
            }
        ]
    },
    {
        id: 'functions',
        title: '常用函数库',
        items: [
            {
                title: 'bitand / bitor / bitxor',
                desc: '位与 / 位或 / 位异或，用于从状态字中提取和组合标志位。',
                snippet: '示例: bitand(v, 4) != 0'
            },
            {
                title: 'bitnot / bitshl / bitshr',
                desc: '按位取反、左移、右移，对寄存器位进行操作。',
                snippet: '示例: bitshr(v, 1)'
            }
        ]
    },
    {
        id: 'faq',
        title: 'FAQ 与实践建议',
        items: [
            {
                title: '读公式与写公式如何配对',
                desc: '通常读公式是从寄存器值到工程值，写公式则是工程值到寄存器值，例如读: v * 0.1，对应写: v / 0.1。',
                snippet: '读: v * 0.1\n写: v / 0.1'
            },
            {
                title: '公式与缩放比例如何选择',
                desc: '建议优先使用公式描述复杂逻辑，Scale/Offset 只做简单线性换算，避免同一含义重复配置。',
                snippet: ''
            }
        ]
    }
]

const filteredHelpSections = computed(() => {
    const key = (helpDialog.search || '').trim().toLowerCase()
    if (!key) return helpSections
    return helpSections
        .map(section => {
            const items = section.items.filter(item => {
                const text = `${item.title} ${item.desc} ${item.snippet || ''}`.toLowerCase()
                return text.includes(key)
            })
            return { ...section, items }
        })
        .filter(section => section.items.length > 0)
})

const copySnippet = (text) => {
    if (!text) return
    navigator.clipboard.writeText(text)
        .then(() => {
            showMessage('示例已复制到剪贴板', 'success')
        })
        .catch(() => {
            showMessage('复制失败，请手动选择文本', 'warning')
        })
}

const openQuickValidate = () => {
    quickValidate.visible = true
    quickValidate.status = ''
    quickValidate.previewHtml = ''
    quickValidate.rawHex = ''
    quickValidate.registerValues = ''
    quickValidate.expected = ''
    const addr = pointDialog.form.address
    quickValidate.registerBaseAddress = typeof addr === 'string' ? addr : String(addr || '')
}

const runQuickValidate = () => {
    let hex = (quickValidate.rawHex || '').replace(/[^0-9a-fA-F]/g, '')
    if (!hex) {
        const text = (quickValidate.registerValues || '').trim()
        if (text) {
            const parts = text.split(/[\s,]+/).filter(Boolean)
            const regs = parts.map(p => {
                if (p.toLowerCase().startsWith('0x')) {
                    return parseInt(p, 16)
                }
                return parseInt(p, 10)
            })
            const bytes = registersToBytes(regs)
            hex = bytes.map(b => b.toString(16).padStart(2, '0')).join('')
        }
    }
    if (!hex) {
        quickValidate.previewHtml = sanitizeHtml('请输入有效的十六进制报文')
        quickValidate.status = ''
        return
    }
    const bytesNeeded = pointDialog.byteLength || 0
    if (bytesNeeded > 0 && hex.length < bytesNeeded * 2) {
        quickValidate.previewHtml = sanitizeHtml('报文长度不足，无法解析')
        quickValidate.status = ''
        return
    }
    const buf = []
    for (let i = 0; i < hex.length; i += 2) {
        const b = parseInt(hex.slice(i, i + 2), 16)
        if (isNaN(b)) continue
        buf.push(b)
    }
    const slice = bytesNeeded > 0 ? buf.slice(0, bytesNeeded) : buf
    const reordered = reorderBytes(slice, pointDialog.byteLength, pointDialog.wordOrderOption)
    const rawValue = parseByType(reordered, pointDialog.parseType)
    const engineValue = applyFormula(rawValue, pointDialog.form.read_formula, pointDialog.form.scale, pointDialog.form.offset)
    const valueStr = engineValue === undefined ? '解析失败' : String(engineValue)
    const expectedInput = (quickValidate.expected || '').trim()
    let expected = expectedInput
    if (expectedInput) {
        const parts = expectedInput.split(/\s+/)
        expected = parts[parts.length - 1]
    }
    if (expected) {
        const actualNum = Number(engineValue)
        const expectedNum = Number(expected)
        let pass = false
        if (Number.isFinite(actualNum) && Number.isFinite(expectedNum)) {
            pass = Math.abs(actualNum - expectedNum) < 1e-6
        } else {
            pass = String(engineValue) === expected
        }
        quickValidate.status = pass ? 'pass' : 'fail'
        const diffHtml = highlightDiff(valueStr, expected)
        const html = `实际: ${valueStr}<br/>期望: ${expected}<br/>差异: ${diffHtml}`
        quickValidate.previewHtml = sanitizeHtml(html)
    } else {
        quickValidate.status = ''
        quickValidate.previewHtml = sanitizeHtml(valueStr)
    }
}

const highlightDiff = (actual, expected) => {
    const a = String(actual)
    const b = String(expected)
    const len = Math.max(a.length, b.length)
    let html = ''
    for (let i = 0; i < len; i++) {
        const ca = a[i] || ''
        const cb = b[i] || ''
        if (ca === cb) {
            html += ca
        } else {
            html += `<span style="color: red; font-weight: 600">${ca || ' '}</span>`
        }
    }
    return html
}

const updateRecentFormats = (id) => {
    if (!id) {
        return
    }
    const list = recentFormatIds.value.filter(x => x !== id)
    list.unshift(id)
    recentFormatIds.value = list.slice(0, 2)
}

const presetIdToFormat = (id) => {
    if (!id) return ''
    if (id === 'Hex') return 'hex'
    if (id === 'Binary') return 'binary'
    return id
}

const inferPresetFromPoint = (p) => {
    if (!p) return null
    const dt = (p.datatype || '').toLowerCase()
    const fmt = (p.format || '').toLowerCase()
    const wo = (p.word_order || '').toUpperCase()

    if (fmt === 'hex') return 'Hex'
    if (fmt === 'binary') return 'Binary'

    if (dt === 'int16') return 'Signed'
    if (dt === 'uint16') return 'Unsigned'

    if (dt === 'int32') {
        if (wo === 'CDAB') return 'LongCDAB'
        if (wo === 'BADC') return 'LongBADC'
        if (wo === 'DCBA') return 'LongDCBA'
        return 'LongABCD'
    }

    if (dt === 'float32') {
        if (wo === 'CDAB') return 'FloatCDAB'
        if (wo === 'BADC') return 'FloatBADC'
        if (wo === 'DCBA') return 'FloatDCBA'
        return 'FloatABCD'
    }

    if (dt === 'float64') {
        if (wo === 'CDAB') return 'DoubleGHEFCDAB'
        if (wo === 'BADC') return 'DoubleBADCFEHG'
        if (wo === 'DCBA') return 'DoubleHGFEDCBA'
        return 'DoubleABCDEFGH'
    }

    return null
}

const onSelectFormatPreset = (id) => {
    if (!id) {
        return
    }
    const preset = formatPresets.find(p => p.id === id)
    if (!preset) {
        return
    }
    pointDialog.byteLength = preset.bytes
    pointDialog.wordOrderOption = preset.wordOrder
    pointDialog.parseType = preset.parseType
    pointDialog.form.datatype = preset.datatype
    pointDialog.form.format = presetIdToFormat(id)
    updateRecentFormats(id)
}

const toggleRecentFormats = () => {
    if (recentFormatIds.value.length === 0) {
        return
    }
    let target = recentFormatIds.value[0]
    if (formatPresetSelected.value === recentFormatIds.value[0] && recentFormatIds.value[1]) {
        target = recentFormatIds.value[1]
    }
    formatPresetSelected.value = target
    onSelectFormatPreset(target)
}

const openTemplateDialog = () => {
    templateDialog.visible = true
}

const applyTemplate = (tpl) => {
    if (!tpl) return
    const name = tpl.name || ''
    pointDialog.form.name = name
    pointDialog.form.datatype = tpl.datatype || pointDialog.form.datatype
    pointDialog.form.unit = tpl.unit || ''
    pointDialog.form.readwrite = tpl.readwrite || 'R'
    pointDialog.form.read_formula = tpl.readFormula || ''
    pointDialog.form.write_formula = tpl.writeFormula || ''
    pointDialog.byteLength = tpl.byteLength || pointDialog.byteLength
    pointDialog.wordOrderOption = tpl.wordOrder || pointDialog.wordOrderOption
    pointDialog.parseType = tpl.parseType || pointDialog.parseType
    pointDialog.defaultValue = tpl.defaultValue !== undefined ? String(tpl.defaultValue) : ''
    pointDialog.form.word_order = wordOrderToBackend(pointDialog.wordOrderOption)
    validateFormula('read')
    validateFormula('write')
    templateDialog.visible = false
}

const copyTemplate = (tpl) => {
    if (!tpl) return
    const text = JSON.stringify(tpl, null, 2)
    navigator.clipboard.writeText(text)
        .then(() => {
            showMessage('模板配置已复制到剪贴板', 'success')
        })
        .catch(() => {
            showMessage('复制失败，请手动选择文本', 'warning')
        })
}

const saveCurrentAsTemplate = () => {
    if (quickValidate.status !== 'pass') return
    const tpl = {
        id: `custom_${Date.now()}`,
        protocol: channelProtocol.value,
        name: pointDialog.form.name || '自定义模板',
        datatype: pointDialog.form.datatype,
        byteLength: pointDialog.byteLength,
        wordOrder: pointDialog.wordOrderOption,
        parseType: pointDialog.parseType,
        unit: pointDialog.form.unit,
        readFormula: pointDialog.form.read_formula,
        writeFormula: pointDialog.form.write_formula,
        defaultValue: pointDialog.defaultValue,
        readwrite: pointDialog.form.readwrite,
        description: '通过快速验证生成的自定义模板'
    }
    templateDialog.runtimeTemplates.push(tpl)
    showMessage('当前配置已保存为本地模板(会话内有效)', 'success')
}

const validateFormula = (type) => {
    const val = type === 'read' ? (pointDialog.form.read_formula || '') : (pointDialog.form.write_formula || '')
    const target = type === 'read' ? 'read' : 'write'
    if (!val) {
        formulaErrors[target] = ''
        return
    }
    const allowed = /^[0-9vV\+\-\*\/%\&\|\^\<\>\(\)\?\:\s\.]+$/
    if (!allowed.test(val)) {
        formulaErrors[target] = '仅允许使用数字、v、运算符和括号'
        return
    }
    let balance = 0
    for (let i = 0; i < val.length; i++) {
        if (val[i] === '(') balance++
        if (val[i] === ')') balance--
        if (balance < 0) break
    }
    if (balance !== 0) {
        formulaErrors[target] = '括号不匹配'
        return
    }
    formulaErrors[target] = ''
}

const onSelectFormulaTemplate = (type) => {
    if (type === 'read') {
        const tpl = formulaTemplates.find(t => t.expr === pointDialog.form.read_formula_template)
        if (tpl) {
            pointDialog.form.read_formula = tpl.expr
            validateFormula('read')
        }
    } else {
        const tpl = formulaTemplates.find(t => t.expr === pointDialog.form.write_formula_template)
        if (tpl) {
            pointDialog.form.write_formula = tpl.expr
            validateFormula('write')
        }
    }
}

const registerTypes = [
    { title: 'Coils (outputs) - 01', value: 'coil' },
    { title: 'Discrete Inputs - 02', value: 'discrete' },
    { title: 'Input Registers - 04', value: 'input' },
    { title: 'Holding Registers - 03', value: 'holding' }
]

const registerIndexError = ref('')
const registerOffsetError = ref('')

const getRegisterIndexMin = () => {
    // Get start_address from device config
    const startAddress = deviceInfo.value?.config?.start_address || deviceInfo.value?.config?.address_base || 0
    return startAddress
}

const getRegisterIndexMax = () => {
    // Get start_address from device config
    const startAddress = deviceInfo.value?.config?.start_address || deviceInfo.value?.config?.address_base || 0
    // PDU address max is 65535, so UI display address max is start_address + 65535
    return startAddress + 65535
}

const validateRegisterIndex = () => {
    const idx = parseInt(pointDialog.registerIndex) || 0
    const min = getRegisterIndexMin()
    const max = getRegisterIndexMax()
    
    if (idx < min || idx > max) {
        registerIndexError.value = `寄存器索引必须在 ${min} 到 ${max} 之间`
    } else {
        registerIndexError.value = ''
    }
}

const validateRegisterOffset = () => {
    const offset = parseInt(pointDialog.registerOffset) || 0
    if (offset < 0 || offset > 9999) {
        registerOffsetError.value = '起始偏移量必须在 0 到 9999 之间'
    } else {
        registerOffsetError.value = ''
        saveRegisterOffset(offset)
    }
}

const updateAddress = () => {
    const idx = parseInt(pointDialog.registerIndex) || 0
    const offset = parseInt(pointDialog.registerOffset) || 0
    let address = 0
    
    // Get start_address from device config
    const startAddress = deviceInfo.value?.config?.start_address || deviceInfo.value?.config?.address_base || 0
    
    // Validate input address
    if (idx < startAddress) {
        registerIndexError.value = `地址不能小于基准地址 ${startAddress}`
        return
    }
    
    // Calculate PDU 0-based address
    address = idx - startAddress + offset
    
    // Validate PDU address range
    if (address < 0 || address > 65535) {
        registerIndexError.value = 'PDU地址必须在 0 到 65535 之间'
        return
    }
    
    registerIndexError.value = ''
    pointDialog.form.address = address.toString()
}

const updateBACnetAddress = () => {
	pointDialog.form.address = `${pointDialog.bacnetType}:${pointDialog.bacnetInstance}`
}

const updateDLT645Address = () => {
    if (pointDialog.dlt645DeviceAddr && pointDialog.dlt645DataID) {
        pointDialog.form.address = `${pointDialog.dlt645DeviceAddr}#${pointDialog.dlt645DataID}`
    } else {
        pointDialog.form.address = ''
    }
}

const parseAddressToUI = (addrStr) => {
	if (channelProtocol.value.startsWith('modbus')) {
		const addr = parseInt(addrStr)
		if (isNaN(addr)) return

		// Get start_address from device config
		const startAddress = deviceInfo.value?.config?.start_address || deviceInfo.value?.config?.address_base || 0
		
		// Calculate UI display address
		const displayAddress = addr + startAddress
		
		// Set register type and function code
		pointDialog.registerType = 'holding'
		pointDialog.registerIndex = displayAddress
		pointDialog.functionCode = 3
	} else if (channelProtocol.value === 'bacnet-ip') {
		const parts = addrStr.split(':')
		if (parts.length === 2) {
			pointDialog.bacnetType = parts[0]
			pointDialog.bacnetInstance = parseInt(parts[1]) || 0
		}
	} else if (channelProtocol.value === 'dlt645') {
        const parts = addrStr.split('#')
        if (parts.length === 2) {
            pointDialog.dlt645DeviceAddr = parts[0]
            pointDialog.dlt645DataID = parts[1]
        }
    }
}

// Delete Dialog State
const deleteDialog = reactive({
	visible: false,
	point: null,
	loading: false,
    isBatch: false,
    batchCount: 0
})

const openAddDialog = () => {
	pointDialog.isEdit = false
	pointDialog.form = {
		id: '',
		name: '',
		address: '',
        format: '',
		datatype: 'float32',
		readwrite: 'R',
		unit: '',
		scale: 1.0,
		offset: 0.0,
        read_formula: '',
        write_formula: '',
        read_formula_template: null,
        write_formula_template: null
	}
	
	// Defaults
	pointDialog.registerType = 'holding'
	pointDialog.registerIndex = 1
	pointDialog.functionCode = 3
	pointDialog.bacnetType = 'AnalogInput'
	pointDialog.bacnetInstance = 1
    pointDialog.dlt645DeviceAddr = ''
    pointDialog.dlt645DataID = ''
    pointDialog.byteLength = 4
    pointDialog.wordOrderOption = 'ABCD'
    pointDialog.parseType = 'FLOAT32'
    pointDialog.defaultValue = ''
	
	if (channelProtocol.value.startsWith('modbus')) {
		// Get start_address from device config
		const startAddress = deviceInfo.value?.config?.start_address || deviceInfo.value?.config?.address_base || 0
		// Set default address based on start_address
		pointDialog.form.address = startAddress.toString()
		// Set register index based on start_address
		pointDialog.registerIndex = startAddress
	} else if (channelProtocol.value === 'bacnet-ip') {
		pointDialog.form.address = 'AnalogInput:1'
	} else if (channelProtocol.value === 'dlt645') {
        if (deviceInfo.value && deviceInfo.value.config) {
            const addr = deviceInfo.value.config.station_address || deviceInfo.value.config.address || ''
            if (addr) {
                pointDialog.dlt645DeviceAddr = addr
                // Pre-fill a common data ID or leave empty
                pointDialog.dlt645DataID = '02-01-01-00' // Voltage A
                updateDLT645Address()
            }
        }
    }
	
	pointDialog.visible = true
}

const openEditDialog = (point) => {
    pointDialog.isEdit = true
    
    // Try to find full config from deviceInfo if available
    if (deviceInfo.value && deviceInfo.value.points) {
        const fullPoint = deviceInfo.value.points.find(p => p.id === point.id)
        if (fullPoint) {
            pointDialog.form = { ...fullPoint }
            if (pointDialog.form.scale === undefined) pointDialog.form.scale = 1.0
            if (pointDialog.form.offset === undefined) pointDialog.form.offset = 0.0
            if (pointDialog.form.read_formula === undefined) pointDialog.form.read_formula = ''
            if (pointDialog.form.write_formula === undefined) pointDialog.form.write_formula = ''
        } else {
            pointDialog.form = {
                ...point,
                scale: 1.0,
                offset: 0.0,
                read_formula: '',
                write_formula: '',
                read_formula_template: null,
                write_formula_template: null
            }
        }
    } else {
        pointDialog.form = {
            ...point,
            scale: 1.0,
            offset: 0.0,
            read_formula: '',
            write_formula: '',
            read_formula_template: null,
            write_formula_template: null
        }
    }
    
    if (pointDialog.form.address) {
        parseAddressToUI(pointDialog.form.address)
    }
    
    // 加载register_type和function_code
    if (pointDialog.form.register_type) {
        pointDialog.registerType = pointDialog.form.register_type
    }
    if (pointDialog.form.function_code) {
        pointDialog.functionCode = pointDialog.form.function_code
    } else {
        // 根据registerType设置默认functionCode
        const typeToCode = { 'coil': 1, 'discrete': 2, 'input': 4, 'holding': 3 }
        pointDialog.functionCode = typeToCode[pointDialog.registerType] || 3
    }
    
    const dt = (pointDialog.form.datatype || '').toLowerCase()
    if (dt === 'int16' || dt === 'uint16' || dt === 'bool' || dt === 'word') {
        pointDialog.byteLength = 2
    } else if (dt === 'int32' || dt === 'uint32' || dt === 'float32' || dt === 'dword' || dt === 'lword') {
        pointDialog.byteLength = 4
    } else if (dt === 'float64') {
        pointDialog.byteLength = 8
    }
    const wo = (pointDialog.form.word_order || '').toUpperCase()
    if (pointDialog.byteLength === 2) {
        if (wo === 'DCBA') {
            pointDialog.wordOrderOption = 'BA'
        } else {
            pointDialog.wordOrderOption = 'AB'
        }
    } else if (wo) {
        pointDialog.wordOrderOption = wo
    }

    const presetId = inferPresetFromPoint(pointDialog.form)
    formatPresetSelected.value = presetId

    pointDialog.visible = true
}

const submitPoint = async () => {
    pointDialog.loading = true
    try {
        const url = pointDialog.isEdit 
            ? `/api/channels/${channelId.value}/devices/${deviceId.value}/points/${pointDialog.form.id}`
            : `/api/channels/${channelId.value}/devices/${deviceId.value}/points`
        if (formatPresetSelected.value) {
            pointDialog.form.format = presetIdToFormat(formatPresetSelected.value)
        }
        pointDialog.form.word_order = wordOrderToBackend(pointDialog.wordOrderOption)
        
        // 添加寄存器类型和功能码
        pointDialog.form.register_type = pointDialog.registerType
        if (pointDialog.functionCode && pointDialog.functionCode !== 0) {
            pointDialog.form.function_code = pointDialog.functionCode
        }
        
        if (pointDialog.isEdit) {
            await request.put(url, pointDialog.form)
        } else {
            await request.post(url, pointDialog.form)
        }

        showMessage(pointDialog.isEdit ? '点位更新成功' : '点位添加成功', 'success')
        pointDialog.visible = false
        fetchPoints() // Refresh list
    } catch (e) {
        showMessage(e.message, 'error')
    } finally {
        pointDialog.loading = false
    }
}

const confirmDelete = (point) => {
    deleteDialog.isBatch = false
    deleteDialog.point = point
    deleteDialog.visible = true
}

const executeDelete = async () => {
    deleteDialog.loading = true
    try {
        if (deleteDialog.isBatch) {
            // Batch delete
            await request.delete(`/api/channels/${channelId.value}/devices/${deviceId.value}/points`, { data: selection.selectedIds })
            showMessage(`成功删除 ${selection.selectedIds.length} 个点位`, 'success')
            selection.selectedIds = []
        } else {
            // Single delete
            if (!deleteDialog.point) return
            await request.delete(`/api/channels/${channelId.value}/devices/${deviceId.value}/points/${deleteDialog.point.id}`)
            showMessage('点位删除成功', 'success')
        }

        deleteDialog.visible = false
        fetchPoints() // Refresh list
    } catch (e) {
        showMessage(e.message, 'error')
    } finally {
        deleteDialog.loading = false
    }
}

const cloneDialog = reactive({
    visible: false,
    loading: false,
    channels: [],
    selectedChannel: null,
    devices: [],
    selectedDevice: null,
    points: [],
    selected: [],
    selectAll: false,
    search: ''
})

const openCloneDialog = async () => {
    cloneDialog.visible = true
    cloneDialog.loading = true
    cloneDialog.channels = []
    cloneDialog.devices = []
    cloneDialog.points = []
    cloneDialog.selected = []
    cloneDialog.selectAll = false
    try {
        const chs = await request.get('/api/channels', { timeout: 10000, silent: true })
        const same = (chs || []).filter(ch => ch.protocol === channelProtocol.value)
        cloneDialog.channels = same
        if (same.length === 1) {
            cloneDialog.selectedChannel = same[0].id
            await onCloneChannelChange(same[0].id)
        }
    } catch (e) {
    } finally {
        cloneDialog.loading = false
    }
}

const onCloneChannelChange = async (cid) => {
    cloneDialog.loading = true
    cloneDialog.devices = []
    cloneDialog.points = []
    cloneDialog.selected = []
    cloneDialog.selectAll = false
    try {
        if (!cid) return
        const devs = await request.get(`/api/channels/${cid}/devices`, { timeout: 10000, silent: true })
        const list = devs || []
        cloneDialog.devices = list.filter(d => !(cid === channelId.value && d.id === deviceId.value))
    } catch (e) {
    } finally {
        cloneDialog.loading = false
    }
}

const onCloneDeviceChange = async (did) => {
    cloneDialog.loading = true
    cloneDialog.points = []
    cloneDialog.selected = []
    cloneDialog.selectAll = false
    try {
        const cid = cloneDialog.selectedChannel
        if (!cid || !did) return
        const pts = await request.get(`/api/channels/${cid}/devices/${did}/points`, { timeout: 8000, silent: true })
        cloneDialog.points = (pts || []).map(p => ({
            id: p.id,
            name: p.name,
            address: p.address,
            datatype: p.datatype,
            unit: p.unit || '',
            readwrite: p.readwrite || 'R'
        }))
    } catch (e) {
    } finally {
        cloneDialog.loading = false
    }
}

const toggleCloneSelectAll = () => {
    if (cloneDialog.selectAll) {
        cloneDialog.selected = [...cloneDialog.points]
    } else {
        cloneDialog.selected = []
    }
}

const filteredClonePoints = computed(() => {
    const list = cloneDialog.points || []
    const key = (cloneDialog.search || '').trim().toLowerCase()
    if (!key) return list
    return list.filter(p => {
        const name = (p.name || '').toLowerCase()
        const addr = (p.address || '').toLowerCase()
        return name.includes(key) || addr.includes(key)
    })
})

const executeClone = async () => {
    if (!cloneDialog.selected || cloneDialog.selected.length === 0) return
    cloneDialog.loading = true
    try {
        const payload = cloneDialog.selected.map(p => ({
            id: p.id,
            name: p.name,
            address: p.address,
            datatype: p.datatype,
            unit: p.unit || '',
            readwrite: p.readwrite || 'R',
            scale: 1.0,
            offset: 0.0,
            read_formula: '',
            write_formula: ''
        }))

        await request.post(`/api/channels/${channelId.value}/devices/${deviceId.value}/points`, payload, { timeout: 10000, silent: true })
        showMessage(`克隆完成：成功 ${payload.length} 个`, 'success')
        cloneDialog.visible = false
        await fetchPoints()
    } finally {
        cloneDialog.loading = false
    }
}

const fetchPoints = async () => {
    console.log('Fetching points for channel:', channelId.value, 'device:', deviceId.value)
    loading.value = true
    try {
        // 1) 优先获取设备信息（包含点位配置），快速首屏渲染
        if (!deviceInfo.value) {
            try {
                console.log('Fetching device info...')
                const dev = await request.get(`/api/channels/${channelId.value}/devices/${deviceId.value}`)
                if (dev) {
                    deviceInfo.value = dev
                    globalState.navTitle = deviceInfo.value.name
                    console.log('Device info fetched:', dev.name, 'Points count:', dev.points?.length)
                }
            } catch (e) {
                console.error('Failed to fetch device info', e)
            }
        }

        // 用设备配置中的点位生成基础列表（无阻塞首屏）
        if (deviceInfo.value && Array.isArray(deviceInfo.value.points)) {
            const now = new Date()
            points.value = deviceInfo.value.points.map(p => ({
                id: p.id,
                name: p.name,
                address: p.address,
                datatype: p.datatype,
                unit: p.unit || '',
                readwrite: p.readwrite || 'R',
                value: null,
                quality: 'Bad',
                timestamp: now
            }))
            console.log('Initial points list created from device info:', points.value.length)
        } else {
            points.value = []
            console.warn('No points found in device info')
        }

        // 2) 合并实时缓存（快速填充值，仅当前设备）
        try {
            console.log('Fetching realtime values...')
            const realtime = await request.get(`/api/values/realtime?channel_id=${channelId.value}&device_id=${deviceId.value}`)
            if (realtime && typeof realtime === 'object') {
                console.log('Realtime values fetched:', Object.keys(realtime).length)
                for (let i = 0; i < points.value.length; i++) {
                    const pid = points.value[i].id
                    const v = realtime[pid]
                    if (v) {
                        points.value[i].value = v.value
                        points.value[i].quality = v.quality || 'Good'
                        if (v.ts) points.value[i].timestamp = v.ts
                    }
                }
            }
        } catch (e) {
            // 实时缓存失败不阻塞 UI
            console.warn('Fetch realtime values failed', e)
        }

        // 3) 后台拉取最新实时值（超时短，不阻塞页面交互）
        // 成功则用返回结果覆盖；失败/超时忽略，等待 WebSocket 或下次刷新
        console.log('Triggering background point fetch...')
        request.get(`/api/channels/${channelId.value}/devices/${deviceId.value}/points`, { timeout: 2500, silent: true })
            .then(pts => {
                if (Array.isArray(pts) && pts.length > 0) {
                    console.log('Background point fetch successful, updating points. Points:', pts.length)
                    points.value = pts
                } else if (Array.isArray(pts) && pts.length === 0 && points.value.length === 0) {
                    console.log('Background fetch returned empty points array')
                }
            })
            .catch((err) => {
                console.warn('Background point fetch failed or timed out:', err.message)
                // 如果 deviceInfo 也没拿到点位，尝试一次不带超时的拉取作为兜底
                if (points.value.length === 0) {
                    console.log('Fallback: Attempting full point fetch without short timeout...')
                    request.get(`/api/channels/${channelId.value}/devices/${deviceId.value}/points`)
                        .then(pts => {
                            if (Array.isArray(pts)) points.value = pts
                        })
                }
            })
    } catch (e) {
        console.error('fetchPoints error:', e)
        showMessage('获取点位失败: ' + e.message, 'error')
    } finally {
        // 不等待后台拉取完成，首屏已就绪
        loading.value = false
        console.log('fetchPoints finished. Loading state:', loading.value)
    }
}

const scanDialog = reactive({
    visible: false,
    loading: false,
    results: [],
    selected: [],
    selectAll: false,
    varsOnly: true,
    mode: 'fast'
})

const existingAddresses = computed(() => {
    const set = new Set()
    for (const p of points.value || []) {
        if (p && p.address) set.add(p.address)
    }
    return set
})

const scanFilteredResults = computed(() => {
    if (!scanDialog.varsOnly) return scanDialog.results
    const filtered = scanDialog.results.filter(r => !r.isOpcNode || r.type === 'Variable')
    if (channelProtocol.value === 'opc-ua' && scanDialog.results.length > 0 && filtered.length === 0) {
        return scanDialog.results
    }
    return filtered
})

const getStatusColor = (status) => {
    switch (status) {
        case 'new': return 'success'
        case 'existing': return 'grey'
        case 'removed': return 'error'
        default: return 'grey'
    }
}

const getStatusText = (status) => {
    switch (status) {
        case 'new': return '新增'
        case 'existing': return '存量'
        case 'removed': return '已删除'
        default: return status
    }
}

const openDiscoverDialog = () => {
    scanDialog.visible = true
    scanDialog.results = []
    scanDialog.selected = []
    scanDialog.selectAll = false
    scanDialog.varsOnly = true
    // 自动开始扫描，减少多余操作
    scanPoints()
}

// Value Detail Dialog Logic
const valueDialog = reactive({
    visible: false,
    value: '',
    decodedValue: '',
    decodeType: 'text',
    isBase64: false,
    byteLength: 2,
    wordOrder: 'AB'
})

const valueWordOrderOptions = computed(() => {
    const bytes = Number(valueDialog.byteLength) || 0
    return getWordOrderOptionsForBytes(bytes)
})

const numericFormats = computed(() => {
    const raw = valueDialog.value
    if (raw === '' || raw === null || raw === undefined) return null
    const n = Number(raw)
    if (!Number.isFinite(n)) return null
    const byteLength = Number(valueDialog.byteLength) || 2
    const bits = BigInt(byteLength * 8)
    try {
        let base = BigInt(Math.trunc(n))
        const one = 1n
        const mask = (one << bits) - one
        base = base & mask

        const bytes = new Array(byteLength)
        let tmp = base
        for (let i = byteLength - 1; i >= 0; i--) {
            bytes[i] = Number(tmp & 0xFFn)
            tmp >>= 8n
        }

        let reordered = bytes
        const wo = valueDialog.wordOrder || ''
        if (byteLength > 1 && wo) {
            reordered = reorderBytes(bytes, byteLength, wo)
        }

        let unsigned = 0n
        for (const b of reordered) {
            unsigned = (unsigned << 8n) + BigInt(b & 0xFF)
        }

        const signBit = 1n << (bits - 1n)
        let signed = unsigned
        if (unsigned & signBit) {
            signed = unsigned - (one << bits)
        }

        const hexDigits = Math.max(2, (byteLength * 8) / 4)
        const hex = '0x' + unsigned.toString(16).toUpperCase().padStart(hexDigits, '0')
        const binary = unsigned.toString(2).padStart(byteLength * 8, '0')

        return {
            signed: signed.toString(),
            unsigned: unsigned.toString(),
            hex,
            binary
        }
    } catch (e) {
        return null
    }
})

const isBase64 = (str) => {
    if (typeof str !== 'string' || str.length === 0) return false;
    try {
        return btoa(atob(str)) === str;
    } catch (err) {
        return false;
    }
}

const showFullValue = (payload) => {
    let val = payload
    let byteLength = valueDialog.byteLength
    let wordOrder = valueDialog.wordOrder

    if (payload && typeof payload === 'object' && 'value' in payload) {
        val = payload.value
        if (payload.byteLength) {
            byteLength = payload.byteLength
        }
        if (payload.wordOrder) {
            wordOrder = payload.wordOrder
        }
    }

    if (typeof val === 'object' && val !== null) {
        val = JSON.stringify(val)
    }

    valueDialog.value = String(val)
    valueDialog.decodedValue = ''
    valueDialog.decodeType = 'text'
    valueDialog.byteLength = byteLength || 2
    valueDialog.wordOrder = valueDialog.byteLength > 1 ? (wordOrder || 'AB') : ''

    valueDialog.isBase64 = isBase64(valueDialog.value)
    if (valueDialog.isBase64) {
        tryDecode('text')
    }
    valueDialog.visible = true
}

const tryDecode = (type) => {
    valueDialog.decodeType = type
    if (!valueDialog.value) return
    
    try {
        const raw = atob(valueDialog.value)
        if (type === 'text') {
            const bytes = Uint8Array.from(raw, c => c.charCodeAt(0))
            valueDialog.decodedValue = new TextDecoder().decode(bytes)
        } else if (type === 'hex') {
            let result = ''
            for (let i = 0; i < raw.length; i++) {
                const hex = raw.charCodeAt(i).toString(16)
                result += (hex.length === 2 ? hex : '0' + hex) + ' '
            }
            valueDialog.decodedValue = result.toUpperCase().trim()
        } else if (type === 'json') {
            const bytes = Uint8Array.from(raw, c => c.charCodeAt(0))
            const str = new TextDecoder().decode(bytes)
            valueDialog.decodedValue = JSON.stringify(JSON.parse(str), null, 2)
        }
    } catch (e) {
        valueDialog.decodedValue = 'Decode failed: ' + e.message
    }
}

const scanPoints = async () => {
    scanDialog.loading = true
    scanDialog.results = []
    try {
        // Ensure device info is loaded
        if (!deviceInfo.value) {
             try {
                 const dev = await request.get(`/api/channels/${channelId.value}/devices/${deviceId.value}`)
                 if (dev) {
                     deviceInfo.value = dev
                 }
             } catch (e) {
                 console.error('Failed to re-fetch device info', e)
             }
        }

        console.log('Scanning points for device:', deviceInfo.value)
        if (!deviceInfo.value || !deviceInfo.value.config) {
             showMessage('无法获取设备配置 (请检查设备连接或配置)', 'error')
             return
        }
        // OPC UA 设备前置校验：必须存在 endpoint
        if (channelProtocol.value === 'opc-ua') {
            const ep = deviceInfo.value.config.endpoint
            if (!ep || typeof ep !== 'string' || ep.length === 0) {
                showMessage('OPC UA 设备未配置 endpoint，无法扫描', 'error')
                return
            }
        }
        
        // Handle device_id being 0 or string "0" for BACnet
        let targetDeviceId = null
        if (channelProtocol.value === 'bacnet-ip') {
            // 优先使用 instance_id，其次 device_id
            const cfg = deviceInfo.value.config || {}
            targetDeviceId = cfg.instance_id
            if (targetDeviceId === undefined || targetDeviceId === null || targetDeviceId === '') {
                targetDeviceId = cfg.device_id
            }

            if (targetDeviceId === undefined || targetDeviceId === null || targetDeviceId === '') {
                showMessage('无法获取设备实例ID (config.instance_id 或 device_id)', 'error')
                return
            }
        }
        
        // Call device-specific scan endpoint
        const payload = {
            mode: scanDialog.mode
        }
        // 如果提取到了 device_id (BACnet Instance ID)，则显式传递给后端
        if (targetDeviceId !== null) {
            payload.device_id = parseInt(targetDeviceId)
        }

        const res = await request.post(`/api/channels/${channelId.value}/devices/${deviceId.value}/scan`, payload, { timeout: 60000 })
        
        if (Array.isArray(res)) {
            if (res.length === 0) {
                showMessage('扫描结果为空', 'warning')
            }
            if (channelProtocol.value === 'opc-ua') {
                // Flatten OPC UA tree for display
                scanDialog.results = flattenOpcNodes(res)
            } else {
                // For BACnet (and others), process diff_status based on existing points in UI
                // This overrides backend's "existing" status (from driver history) if the point was deleted in App
                scanDialog.results = res.map(item => {
                    if (channelProtocol.value === 'bacnet-ip') {
                         const key = `${item.type}:${item.instance}`
                         if (existingAddresses.value.has(key)) {
                             item.diff_status = 'existing'
                         } else {
                             // If not in App, reset 'existing' to 'new' so it can be re-added
                             if (item.diff_status === 'existing') {
                                 item.diff_status = 'new'
                             }
                         }
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
        scanDialog.loading = false
    }
}

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
            type: node.type, // "Variable" or "Folder"
            description: node.node_id // Show NodeID in description/extra
        }
        // Mark existing/new for sync status
        if (node.type === 'Variable' && node.node_id) {
            item.diff_status = existingAddresses.value.has(node.node_id) ? 'existing' : 'new'
        }
        result.push(item)
        
        // Process children
        if (node.children && node.children.length > 0) {
            result = result.concat(flattenOpcNodes(node.children, level + 1))
        }
    }
    return result
}

const toggleSelectAllScan = (val) => {
    if (val) {
        // Only select non-disabled rows
        scanDialog.selected = scanFilteredResults.value.filter(r => !(r.diff_status === 'existing'))
    } else {
        scanDialog.selected = []
    }
}

const addSelectedPoints = async () => {
    if (scanDialog.selected.length === 0) return
    
    scanDialog.loading = true
    let successCount = 0
    let failCount = 0
    
    for (const obj of scanDialog.selected) {
        let pointPayload = {}
        
        if (channelProtocol.value === 'opc-ua') {
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
            await request.post(`/api/channels/${channelId.value}/devices/${deviceId.value}/points`, pointPayload)
            successCount++
        } catch (e) {
            console.error(e)
            failCount++
        }
    }
    
    scanDialog.loading = false
    showMessage(`已添加 ${successCount} 个点位${failCount > 0 ? `，${failCount} 个失败` : ''}`, failCount > 0 ? 'warning' : 'success')
    scanDialog.visible = false
    fetchPoints()
}

// WebSocket Logic
let ws = null
const connectWs = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    ws = new WebSocket(`${protocol}//${host}/api/ws/values`)

    ws.onopen = () => {
        globalState.wsStatus.connected = true
    }

    ws.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data)
            if (data.channel_id === channelId.value && data.device_id === deviceId.value) {
                const idx = points.value.findIndex(p => p.id === data.point_id)
                if (idx !== -1) {
                    points.value[idx].value = data.value
                    points.value[idx].quality = data.quality
                    points.value[idx].timestamp = data.timestamp
                }
            }
        } catch (e) { console.error(e) }
    }

    ws.onclose = () => {
        globalState.wsStatus.connected = false
    }
}

const channelProtocol = ref('')
const bacnetObjectTypes = [
    'AnalogInput', 'AnalogOutput', 'AnalogValue',
    'BinaryInput', 'BinaryOutput', 'BinaryValue',
    'MultiStateInput', 'MultiStateOutput', 'MultiStateValue'
]

const fetchChannel = async () => {
    try {
        const data = await request.get(`/api/channels/${channelId.value}`)
        if (data && data.protocol) {
            channelProtocol.value = data.protocol
        }
    } catch (e) {
        console.error('Failed to fetch channel info', e)
    }
}

const metrics = reactive({
    connectionSeconds: 0,
    reconnectCount: 0,
    localAddr: '',
    remoteAddr: '',
    lastDisconnectTime: null,
    loading: false
})

const fetchMetrics = async () => {
    if (!channelId.value) return
    metrics.loading = true
    try {
        const data = await request.get(`/api/channels/${channelId.value}/metrics`)
        if (data) {
            metrics.connectionSeconds = data.connectionSeconds || 0
            metrics.reconnectCount = data.reconnectCount || 0
            metrics.localAddr = data.localAddr || ''
            metrics.remoteAddr = data.remoteAddr || ''
            metrics.lastDisconnectTime = data.lastDisconnectTime || null
        }
    } catch (e) {
        console.error('Failed to fetch metrics', e)
    } finally {
        metrics.loading = false
    }
}

let metricsTimer = null

onMounted(() => {
    fetchPoints()
    fetchChannel()
    connectWs()
    fetchMetrics()
    metricsTimer = setInterval(fetchMetrics, 5000)
})

onUnmounted(() => {
    if (ws) ws.close()
    if (metricsTimer) clearInterval(metricsTimer)
})

// Helpers
const formatDuration = (seconds) => {
    if (!seconds || seconds < 0) return '未连接'
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = Math.floor(seconds % 60)
    if (h > 0) return `${h}时${m}分${s}秒`
    if (m > 0) return `${m}分${s}秒`
    return `${s}秒`
}

const formatValue = (val) => {
    if (typeof val === 'number') return val.toFixed(2)
    return val
}
const formatDate = (ts) => new Date(ts).toLocaleString()
const isQualityGood = (q) => q === 'Good' || q === 'good'

const getRegisterCountForDatatype = (dt) => {
    const t = (dt || '').toLowerCase()
    if (['int32', 'uint32', 'float32', 'dword'].includes(t)) return 2
    if (['int64', 'uint64', 'float64', 'double', 'lword'].includes(t)) return 4
    return 1
}

const getRegisterHint = (point) => {
    const dt = point.datatype || point.dataType || ''
    const count = getRegisterCountForDatatype(dt)
    const addr = typeof point.address === 'string' ? point.address : String(point.address || '')
    const base = Number(addr)
    if (!Number.isFinite(base) || count <= 0) {
        if (dt && addr) return `${addr} · ${dt}`
        if (addr) return addr
        return dt
    }
    if (count === 1) {
        return `${base} (1 reg) · ${dt}`
    }
    const end = base + count - 1
    return `${base}-${end} (${count} regs) · ${dt}`
}

// Write Logic
const openWriteDialog = (point) => {
    writeDialog.deviceID = deviceId.value
    writeDialog.pointID = point.id
    writeDialog.dataType = (point.datatype || '').toLowerCase()
    // 初始化不同类型的输入
    if (isBoolType(writeDialog.dataType)) {
        writeDialog.valueBool = false
    } else if (isStringType(writeDialog.dataType)) {
        writeDialog.valueStr = ''
    } else {
        writeDialog.valueNum = 0
    }
    writeDialog.visible = true
}

const submitWrite = async () => {
    writeDialog.loading = true
    try {
        const payloadValue = normalizeWriteValue()
        
        // Handle BACnet Priority
        let finalValue = payloadValue
        if (channelProtocol.value === 'bacnet-ip') {
            finalValue = {
                value: payloadValue,
                priority: writeDialog.priority
            }
        }

        await request.post('/api/write', {
            channel_id: channelId.value,
            device_id: deviceId.value,
            point_id: writeDialog.pointID,
            value: finalValue
        })
        showMessage('写入命令已发送', 'success')
        writeDialog.visible = false
    } catch (e) {
        showMessage('写入失败: ' + e.message, 'error')
    } finally {
        writeDialog.loading = false
    }
}

// 打开点位调试（调用后端 /api/points/:id/debug 并用浏览器提示显示）
const openDebug = async (point) => {
    try {
        const resp = await request.get(`/api/points/${point.id}/debug`, { timeout: 3000, silent: true })
        // 简单弹窗展示调试信息，前端可替换为更复杂的对话框
        alert(JSON.stringify(resp, null, 2))
    } catch (e) {
        showMessage('获取点位调试信息失败: ' + (e.message || e), 'error')
    }
}

// 类型判断与转换
const getProtocolTransport = (p) => {
    if (!p) return 'Unknown'
    const proto = p.toLowerCase()
    if (proto.includes('bacnet')) return 'UDP'
    if (proto.includes('snmp')) return 'UDP'
    if (proto.includes('tcp') || proto.includes('modbus-tcp') || proto.includes('opc') || proto.includes('s7')) return 'TCP'
    if (proto.includes('rtu') || proto.includes('serial')) return 'Serial'
    return 'TCP/IP'
}

const isBoolType = (dt) => ['bool', 'boolean', 'bit'].includes((dt || '').toLowerCase())
const isStringType = (dt) => ['string'].includes((dt || '').toLowerCase())
const isFloatType = (dt) => ['float', 'float32', 'float64', 'double'].includes((dt || '').toLowerCase())
const isIntType = (dt) => ['int8','int16','int32','int64','uint8','uint16','uint32','uint64','word','dword','lword','int','uint'].includes((dt || '').toLowerCase())

const normalizeWriteValue = () => {
    const dt = (writeDialog.dataType || '').toLowerCase()
    if (isBoolType(dt)) {
        return writeDialog.valueBool
    }
    if (isStringType(dt)) {
        return writeDialog.valueStr
    }
    if (isFloatType(dt)) {
        const n = Number(writeDialog.valueNum)
        return isNaN(n) ? 0 : n
    }
    if (isIntType(dt)) {
        const n = parseInt(writeDialog.valueNum)
        return isNaN(n) ? 0 : n
    }
    // Fallback: 原样字符串
    return writeDialog.valueStr || writeDialog.valueNum
}
</script>

<style scoped>
.scan-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: nowrap;
  white-space: nowrap;
  overflow-x: hidden;
  min-width: 0;
}
.scan-toolbar :deep(.v-btn-group) {
  flex: 0 0 auto;
}
.scan-toolbar :deep(.v-btn) {
  flex: 0 0 auto;
  min-width: auto;
}
.scan-toolbar :deep(.v-switch) {
  flex: 0 0 auto;
}
.clone-dialog-full {
  align-items: stretch;
}
</style>
