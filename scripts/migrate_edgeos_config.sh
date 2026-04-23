#!/bin/bash
# EdgeOS 设备配置迁移工具
# 用于将旧的设备配置格式(map[string]bool)迁移到新格式(map[string]DevicePublishConfig)

CONFIG_FILE="$1"

if [ -z "$CONFIG_FILE" ]; then
    echo "用法: $0 <config_file.yaml>"
    echo ""
    echo "示例:"
    echo "  $0 conf/northbound.yaml"
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo "错误: 配置文件不存在: $CONFIG_FILE"
    exit 1
fi

# 创建备份
BACKUP_FILE="${CONFIG_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
echo "备份原配置到: $BACKUP_FILE"
cp "$CONFIG_FILE" "$BACKUP_FILE"

# 使用 Python 脚本进行配置迁移
python3 << 'EOF'
import sys
import re
from datetime import datetime

config_file = sys.argv[1]

try:
    with open(config_file, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 检测需要迁移的模式
    # 旧格式: device-001: true
    # 新格式: device-001:\n    enable: true\n    strategy: "realtime"\n    interval: "0s"
    
    changes_made = False
    
    # 匹配 edgeos_mqtt 配置块
    edgeos_mqtt_pattern = r'(edgeos_mqtt:.*?devices:\s*\n)((?:\s+\S+:\s*(?:true|false)\s*\n)+)'
    
    def migrate_devices_block(match):
        prefix = match.group(1)
        devices_block = match.group(2)
        
        new_devices = []
        for line in devices_block.split('\n'):
            if not line.strip():
                continue
            
            # 匹配: device-001: true
            device_match = re.match(r'(\s+)(\S+):\s*(true|false)\s*', line)
            if device_match:
                indent = device_match.group(1)
                device_id = device_match.group(2)
                enabled = device_match.group(3)
                
                # 生成新格式
                new_devices.append(f"{indent}{device_id}:")
                new_devices.append(f"{indent}  enable: {enabled}")
                new_devices.append(f"{indent}  strategy: \"realtime\"")
                new_devices.append(f"{indent}  interval: \"0s\"")
                new_devices.append("")
                changes_made = True
            else:
                new_devices.append(line)
        
        return prefix + '\n'.join(new_devices)
    
    # 匹配 edgeos_nats 配置块
    edgeos_nats_pattern = r'(edgeos_nats:.*?devices:\s*\n)((?:\s+\S+:\s*(?:true|false)\s*\n)+)'
    
    content = re.sub(edgeos_mqtt_pattern, migrate_devices_block, content, flags=re.DOTALL)
    content = re.sub(edgeos_nats_pattern, migrate_devices_block, content, flags=re.DOTALL)
    
    # 写回文件
    with open(config_file, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print("✅ 配置迁移完成!")
    print("📝 主要变更:")
    print("  - 设备配置格式从 map[string]bool 升级为 DevicePublishConfig")
    print("  - 每个设备配置增加: strategy (realtime/periodic) 和 interval 字段")
    print("  - 默认策略: realtime (立即推送)")
    print("  - 默认周期: 0s (实时模式)")
    
EOF

if [ $? -eq 0 ]; then
    echo ""
    echo "迁移完成! 新配置已保存到: $CONFIG_FILE"
    echo "原配置已备份到: $BACKUP_FILE"
    echo ""
    echo "下一步:"
    echo "1. 检查新配置是否正确"
    echo "2. 根据需求调整设备的 strategy 和 interval 设置"
    echo "3. 重启服务使配置生效"
else
    echo "迁移失败! 恢复原配置..."
    mv "$BACKUP_FILE" "$CONFIG_FILE"
    exit 1
fi
