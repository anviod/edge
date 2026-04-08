<template>
    <div class="edge-compute-container">
        <a-tabs v-model:active-key="tab" class="mb-4">
            <a-tab-pane key="metrics" title="监控面板">
                <EdgeComputeMetrics />
            </a-tab-pane>
            <a-tab-pane key="rules" title="规则管理">
                <a-card class="mb-4 borderless-card">
                    <a-card-header class="d-flex justify-space-between align-items-center">
                        <a-button type="primary" @click="openDialog">
                            <template #icon><IconPlus /></template>
                            添加规则
                        </a-button>
                    </a-card-header>
                    
                    <a-card-body>
                        <!-- 批量操作工具栏 -->
                        <div v-show="selectedRuleKeys.length > 0" class="table-toolbar-industrial">
                            <div class="flex items-center gap-2">
                                <span class="selection-count">已选 {{ selectedRuleKeys.length }} 项</span>
                                <a-divider direction="vertical" />
                                <a-button size="small" type="outline" @click="handleBatchEnable(true)">批量启用</a-button>
                                <a-button size="small" type="outline" @click="handleBatchEnable(false)">批量禁用</a-button>
                                <a-button size="small" type="outline" status="danger" @click="handleBatchDelete">批量删除</a-button>
                            </div>
                        </div>
                        
                        <a-table
                            :columns="ruleColumns"
                            :data="rules"
                            size="small"
                            :bordered="true"
                            :row-selection="{ type: 'checkbox', showCheckedAll: true }"
                            v-model:selected-keys="selectedRuleKeys"
                            row-key="id"
                            class="industrial-table"
                        >
                            <template #operations="{ record }">
                                <a-button type="text" size="small" @click="editRule(record)">
                                    <template #icon><IconEdit /></template>
                                </a-button>
                                <a-button type="text" size="small" status="danger" @click="deleteRule(record)">
                                    <template #icon><IconDelete /></template>
                                </a-button>
                            </template>
                            <template #enable="{ record }">
                                <a-tag :color="record.enable ? 'success' : 'default'" size="small">
                                    {{ record.enable ? '启用' : '禁用' }}
                                </a-tag>
                            </template>
                            <template #type="{ record }">
                                {{ formatRuleType(record.type) }}
                            </template>
                            <template #trigger_mode="{ record }">
                                {{ formatTriggerMode(record.trigger_mode) }}
                            </template>
                        </a-table>
                    </a-card-body>
                </a-card>
            </a-tab-pane>
            <a-tab-pane key="status" title="运行记录">
                <a-card class="borderless-card">
                    <a-card-header class="d-flex justify-end align-items-center">
                        <a-button type="text" @click="fetchRuleStates">
                            <template #icon><IconRefresh /></template>
                        </a-button>
                    </a-card-header>
                    <a-card-body>
                        <a-table
                            :columns="statusColumns"
                            :data="ruleStates"
                            size="small"
                            :bordered="true"
                            class="industrial-table"
                        >
                            <template #current_status="{ record }">
                                <a-tag :color="getStatusColor(record.current_status)" size="small">
                                    {{ record.current_status }}
                                </a-tag>
                            </template>
                            <template #last_trigger="{ record }">
                                {{ formatDate(record.last_trigger) }}
                            </template>
                            <template #operations="{ record }">
                                <a-button type="text" size="small" @click="viewWindowData(record.rule_id, record.rule_name)">
                                    查看窗口数据
                                </a-button>
                            </template>
                            <template #error_message="{ record }">
                                <span class="text-error">{{ record.error_message }}</span>
                            </template>
                        </a-table>
                    </a-card-body>
                </a-card>
            </a-tab-pane>
            <a-tab-pane key="logs" title="日志查询">
                <a-card class="h-100">
                    <a-card-body>
                        <a-row :gutter="[16, 16]" class="mb-4">
                            <a-col :span="24" :md="6">
                                <a-input v-model="query.start" placeholder="开始时间" type="datetime-local" size="small" />
                            </a-col>
                            <a-col :span="24" :md="6">
                                <a-input v-model="query.end" placeholder="结束时间" type="datetime-local" size="small" />
                            </a-col>
                            <a-col :span="24" :md="4">
                                <a-input v-model="query.ruleId" placeholder="规则ID (可选)" size="small" />
                            </a-col>
                            <a-col :span="24" :md="4">
                                <a-button type="primary" block size="small" @click="queryLogs">查询</a-button>
                            </a-col>
                            <a-col :span="24" :md="4">
                               <a-button type="success" size="small" @click="exportLogs" :disabled="logs.length === 0">
                                   <template #icon><IconDownload /></template>
                                   导出 CSV
                               </a-button>
                            </a-col>
                        </a-row>
                        
                        <div class="logs-table-container">
                            <a-table
                                :columns="logColumns"
                                :data="logs"
                                size="small"
                                :bordered="false"
                                :scroll="{ x: 1200 }"
                            >
                                <template #status="{ record }">
                                    <a-tag :color="getStatusColor(record.status)" size="small">
                                        {{ record.status }}
                                    </a-tag>
                                </template>
                                <template #last_value="{ record }">
                                <span class="single-line-cell" @click="showDetails('完整值', record.last_value)" title="点击查看详情">
                                    {{ record.last_value }}
                                </span>
                            </template>
                            <template #error_message="{ record }">
                                <span class="single-line-cell text-error" @click="showDetails('错误详情', record.error_message)" title="点击查看详情">
                                    {{ record.error_message }}
                                </span>
                            </template>
                            </a-table>
                        </div>
                    </a-card-body>
                </a-card>
            </a-tab-pane>
        </a-tabs>

        <!-- Window Data Dialog -->
        <a-modal v-model:visible="windowDialog" title="窗口数据预览 ({{ currentWindowRuleName }})" width="600px">
            <a-table
                :columns="windowDataColumns"
                :data="windowData"
                size="small"
                :bordered="false"
            >
                <template #ts="{ record }">
                    {{ formatDate(record.ts) }}
                </template>
            </a-table>
            <template #footer>
                <a-button type="primary" @click="windowDialog = false">关闭</a-button>
            </template>
        </a-modal>

        <!-- Details Dialog -->
        <a-modal v-model:visible="detailsDialog" title="{{ detailTitle }}" width="800px">
            <a-tabs v-model:active-key="detailsTab" class="mb-4">
                <a-tab-pane key="text" title="文本/原始内容"></a-tab-pane>
                <a-tab-pane key="hex" title="Hex 视图" :disabled="!decodedHex"></a-tab-pane>
            </a-tabs>
            
            <div v-if="detailsTab === 'text'">
                <div class="text-gray-500 mb-2">内容长度: {{ detailContent.length }}</div>
                <a-textarea
                    v-model="detailContent"
                    readonly
                    :auto-size="{ minRows: 5, maxRows: 15 }"
                    style="font-family: monospace;"
                />
            </div>
            
            <div v-else-if="detailsTab === 'hex'">
                <div class="text-gray-500 mb-2">Hex 视图 ({{ decodedBytes ? decodedBytes.length : 0 }} bytes)</div>
                <a-textarea
                    v-model="decodedHex"
                    readonly
                    :auto-size="{ minRows: 5, maxRows: 15 }"
                    style="font-family: monospace;"
                />
            </div>

            <a-alert v-if="detectedFile" type="info" class="mt-4">
                <div class="d-flex justify-space-between align-items-center">
                    <span>检测到文件格式: <strong>{{ detectedFile.name }} ({{ detectedFile.ext }})</strong></span>
                    <a-button type="primary" size="small" @click="downloadDetectedFile">
                        <template #icon><IconDownload /></template>
                        下载文件
                    </a-button>
                </div>
            </a-alert>
            <template #footer>
                <a-button v-if="!decodedHex" @click="tryDecode">
                    <template #icon><IconCode /></template>
                    尝试 Base64 解码
                </a-button>
                <a-button type="primary" @click="detailsDialog = false">关闭</a-button>
            </template>
        </a-modal>

        <!-- Rule Dialog -->
        <a-modal v-model:visible="dialog" :title="editingRule ? '编辑规则' : '添加规则'" width="80%" modal-class="industrial-white-modal">
            <a-form ref="form" :model="currentRule" layout="vertical" class="industrial-form">
                <div class="form-section">
                    <div class="section-title">基础配置</div>
                    <a-row :gutter="16">
                        <a-col :span="16">
                            <a-form-item field="name" label="规则名称" required>
                                <a-input v-model="currentRule.name" placeholder="请输入规则名称" class="rect-input" />
                            </a-form-item>
                        </a-col>
                        <a-col :span="8">
                            <a-form-item field="enable" label="启用状态">
                                <a-switch v-model="currentRule.enable" type="round" />
                            </a-form-item>
                        </a-col>
                    </a-row>
                    <a-row :gutter="16">
                        <a-col :span="6">
                            <a-form-item field="type" label="规则类型">
                                <a-select 
                                    v-model="currentRule.type" 
                                    class="rect-input"
                                    :options="[
                                        {label: 'Threshold (阈值触发)', value: 'threshold'},
                                        {label: 'Calculation (计算公式)', value: 'calculation'},
                                        {label: 'Window (时间/计数窗口)', value: 'window'},
                                        {label: 'State (状态持续)', value: 'state'}
                                    ]" 
                                />
                            </a-form-item>
                        </a-col>
                        <a-col :span="6">
                            <a-form-item field="priority" label="优先级">
                                <a-input-number v-model="currentRule.priority" class="rect-input" />
                            </a-form-item>
                        </a-col>
                        <a-col :span="6">
                            <a-form-item field="trigger_mode" label="触发模式">
                                <a-select
                                    v-model="currentRule.trigger_mode"
                                    class="rect-input"
                                    :options="[{label: '始终触发', value: 'always'}, {label: '仅状态改变时触发', value: 'on_change'}]"
                                />
                            </a-form-item>
                        </a-col>
                        <a-col :span="6">
                            <a-form-item field="check_interval" label="检查频率">
                                <a-select
                                    v-model="currentRule.check_interval"
                                    class="rect-input"
                                    :options="['1s', '5s', '10s', '30s', '1m']"
                                />
                            </a-form-item>
                        </a-col>
                    </a-row>
                    <div class="text-gray-500 text-sm mt-1">{{ getRuleTypeExplanation(currentRule.type) }}</div>
                </div>

                <div class="form-section">
                    <div class="section-header-row">
                        <div class="section-title">数据源 Sources</div>
                        <div class="flex gap-2">
                            <a-button type="outline" size="small" @click="detectInvalidSources">
                                <template #icon><IconRefresh /></template>
                                自动检测
                            </a-button>
                            <a-button type="outline" status="danger" size="small" @click="clearInvalidSources">
                                <template #icon><IconDelete /></template>
                                一键清除
                            </a-button>
                            <a-button type="primary" size="small" @click="addSource">
                                <template #icon><IconPlus /></template>
                                添加
                            </a-button>
                        </div>
                    </div>
                    <div class="text-gray-500 text-sm mb-4">
                        请为每个数据源设置别名（如 t1, t2），然后在触发条件中使用别名编写逻辑公式（例如：t1 > 20 || t2 > 30）。
                    </div>
                    <div v-for="(src, index) in currentRule.sources" :key="index" class="source-row">
                        <div class="source-index">#{{ index + 1 }}</div>
                        <a-row :gutter="12" class="flex-1">
                            <a-col :span="24" :md="5">
                                <a-select
                                    v-model="src.channel_id"
                                    :options="channels"
                                    placeholder="通道"
                                    class="rect-input"
                                    @change="() => onSourceChannelChange(src)"
                                />
                            </a-col>
                            <a-col :span="24" :md="5">
                                <a-select
                                    v-model="src.device_id"
                                    :options="src._deviceList || []"
                                    placeholder="设备"
                                    class="rect-input"
                                    :disabled="!src.channel_id"
                                    @change="() => onSourceDeviceChange(src)"
                                    @click="() => loadSourceDevices(src)"
                                />
                            </a-col>
                            <a-col :span="24" :md="5">
                                <a-select
                                    v-model="src.point_id"
                                    :options="src._pointList || []"
                                    placeholder="点位"
                                    class="rect-input"
                                    :disabled="!src.device_id"
                                    @click="() => loadSourcePoints(src)"
                                />
                            </a-col>
                            <a-col :span="24" :md="7">
                                <a-input 
                                    v-model="src.alias" 
                                    placeholder="t1"
                                    class="rect-input w-full"
                                />
                            </a-col>
                            <a-col :span="24" :md="2" class="flex items-center justify-end">
                                <a-button type="text" status="danger" @click="removeSource(index)">
                                    <IconDelete />
                                </a-button>
                            </a-col>
                        </a-row>
                    </div>
                </div>

                <!-- 窗口配置 -->
                <div v-if="currentRule.type === 'window'" class="form-section">
                    <div class="section-title">窗口配置</div>
                    <a-row :gutter="16">
                        <a-col :span="8">
                            <a-form-item field="window.type" label="窗口类型">
                                <a-select v-model="currentRule.window.type" :options="['sliding', 'tumbling']" class="rect-input" />
                            </a-form-item>
                        </a-col>
                        <a-col :span="8">
                            <a-form-item field="window.size" label="窗口大小">
                                <a-input v-model="currentRule.window.size" placeholder="例如: 10s 或 100" class="rect-input" />
                            </a-form-item>
                        </a-col>
                        <a-col :span="8">
                            <a-form-item field="window.aggr_func" label="聚合函数">
                                <a-select v-model="currentRule.window.aggr_func" :options="['avg', 'min', 'max', 'sum', 'count', 'rate']" class="rect-input" />
                            </a-form-item>
                        </a-col>
                    </a-row>
                </div>

                <!-- 状态维持 -->
                <div v-if="currentRule.type === 'state' || currentRule.type === 'threshold'" class="form-section">
                    <div class="section-title">状态维持</div>
                    <a-row :gutter="16">
                        <a-col :span="12">
                            <a-form-item field="state.duration" label="持续时间 (Duration)">
                                <a-input v-model="currentRule.state.duration" placeholder="例如: 10s" class="rect-input" />
                            </a-form-item>
                        </a-col>
                        <a-col :span="12">
                            <a-form-item field="state.count" label="连续次数 (Count)">
                                <a-input-number v-model="currentRule.state.count" class="rect-input" />
                            </a-form-item>
                        </a-col>
                    </a-row>
                </div>

                <!-- 规则逻辑 Logic -->
                <div v-if="currentRule.type !== 'calculation'" class="form-section">
                    <div class="section-header-row">
                        <div class="section-title">规则逻辑 Logic</div>
                        <a-button size="small" @click="openHelper(currentRule.condition, (v) => currentRule.condition = v)">
                            公式助手
                        </a-button>
                    </div>
                    <a-form-item label="表达式">
                        <a-textarea
                            v-model="currentRule.condition"
                            placeholder="t1 > 50 && t2 < 80"
                            :rows="3"
                            class="code-input rect-input"
                        />
                        <template #extra>支持数据源别名（如 t1, t2）和逻辑运算符</template>
                    </a-form-item>
                </div>

                <!-- 计算公式 -->
                <div v-if="currentRule.type === 'calculation'" class="form-section">
                    <div class="section-header-row">
                        <div class="section-title">计算公式</div>
                        <a-button size="small" @click="openHelper(currentRule.expression, (v) => currentRule.expression = v)">
                            公式助手
                        </a-button>
                    </div>
                    <a-form-item label="表达式">
                        <a-textarea
                            v-model="currentRule.expression"
                            placeholder="value * 1.5 + 32"
                            :rows="3"
                            class="code-input rect-input"
                        />
                        <template #extra>支持数学运算符和函数（如 abs, sqrt, sin 等）</template>
                    </a-form-item>
                </div>

                <!-- 动作 Actions -->
                <div class="form-section">
                    <div class="section-header-row">
                        <div class="section-title">执行动作 (Action)</div>
                        <div class="flex gap-2">
                            <a-button type="outline" size="small" @click="detectInvalidActions">
                                <template #icon><IconRefresh /></template>
                                自动检测
                            </a-button>
                            <a-button type="outline" status="danger" size="small" @click="clearInvalidActions">
                                <template #icon><IconDelete /></template>
                                一键清除
                            </a-button>
                            <a-button type="primary" size="small" @click="addAction">
                                <template #icon><IconPlus /></template>
                                添加动作
                            </a-button>
                        </div>
                    </div>
                    <div v-if="!currentRule.actions || currentRule.actions.length === 0" class="empty-box">
                        无动作
                    </div>
                    <div v-else>
                        <div v-for="(action, index) in currentRule.actions" :key="index" class="action-card">
                            <div class="action-index">STEP {{ index + 1 }}</div>
                            <ActionEditor 
                                v-model="currentRule.actions[index]" 
                                :channels="channels" 
                                @remove="removeAction(index)" 
                            />
                            <a-button type="text" status="danger" @click="removeAction(index)" class="mt-2">
                                <IconDelete />
                            </a-button>
                        </div>
                    </div>
                </div>
            </a-form>
            <template #footer>
                <a-button @click="dialog = false">取消</a-button>
                <a-button type="primary" @click="saveRule">保存</a-button>
            </template>
        </a-modal>

        <!-- Help Dialog -->
        <a-modal v-model:visible="helpDialog" title="边缘计算规则配置指南" width="800px" :scrollable="true">
            <div class="help-content">
                <div class="text-lg font-medium mb-4">1. 基础概念</div>
                <a-alert type="info" class="mb-4">
                    <ul>
                        <li><strong>数据源 (Sources)</strong>: 规则的输入变量。请为每个源设置简短的 <code>别名 (Alias)</code> (如 t1, p1)，以便在表达式中引用。</li>
                        <li><strong>触发条件 (Condition)</strong>: 返回 true/false 的布尔表达式。仅当条件满足时触发动作。</li>
                        <li><strong>动作 (Actions)</strong>: 规则触发后执行的一系列操作。</li>
                    </ul>
                </a-alert>

                <div class="text-lg font-medium mb-4">2. 常见场景最佳实践</div>
                
                <a-collapse class="mb-4">
                    <a-collapse-item title="场景 A: 简单越限报警 (Threshold)">
                        <p><strong>目标</strong>: 当温度 (t1) 超过 50 度时，记录日志并发送 MQTT 告警。</p>
                        <ul class="pl-6 mt-2">
                            <li><strong>类型</strong>: Threshold</li>
                            <li><strong>数据源</strong>: 添加温度点位，别名设为 <code>t1</code></li>
                            <li><strong>触发条件</strong>: <code>t1 > 50</code></li>
                            <li><strong>动作</strong>: 
                                <ol class="pl-6">
                                    <li>Log: 级别 Warn, 内容 "温度过高: ${t1}"</li>
                                    <li>MQTT: Topic "alarm/temp", 内容 "温度异常: ${t1}"</li>
                                </ol>
                            </li>
                        </ul>
                    </a-collapse-item>
                    <a-collapse-item title="场景 B: 顺序联动控制 (Sequence Workflow)">
                        <p><strong>目标</strong>: 启动设备 A，等待 30秒，确认 A 已启动后再启动设备 B。如果 A 启动失败，则回退关闭 A。</p>
                        <ul class="pl-6 mt-2">
                            <li><strong>类型</strong>: Threshold (或 State)</li>
                            <li><strong>触发条件</strong>: <code>start_signal == 1</code> (启动信号)</li>
                            <li><strong>动作</strong>: 选择 <strong>Sequence</strong> 类型，添加以下步骤：
                                <ol class="pl-6 mt-1">
                                    <li><strong>Device Control</strong>: 开启设备 A (Value: 1)</li>
                                    <li><strong>Delay</strong>: 30s</li>
                                    <li><strong>Check</strong>: 
                                        <ul class="pl-6">
                                            <li>选择设备 A 的状态点位</li>
                                            <li>表达式: <code>v == 1</code> (确认运行中)</li>
                                            <li>重试: 3次, 间隔: 2s</li>
                                            <li><strong>On Fail (失败回退)</strong>: 添加 Device Control 动作 -> 关闭设备 A (Value: 0)</li>
                                        </ul>
                                    </li>
                                    <li><strong>Device Control</strong>: 开启设备 B (Value: 1)</li>
                                </ol>
                            </li>
                        </ul>
                        <a-alert type="warning" class="mt-2">
                            <strong>注意:</strong> Sequence 中的 Check 动作如果失败且未在 On Fail 中成功处理异常（通常用于记录日志或回退），整个 Sequence 将会终止，后续步骤（如开启设备 B）不会执行。这是实现安全联动逻辑的关键。
                        </a-alert>
                    </a-collapse-item>
                    <a-collapse-item title="场景 C: 批量设备控制 (Batch Control)">
                        <p><strong>目标</strong>: 一键关闭所有相关设备 (A, B, C)。</p>
                        <ul class="pl-6 mt-2">
                            <li><strong>动作</strong>: 选择 <strong>Device Control</strong> 类型</li>
                            <li><strong>配置</strong>: 开启 <strong>Batch Control (批量控制)</strong> 开关</li>
                            <li><strong>目标列表</strong>:
                                <ul class="pl-6">
                                    <li>目标 1: 设备 A, 开关点位, 值 0</li>
                                    <li>目标 2: 设备 B, 开关点位, 值 0</li>
                                    <li>目标 3: 设备 C, 开关点位, 值 0</li>
                                </ul>
                            </li>
                        </ul>
                        <p class="mt-2 text-sm text-gray-500">优势: 批量控制会并行发送写入请求，相比连续的单点控制动作，响应速度更快。</p>
                    </a-collapse-item>
                    <a-collapse-item title="场景 D: 位运算与状态字控制 (Bitwise)">
                        <p><strong>目标</strong>: 仅修改状态字的第 4 位 (置 1)，保持其他位不变。</p>
                        <ul class="pl-6 mt-2">
                            <li><strong>动作</strong>: Device Control</li>
                            <li><strong>Expr (公式)</strong>: <code>bitset(v, 4)</code> 或 <code>v | 8</code> (0-based index)</li>
                            <li><strong>说明</strong>: 系统会自动读取当前值 -> 计算新值 -> 写入 (Read-Modify-Write 机制)。</li>
                        </ul>
                        <a-alert type="success" class="mt-2">
                            <strong>RMW 机制:</strong> 网关会自动处理并发冲突，确保在修改某一位时，不会覆盖其他位在同一时刻发生的变化（仅针对支持原子操作或网关级锁定的场景）。
                        </a-alert>
                    </a-collapse-item>
                </a-collapse>

                <div class="text-lg font-medium mb-4">3. 表达式语法参考</div>
                <a-table :columns="syntaxColumns" :data="syntaxData" size="small" :bordered="false" class="mb-4"></a-table>
                
                <div class="text-lg font-medium mb-4">4. 动作类型详解</div>
                <a-collapse>
                    <a-collapse-item title="Log (日志)">
                        记录规则触发信息到系统日志。
                        <ul class="pl-6 mt-2">
                            <li><strong>Level</strong>: 日志级别 (Info/Warn/Error)。</li>
                            <li><strong>Message</strong>: 支持 <code>${v}</code> 或 <code>${alias}</code> 模板变量。</li>
                        </ul>
                    </a-collapse-item>
                    <a-collapse-item title="Device Control (设备控制)">
                        向设备写入值。
                        <ul class="pl-6 mt-2">
                            <li><strong>单点模式</strong>: 直接控制一个点位。</li>
                            <li><strong>批量模式</strong>: 同时控制多个点位。</li>
                            <li><strong>Expression</strong>: 可选。用于计算写入值（支持位操作）。</li>
                        </ul>
                    </a-collapse-item>
                    <a-collapse-item title="Sequence (顺序执行)">
                        严格按顺序执行子动作。如果任一步骤失败（如 Check 失败且未处理），整个序列终止。
                    </a-collapse-item>
                    <a-collapse-item title="Check (校验)">
                        读取点位并校验条件。
                        <ul class="pl-6 mt-2">
                            <li><strong>Expression</strong>: 校验公式 (如 <code>v == 1</code>)。</li>
                            <li><strong>Retry</strong>: 失败重试次数。</li>
                                    <li><strong>On Fail</strong>: 校验最终失败后执行的回退动作序列。</li>
                                </ul>
                            </a-collapse-item>
                        </a-collapse>
                    </div>
                    <template #footer>
                        <a-button type="primary" @click="helpDialog = false">关闭</a-button>
                    </template>
                </a-modal>

                <!-- Expression Helper Dialog -->
                <a-modal v-model:visible="helperDialog" title="表达式转换助手" width="600px">
                    <div class="pt-4">
                        <div class="text-gray-500 mb-2">输入标准表达式 (例如: v & 64, v | 1, ~v):</div>
                        <div class="text-gray-500 text-sm mb-4">提示: 系统已直接支持 v.N 语法 (如 v.4) 及 v.bit.N 语法 (如 v.bit.4) 读取第N位，无需转换。</div>
                        <a-textarea
                            v-model="helperInput"
                            placeholder="标准表达式 (Standard Syntax)"
                            :rows="3"
                            :auto-size="{ minRows: 3, maxRows: 6 }"
                        />
                        
                        <div class="d-flex justify-center my-4 gap-4">
                            <a-button type="text" @click="docsDialog = true">
                                <template #icon><IconBook /></template>
                                查看函数文档 (View Docs)
                            </a-button>
                            <a-button type="secondary" @click="convertHelper">
                                <template #icon><IconArrowDown /></template>
                                转换 (Convert)
                            </a-button>
                        </div>
                        
                        <div class="text-gray-500 mb-2">转换结果 (Function Syntax):</div>
                        <a-textarea
                            v-model="helperOutput"
                            placeholder="函数表达式 (Result)"
                            :rows="3"
                            :auto-size="{ minRows: 3, maxRows: 6 }"
                            readonly
                            :disabled="true"
                        />
                    </div>
                    <template #footer>
                        <a-button @click="helperDialog = false">关闭</a-button>
                        <a-button type="primary" @click="applyHelper" :disabled="!helperOutput">应用并填入</a-button>
                    </template>
                </a-modal>
                <!-- Expression Docs Dialog -->
                <a-modal v-model:visible="docsDialog" title="表达式函数参考文档 (Expression Reference)" width="900px" :scrollable="true">
                    <div class="pa-4" style="max-height: 600px; overflow-y: auto;">
                        
                        <a-alert type="info" class="mb-4">
                            <div class="font-medium">基本变量</div>
                            <div><code>value</code> 或 <code>v</code> : 当前触发点位的值 (The current point value).</div>
                            <div><code>t1</code>, <code>t2</code> ... : 数据源别名 (Source aliases defined in rule).</div>
                        </a-alert>

                        <div class="text-lg font-medium mb-4">1. 位操作函数 (Bitwise Operations)</div>
                        <a-table :columns="docsColumns" :data="bitwiseFunctions" size="small" :bordered="false" class="mb-4"></a-table>

                        <div class="text-lg font-medium mb-4">2. 数学函数 (Mathematical Functions)</div>
                        <a-table :columns="docsColumns" :data="mathFunctions" size="small" :bordered="false" class="mb-4"></a-table>

                        <div class="text-lg font-medium mb-4">3. 逻辑函数 (Logical Functions)</div>
                        <a-table :columns="docsColumns" :data="logicalFunctions" size="small" :bordered="false" class="mb-4"></a-table>
                    </div>
                    <template #footer>
                        <a-button type="primary" @click="docsDialog = false">关闭</a-button>
                    </template>
                </a-modal>
    </div>
</template>

<script setup>
import {
  Tabs, TabPane, Card, Table, Button, Modal, Form, FormItem, Input,
  InputNumber, Select, Switch, Alert, Collapse, CollapseItem, Tag, Row, Col
} from '@arco-design/web-vue'
import {
  IconPlus, IconEdit, IconDelete, IconRefresh, IconDownload, IconCode, IconInfoCircle, IconBook, IconArrowDown
} from '@arco-design/web-vue/es/icon'
import ActionEditor from '@/components/ActionEditor.vue'

// 表格列定义
const ruleColumns = [
  { title: '规则名称', dataIndex: 'name' },
  { title: '类型', dataIndex: 'type', slotName: 'type' },
  { title: '触发模式', dataIndex: 'trigger_mode', slotName: 'trigger_mode' },
  { title: '启用状态', dataIndex: 'enable', slotName: 'enable' },
  { title: '优先级', dataIndex: 'priority' },
  { title: '操作', dataIndex: 'operations', slotName: 'operations', fixed: 'right', width: 120 }
]

const statusColumns = [
  { title: '规则名称', dataIndex: 'rule_name' },
  { title: '当前状态', dataIndex: 'current_status', slotName: 'current_status' },
  { title: '最近触发时间', dataIndex: 'last_trigger', slotName: 'last_trigger' },
  { title: '触发次数', dataIndex: 'trigger_count' },
  { title: '最新值', dataIndex: 'last_value' },
  { title: '操作', dataIndex: 'operations', slotName: 'operations' },
  { title: '错误信息', dataIndex: 'error_message', slotName: 'error_message' }
]

const logColumns = [
  { title: '时间', dataIndex: 'minute', width: 160 },
  { title: '规则ID', dataIndex: 'rule_id', width: 120 },
  { title: '规则名称', dataIndex: 'rule_name', width: 160 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 100 },
  { title: '触发次数', dataIndex: 'trigger_count', width: 100 },
  { title: '值', dataIndex: 'last_value', slotName: 'last_value', width: 400 },
  { title: '错误信息', dataIndex: 'error_message', slotName: 'error_message', width: 400 }
]

const windowDataColumns = [
  { title: '时间', dataIndex: 'ts', slotName: 'ts' },
  { title: '值', dataIndex: 'value' }
]

const syntaxColumns = [
  { title: '语法', dataIndex: 'syntax' },
  { title: '说明', dataIndex: 'description' }
]

const syntaxData = [
  { syntax: '<code>v</code> / <code>value</code>', description: '当前点位的实时值' },
  { syntax: '<code>t1</code>, <code>p1</code>', description: '数据源别名引用' },
  { syntax: '<code>bitget(v, n)</code>', description: '获取第 n 位 (0/1)' },
  { syntax: '<code>bitset(v, n)</code>', description: '将第 n 位置 1' },
  { syntax: '<code>bitclr(v, n)</code>', description: '将第 n 位置 0' }
]

const docsColumns = [
  { title: '函数 (Function)', dataIndex: 'function', width: 200 },
  { title: '说明 (Description)', dataIndex: 'description' },
  { title: '示例 (Example)', dataIndex: 'example' }
]

const bitwiseFunctions = [
  { function: '<code>bitand(a, b)</code>', description: '按位与 (Bitwise AND). 对应 <code>a & b</code>', example: '<code>bitand(v, 1)</code> (判断最低位是否为1)' },
  { function: '<code>bitor(a, b)</code>', description: '按位或 (Bitwise OR). 对应 <code>a | b</code>', example: '<code>bitor(v, 4)</code> (将第3位置1)' },
  { function: '<code>bitxor(a, b)</code>', description: '按位异或 (Bitwise XOR). 对应 <code>a ^ b</code>', example: '<code>bitxor(v, 1)</code> (翻转最低位)' },
  { function: '<code>bitnot(a)</code>', description: '按位非 (Bitwise NOT). 对应 <code>~a</code>', example: '<code>bitnot(v)</code> (翻转所有位)' },
  { function: '<code>bitget(v, n)</code>', description: '获取第 n 位 (0-based)', example: '<code>bitget(v, 3)</code> (获取第4位)' },
  { function: '<code>bitset(v, n)</code>', description: '将第 n 位置 1', example: '<code>bitset(v, 3)</code> (将第4位置1)' },
  { function: '<code>bitclr(v, n)</code>', description: '将第 n 位置 0', example: '<code>bitclr(v, 3)</code> (将第4位置0)' }
]

const mathFunctions = [
  { function: '<code>abs(a)</code>', description: '绝对值', example: '<code>abs(v)</code>' },
  { function: '<code>ceil(a)</code>', description: '向上取整', example: '<code>ceil(v)</code>' },
  { function: '<code>floor(a)</code>', description: '向下取整', example: '<code>floor(v)</code>' },
  { function: '<code>round(a)</code>', description: '四舍五入', example: '<code>round(v)</code>' },
  { function: '<code>sqrt(a)</code>', description: '平方根', example: '<code>sqrt(v)</code>' },
  { function: '<code>pow(a, b)</code>', description: '幂运算', example: '<code>pow(v, 2)</code> (平方)' },
  { function: '<code>sin(a)</code>', description: '正弦函数', example: '<code>sin(v)</code>' },
  { function: '<code>cos(a)</code>', description: '余弦函数', example: '<code>cos(v)</code>' },
  { function: '<code>tan(a)</code>', description: '正切函数', example: '<code>tan(v)</code>' }
]

const logicalFunctions = [
  { function: '<code>and(a, b)</code>', description: '逻辑与', example: '<code>and(t1 > 50, t2 > 30)</code>' },
  { function: '<code>or(a, b)</code>', description: '逻辑或', example: '<code>or(t1 > 50, t2 > 30)</code>' },
  { function: '<code>not(a)</code>', description: '逻辑非', example: '<code>not(t1 > 50)</code>' },
  { function: '<code>eq(a, b)</code>', description: '等于', example: '<code>eq(v, 1)</code>' },
  { function: '<code>ne(a, b)</code>', description: '不等于', example: '<code>ne(v, 0)</code>' },
  { function: '<code>gt(a, b)</code>', description: '大于', example: '<code>gt(v, 50)</code>' },
  { function: '<code>ge(a, b)</code>', description: '大于等于', example: '<code>ge(v, 50)</code>' },
  { function: '<code>lt(a, b)</code>', description: '小于', example: '<code>lt(v, 50)</code>' },
  { function: '<code>le(a, b)</code>', description: '小于等于', example: '<code>le(v, 50)</code>' }
]
import { ref, reactive, computed, watch, watchEffect, onMounted, onUnmounted, provide } from 'vue'
import { useRoute } from 'vue-router'
import request from '@/utils/request'
import { base64ToUint8Array, uint8ArrayToHex, detectFileType, downloadBytes } from '@/utils/decode'
import EdgeComputeMetrics from './EdgeComputeMetrics.vue'

// Expression Helper
const helperDialog = ref(false)
const docsDialog = ref(false)
const helperInput = ref('')
const helperOutput = ref('')
let helperCallback = null

const openHelper = (initialValue, callback) => {
    helperInput.value = initialValue || ''
    helperOutput.value = ''
    helperCallback = callback
    helperDialog.value = true
}

const convertHelper = () => {
    let res = helperInput.value
    if (!res) return

    // Handle ~ (Unary NOT)
    res = res.replace(/~\s*([a-zA-Z0-9_.]+|\([^)]+\))/g, 'bitnot($1)')

    let prev = ''
    let limit = 10
    while (prev !== res && limit > 0) {
        prev = res
        limit--
        
        // << and >>
        res = res.replace(/([a-zA-Z0-9_.]+|\([^)]+\))\s*(<<|>>)\s*([a-zA-Z0-9_.]+|\([^)]+\))/g, (m, a, op, b) => {
             return op === '<<' ? `bitshl(${a}, ${b})` : `bitshr(${a}, ${b})`
        })
        
        // &
        res = res.replace(/([a-zA-Z0-9_.]+|\([^)]+\))\s*&\s*([a-zA-Z0-9_.]+|\([^)]+\))/g, 'bitand($1, $2)')
        
        // ^
        res = res.replace(/([a-zA-Z0-9_.]+|\([^)]+\))\s*\^\s*([a-zA-Z0-9_.]+|\([^)]+\))/g, 'bitxor($1, $2)')
        
        // |
        res = res.replace(/([a-zA-Z0-9_.]+|\([^)]+\))\s*\|\s*([a-zA-Z0-9_.]+|\([^)]+\))/g, 'bitor($1, $2)')
    }
    helperOutput.value = res
}

const applyHelper = () => {
    if (helperCallback && helperOutput.value) {
        helperCallback(helperOutput.value)
    }
    helperDialog.value = false
}

const route = useRoute()
const tab = ref('metrics')
const rules = ref([])
const ruleStates = ref([])
const dialog = ref(false)
const helpDialog = ref(false)
const editingRule = ref(false)
// Northbound Config for Actions
const northboundConfig = ref({ mqtt: [], http: [] })

// MQTT Options for Select
const mqttOptions = computed(() => {
    const cfg = northboundConfig.value || {}
    const list = Array.isArray(cfg.mqtt) ? cfg.mqtt : []
    const status = cfg.status || {}

    const options = list
        .filter(item => item.enable)   // ✅ 只要启用的
        .map(item => ({
            label: `${item.name || item.id}${status[item.id] === 3 ? '' : ' (离线)'}`,
            value: item.id,
            disabled: status[item.id] !== 3   // ✅ 工业级：离线不可选
        }))

    // ✅ 强制兜底（关键：保证一定有 Default MQTT）
    if (!options.find(o => o.label.includes('Default MQTT'))) {
        options.unshift({
            label: 'Default MQTT',
            value: 'mqtt-1'
        })
    }

    return options
})

// HTTP Options for Select
const httpOptions = computed(() => {
    const cfg = northboundConfig.value || {}
    const list = Array.isArray(cfg.http) ? cfg.http : []
    const status = cfg.status || {}

    const options = list
        .filter(item => item.enable)   // ✅ 只要启用的
        .map(item => ({
            label: `${item.name || item.id}${status[item.id] === 3 ? '' : ' (离线)'}`,
            value: item.id,
            disabled: status[item.id] !== 3   // ✅ 工业级：离线不可选
        }))

    return options
})

// Provide only UI data
provide('mqttOptions', mqttOptions)
provide('httpOptions', httpOptions)

// Debug
watchEffect(() => {
    console.log('northboundConfig:', northboundConfig.value)
    console.log('mqttOptions:', mqttOptions.value)
    console.log('httpOptions:', httpOptions.value)
})

const selectedRuleKeys = ref([]) // 批量选择存储

const channels = ref([])
const devices = ref([])
const windowDialog = ref(false)
const windowData = ref([])
const currentWindowRuleName = ref('')
let timer = null

const fetchNorthboundConfig = async () => {
    try {
        const data = await request.get('/api/northbound/config')
        if (data) {
            northboundConfig.value = {
                mqtt: data.mqtt || [],
                http: data.http || [],
                status: data.status || {}
            }
        }
    } catch (e) {
        console.error("Failed to fetch northbound config", e)
    }
}

// Details Dialog
const detailsDialog = ref(false)
const detailTitle = ref('')
const detailContent = ref('')
const detailsTab = ref('text')
const decodedHex = ref('')
const detectedFile = ref(null)
const decodedBytes = ref(null)

const showDetails = (title, content) => {
    detailTitle.value = title
    detailContent.value = String(content || '')
    detailsDialog.value = true
    
    // Reset state
    detailsTab.value = 'text'
    decodedHex.value = ''
    detectedFile.value = null
    decodedBytes.value = null
}

const tryDecode = () => {
    try {
        const bytes = base64ToUint8Array(detailContent.value)
        decodedBytes.value = bytes
        decodedHex.value = uint8ArrayToHex(bytes)
        detectedFile.value = detectFileType(bytes)
        
        detailsTab.value = 'hex'
        showMessage('解码成功', 'success')
    } catch (e) {
        showMessage('Base64 解码失败: ' + e.message, 'error')
    }
}

const downloadDetectedFile = () => {
    if (decodedBytes.value && detectedFile.value) {
        downloadBytes(decodedBytes.value, `download.${detectedFile.value.ext}`)
    }
}

const query = reactive({
    start: '',
    end: '',
    ruleId: ''
})
const logs = ref([])

const currentRule = reactive({
    id: '',
    name: '',
    type: 'threshold',
    priority: 0,
    enable: true,
    trigger_mode: 'always',
    check_interval: '',
    sources: [], // Multi-source
    trigger_logic: 'EXPR', // Default to EXPR for custom logic
    condition: '',
    expression: '',
    window: { type: 'sliding', size: '10s', aggr_func: 'avg' },
    state: { duration: '0s', count: 0 },
    actions: []
})

const getStatusColor = (status) => {
    switch (status) {
        case 'ALARM': return 'error'
        case 'WARNING': return 'warning'
        case 'NORMAL': return 'success'
        default: return 'grey'
    }
}

const formatRuleType = (type) => {
    const map = {
        'threshold': 'Threshold (阈值触发)',
        'calculation': 'Calculation (计算公式)',
        'window': 'Window (时间/计数窗口)',
        'state': 'State (状态持续)'
    }
    return map[type] || type
}

const formatTriggerMode = (mode) => {
    const map = {
        'always': 'Always (始终触发)',
        'on_change': 'On Change (仅状态改变时触发)'
    }
    return map[mode] || mode
}

const getRuleTypeExplanation = (type) => {
    const map = {
        'threshold': '当数值满足条件表达式时触发。适用于简单的越限报警。',
        'calculation': '计算新值并输出，始终触发。适用于数据预处理或单位转换。',
        'window': '在指定时间或次数窗口内聚合数据（如求平均值）',
        'state': '当条件持续满足指定时间后触发。适用于防抖动报警。'
    }
    return map[type] || ''
}

const fetchChannels = async () => {
    try {
        const data = await request.get('/api/channels')
        if (data) {
            channels.value = (data || []).map(ch => ({
                label: ch.name || ch.channel_name || ch.id,
                value: ch.id,
                raw: ch
            }))
        }
    } catch (e) {
        console.error(e)
    }
}

// Source Management
const addSource = () => {
    if (!currentRule.sources) currentRule.sources = []
    currentRule.sources.push({
        channel_id: '',
        device_id: '',
        point_id: '',
        alias: '',
        _deviceList: [], // Local state for dropdown
        _pointList: []
    })
}

const removeSource = (index) => {
    currentRule.sources.splice(index, 1)
}

const detectInvalidSources = () => {
    if (!currentRule.sources) return
    
    let invalidCount = 0
    currentRule.sources.forEach((src, index) => {
        const isInvalid = !src.channel_id || !src.device_id || !src.point_id || !src.alias
        if (isInvalid) {
            invalidCount++
        }
    })
    
    if (invalidCount > 0) {
        alert(`检测到 ${invalidCount} 个失效配置，请点击"一键清除"按钮清理`)
    } else {
        alert('所有数据源配置均有效')
    }
}

const clearInvalidSources = () => {
    if (!currentRule.sources) return
    
    const validSources = currentRule.sources.filter(src => 
        src.channel_id && src.device_id && src.point_id && src.alias
    )
    
    const removedCount = currentRule.sources.length - validSources.length
    currentRule.sources = validSources
    
    if (removedCount > 0) {
        alert(`已清除 ${removedCount} 个失效配置`)
    } else {
        alert('没有发现失效配置')
    }
}

const onSourceChannelChange = async (src) => {
    src.device_id = ''
    src.point_id = ''
    src._deviceList = []
    src._pointList = []
    
    if (!src.channel_id) return
    
    try {
        const data = await request.get(`/api/channels/${src.channel_id}/devices`)
        if (data) {
            src._deviceList = (Array.isArray(data) ? data : []).map(d => ({
                label: d.name || d.device_name || d.id,
                value: d.id,
                raw: d
            }))
        }
    } catch (e) {
        console.error(e)
    }
}

const onSourceDeviceChange = (src) => {
    src.point_id = ''
    src._pointList = []
    updateSourcePointList(src)
}

const updateSourcePointList = (src) => {
    if (!src.device_id || !src._deviceList || !Array.isArray(src._deviceList)) return
    const dev = src._deviceList.find(d => d.value === src.device_id)
    if (dev && dev.raw && dev.raw.points) {
        src._pointList = (dev.raw.points || []).map(p => ({
            label: p.name || p.point_name || p.id,
            value: p.id,
            raw: p
        }))
    } else {
        src._pointList = []
    }
}

const loadSourceDevices = async (src) => {
    if (!src.channel_id || (src._deviceList && src._deviceList.length > 0)) return
    await onSourceChannelChange(src)
}

const loadSourcePoints = (src) => {
    // No-op or alias for updateSourcePointList, but strictly we don't need to load from network
    // providing we have device list.
    if (!src._pointList || src._pointList.length === 0) {
        updateSourcePointList(src)
    }
}

const fetchRules = async () => {
    try {
        const data = await request.get('/api/edge/rules')
        if (data) {
            rules.value = data
        }
    } catch (e) {
        console.error(e)
    }
}

const fetchRuleStates = async () => {
    try {
        const data = await request.get('/api/edge/states')
        if (data) {
            ruleStates.value = Object.values(data)
        }
    } catch (e) {
        console.error(e)
    }
}

const queryLogs = async () => {
    try {
        const params = new URLSearchParams()
        if (query.start) params.append('start_date', query.start.replace('T', ' '))
        if (query.end) params.append('end_date', query.end.replace('T', ' '))
        if (query.ruleId) params.append('rule_id', query.ruleId)
        
        const data = await request.get(`/api/edge/logs?${params.toString()}`)
        logs.value = data || []
    } catch (e) {
        console.error("Failed to query logs", e)
    }
}

const exportLogs = async () => {
    if (!logs.value || logs.value.length === 0) return

    const headers = ['Time', 'Rule ID', 'Rule Name', 'Status', 'Trigger Count', 'Value', 'Error']
    const csvContent = [
        headers.join(','),
        ...logs.value.map(log => [
            log.minute,
            log.rule_id,
            log.rule_name,
            log.status,
            log.trigger_count,
            log.last_value,
            `"${(log.error_message || '').replace(/"/g, '""')}"`
        ].join(','))
    ].join('\n')

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    if (link.download !== undefined) {
        const url = URL.createObjectURL(blob)
        link.setAttribute('href', url)
        link.setAttribute('download', `edge_logs_${new Date().toISOString().slice(0,19).replace(/[:T]/g, '-')}.csv`)
        link.style.visibility = 'hidden'
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
    }
}

const viewWindowData = async (ruleId, ruleName) => {
    currentWindowRuleName.value = ruleName
    windowData.value = []
    windowDialog.value = true
    try {
        const data = await request.get(`/api/edge/rules/${ruleId}/window`)
        if (data) {
            windowData.value = data
        }
    } catch (e) {
        console.error(e)
    }
}

const addAction = () => {
    if (!currentRule.actions) currentRule.actions = []
    currentRule.actions.push({
        type: 'log',
        config: {}
    })
}

const removeAction = (index) => {
    currentRule.actions.splice(index, 1)
}

const detectInvalidActions = () => {
    if (!currentRule.actions) return
    
    let invalidCount = 0
    currentRule.actions.forEach(action => {
        const isInvalid = isActionInvalid(action)
        if (isInvalid) {
            invalidCount++
        }
    })
    
    if (invalidCount > 0) {
        alert(`检测到 ${invalidCount} 个失效动作配置，请点击"一键清除"按钮清理`)
    } else {
        alert('所有动作配置均有效')
    }
}

const clearInvalidActions = () => {
    if (!currentRule.actions) return
    
    const validActions = currentRule.actions.filter(action => !isActionInvalid(action))
    const removedCount = currentRule.actions.length - validActions.length
    currentRule.actions = validActions
    
    if (removedCount > 0) {
        alert(`已清除 ${removedCount} 个失效动作配置`)
    } else {
        alert('没有发现失效动作配置')
    }
}

const isActionInvalid = (action) => {
    if (!action.type) return true
    
    switch (action.type) {
        case 'mqtt':
            return !action.config.mqtt_id
        case 'http':
            return !action.config.http_id
        case 'device_control':
            return !action.config.channel_id || !action.config.device_id || !action.config.point_id
        case 'check':
            return !action.config.condition
        case 'delay':
            return !action.config.duration
        case 'sequence':
            return !action.config.steps || !Array.isArray(action.config.steps) || action.config.steps.length === 0
        default:
            return false
    }
}

const openDialog = () => {
    editingRule.value = false
    // Reset
    currentRule.id = ''
    currentRule.name = ''
    currentRule.type = 'threshold'
    currentRule.priority = 0
    currentRule.enable = true
    currentRule.trigger_mode = 'always'
    currentRule.sources = [] // Reset sources
    currentRule.trigger_logic = 'OR'
    currentRule.condition = ''
    currentRule.expression = ''
    currentRule.window = { type: 'sliding', size: '10s', aggr_func: 'avg' }
    currentRule.state = { duration: '0s', count: 0 }
    currentRule.actions = []
    
    // Add one empty source by default
    addSource()
    
    dialog.value = true
}

const openHelpDialog = () => {
    helpDialog.value = true
}

const editRule = async (rule) => {
    editingRule.value = true
    // Deep copy
    const r = JSON.parse(JSON.stringify(rule))
    Object.assign(currentRule, r)
    
    // Ensure nested objects
    if (!currentRule.sources) currentRule.sources = []
    // Backward compatibility: If source exists but sources is empty
    if (currentRule.sources.length === 0 && r.source && r.source.channel_id) {
        currentRule.sources.push({
            channel_id: r.source.channel_id,
            device_id: r.source.device_id,
            point_id: r.source.point_id,
            alias: r.source.alias || 'val'
        })
    }
    
    if (!currentRule.actions) currentRule.actions = []

    if (!currentRule.window) currentRule.window = { type: 'sliding', size: '10s', aggr_func: 'avg' }
    if (!currentRule.state) currentRule.state = { duration: '0s', count: 0 }
    
    // Load metadata for sources (devices/points list)
    for (const src of currentRule.sources) {
        if (src.channel_id) {
            src._deviceList = await fetchDevices(src.channel_id)
            if (src.device_id) {
                updateSourcePointList(src)
            }
        }
    }
    
    dialog.value = true
}

const deleteRule = async (rule) => {
    if (!confirm('确定删除该规则吗？')) return
    try {
        await request.delete(`/api/edge/rules/${rule.id}`)
        fetchRules()
    } catch (e) {
        alert('删除失败')
    }
}

const saveRule = async () => {
    try {
        await request.post('/api/edge/rules', currentRule)
        dialog.value = false
        fetchRules()
    } catch (e) {
        alert('保存失败: ' + e.message)
    }
}

// 批量操作函数
const handleBatchDelete = async () => {
    if (selectedRuleKeys.value.length === 0) return
    if (!confirm(`确定删除选中的 ${selectedRuleKeys.value.length} 条规则吗？`)) return
    
    try {
        await Promise.all(selectedRuleKeys.value.map(id => request.delete(`/api/edge/rules/${id}`)))
        selectedRuleKeys.value = []
        fetchRules()
        showMessage('批量删除成功', 'success')
    } catch (e) {
        showMessage('批量删除失败: ' + e.message, 'error')
    }
}

const handleBatchEnable = async (status) => {
    if (selectedRuleKeys.value.length === 0) return
    
    try {
        await Promise.all(selectedRuleKeys.value.map(id => request.put(`/api/edge/rules/${id}`, { enable: status })))
        selectedRuleKeys.value = []
        fetchRules()
        showMessage(`批量${status ? '启用' : '禁用'}成功`, 'success')
    } catch (e) {
        showMessage(`批量${status ? '启用' : '禁用'}失败: ` + e.message, 'error')
    }
}

const showMessage = (message, type = 'info') => {
    // 这里可以根据实际使用的消息组件进行调整
    console.log(`[${type.toUpperCase()}] ${message}`)
}

const getLogicColor = (logic) => {
    switch(logic) {
        case 'AND': return 'info'
        case 'OR': return 'warning'
        case 'EXPR': return 'purple'
        default: return 'grey'
    }
}

const formatDate = (ts) => {
    if (!ts || ts === '0001-01-01T00:00:00Z') return '-'
    return new Date(ts).toLocaleString()
}

const calculateDuration = (startTs) => {
    if (!startTs || startTs === '0001-01-01T00:00:00Z') return '-'
    const start = new Date(startTs).getTime()
    const now = new Date().getTime()
    const diff = Math.floor((now - start) / 1000)
    
    if (diff < 60) return `${diff}s`
    if (diff < 3600) return `${Math.floor(diff/60)}m ${diff%60}s`
    return `${Math.floor(diff/3600)}h ${Math.floor((diff%3600)/60)}m`
}

// Helper to fetch devices
const fetchDevices = async (channelId) => {
    if (!channelId) return []
    try {
        const data = await request.get(`/api/channels/${channelId}/devices`)
        if (data) {
            return data
        }
    } catch (e) {
        console.error(e)
    }
    return []
}

// Action Helper Functions removed - moved to ActionEditor component

onMounted(async () => {
    await fetchRules()
    fetchChannels()
    fetchRuleStates()
    fetchNorthboundConfig()
    // Poll status every 5 seconds
    timer = setInterval(fetchRuleStates, 5000)

    if (route.query.rule) {
        const rule = rules.value.find(r => r.id === route.query.rule)
        if (rule) {
            editRule(rule)
        }
    }
})

onUnmounted(() => {
    if (timer) clearInterval(timer)
})
</script>

<style scoped>
.edge-compute-container {
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 16px;
    box-sizing: border-box;
}

.single-line-cell {
    white-space: nowrap;
    overflow-x: auto;
    max-width: 400px;
    display: block;
    cursor: pointer;
    font-family: monospace;
}

.single-line-cell:hover {
    color: #111827;
    background-color: #f9fafb;
}

/* 表格容器允许横向滚动 */
.logs-table-container {
    overflow-x: auto;
}

/* 全局表格强制单行 */
:deep(.arco-table-td),
:deep(.arco-table-th) {
    white-space: nowrap;
}

/* 统一卡片风格 */
.block-card {
    border: 1px solid #e5e7eb;
    border-radius: 0;
    margin-bottom: 12px;
}

:deep(.arco-card-header) {
    padding: 8px 12px;
    font-size: 12px;
    border-bottom: 1px solid #f1f3f5;
}

/* 工业风格卡片样式 */
:deep(.arco-card) {
    border: 1px solid #e5e7eb;
    border-radius: 0;
    background: #ffffff;
    position: relative;
}

:deep(.arco-card::after) {
    content: '';
    position: absolute;
    bottom: 0;
    left: 0;
    width: 100%;
    height: 1px;
    background: #0f172a;
    opacity: 0.05;
}

:deep(.arco-card:hover) {
    box-shadow: none !important;
    border-color: #111827;
}

/* 表格工业风格样式 */
:deep(.arco-table-th) {
    background: #fafafa;
    border-bottom: 1px solid #e5e7eb;
    font-size: 11px;
    color: #6b7280;
    font-weight: 500;
}

:deep(.arco-table-td) {
    font-size: 12px;
    border-bottom: 1px solid #f1f3f5;
}

:deep(.arco-table-tr:hover .arco-table-td) {
    background: #f9fafb;
}

/* 标签工业风格样式 */
:deep(.arco-tag) {
    border-radius: 0;
    font-size: 11px;
    padding: 2px 6px;
}

/* 按钮工业风格样式 */
:deep(.arco-button) {
    border-radius: 0;
}

/* 输入框工业风格样式 */
:deep(.arco-input),
:deep(.arco-select) {
    border-radius: 0;
}

/* 表单标签水平显示 */
:deep(.arco-form-item-label) {
    white-space: nowrap;
    text-align: left;
    font-size: 12px;
    color: #475569;
    font-weight: 500;
}

:deep(.arco-form-item-label-col) {
    display: flex;
    align-items: center;
}

:deep(.arco-form-item) {
    margin-bottom: 0;
}

/* 工业白色风格样式 */
:deep(.industrial-white-modal .arco-modal) {
    border-radius: 2px;
    padding: 0;
}

.form-section {
    margin-bottom: 24px;
}

.section-title {
    font-size: 11px;
    font-weight: bold;
    color: #94a3b8;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin-bottom: 12px;
    border-left: 3px solid #0f172a;
    padding-left: 8px;
}

.section-header-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
}

.action-card {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    padding: 16px;
    margin-bottom: 12px;
}

/* 强制直角 */
.rect-input {
    border-radius: 0 !important;
}

/* 输入组样式修正 */
:deep(.arco-input-group) {
    width: 100%;
}

/* 批量操作工具栏：白色悬浮条 */
.table-toolbar-industrial {
    background: #ffffff;
    border: 1px solid #10b981;
    padding: 8px 16px;
    margin-bottom: 12px;
    box-shadow: 0 2px 8px rgba(16, 185, 129, 0.1);
    display: flex;
    justify-content: flex-end;
    align-items: center;
}

/* 表格规范样式 */
.industrial-table {
    border-radius: 0 !important;
    border: 1px solid #e5e7eb !important;
}

/* 表头样式 */
.industrial-table :deep(.arco-table-th) {
    background-color: #f8fafc !important;
    font-weight: bold !important;
    border-radius: 0 !important;
    padding: 12px !important;
    white-space: nowrap !important;
    overflow: visible !important;
}

/* 单元格样式 */
.industrial-table :deep(.arco-table-td) {
    padding: 12px !important;
    white-space: nowrap !important;
    overflow: hidden !important;
    text-overflow: ellipsis !important;
    border-radius: 0 !important;
}

/* 行高 */
.industrial-table :deep(.arco-table-tr) {
    height: 36px !important;
}

/* 悬停效果 */
.industrial-table :deep(.arco-table-tr:hover) {
    background-color: #f9fafb !important;
}

/* 选中状态 */
.industrial-table :deep(.arco-table-tr.arco-table-tr-selected) {
    background-color: #eff6ff !important;
    border: 1px solid #bfdbfe !important;
}

/* 操作列固定在右侧 */
.industrial-table :deep(.arco-table-col-fixed-right) {
    position: sticky !important;
    right: 0 !important;
    z-index: 1 !important;
    background-color: #ffffff !important;
}

/* 消除表格最后一列的右边框 */
.industrial-table :deep(.arco-table-col-fixed-right .arco-table-td) {
    border-right: none !important;
}

/* 固定列的左边框 */
.industrial-table :deep(.arco-table-col-fixed-right .arco-table-td) {
    border-left: 1px solid #e5e7eb !important;
}

/* 消除表格的边框 */
.industrial-table :deep(.arco-table-container) {
    border-right: none !important;
    border-left: none !important;
}

/* 固定列样式 */
.industrial-table :deep(.arco-table-col-fixed-left .arco-table-th) {
    background-color: #f8fafc !important;
    border-right: 1px solid #e5e7eb !important;
}

.industrial-table :deep(.arco-table-col-fixed-left .arco-table-td) {
    background-color: #ffffff !important;
    border-right: 1px solid #e5e7eb !important;
}

.industrial-table :deep(.arco-table-col-fixed-right .arco-table-th) {
    background-color: #f8fafc !important;
    border-left: 1px solid #e5e7eb !important;
}

.industrial-table :deep(.arco-table-td.arco-table-td-row-select) {
    background-color: #ffffff !important;
    border-right: 1px solid #e5e7eb !important;
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

.selection-count {
    font-size: 12px;
    font-weight: bold;
    color: #10b981;
}

.flex {
    display: flex;
}

.items-center {
    align-items: center;
}

.gap-2 {
    gap: 8px;
}

/* 卡片标题样式 */
:deep(.arco-card-header-title) {
    font-size: 12px;
    font-weight: 600;
    color: #374151;
    letter-spacing: 0.5px;
}

/* 数据源样式 */
.source-row {
    display: flex;
    align-items: center;
    border-bottom: 1px solid #f1f3f5;
    padding: 8px 0;
}

.source-index {
    width: 32px;
    font-size: 11px;
    color: #9ca3af;
}

/* 动作行样式 */
.action-row {
    display: flex;
    align-items: flex-start;
    border-left: 2px solid #e5e7eb;
    padding-left: 12px;
    margin-bottom: 12px;
}

.action-index {
    font-size: 10px;
    color: #6b7280;
    width: 60px;
}

.action-body {
    flex: 1;
}

/* 代码输入框样式 */
.code-input {
    font-family: monospace;
    font-size: 12px;
}

/* 逻辑工具栏样式 */
.logic-toolbar {
    margin-top: 8px;
}

/* 空状态样式 */
.empty-box {
    padding: 24px;
    text-align: center;
    color: #9ca3af;
    font-size: 12px;
    background: #f9fafb;
    border: 1px dashed #e5e7eb;
}
</style>
