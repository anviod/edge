package main

import (
	"context"
	"edge-gateway/internal/driver/bacnet"
	"edge-gateway/internal/model"
	"fmt"
	"time"
)

func main() {
	// 1. 初始化驱动
	d := bacnet.NewBACnetDriver().(*bacnet.BACnetDriver)
	// 驱动配置：绑定到 0.0.0.0:47808，因为设备使用单播响应
	config := model.DriverConfig{
		Config: map[string]any{
			"interface_ip":   "0.0.0.0",
			"interface_port": 47808, // 驱动监听端口
		},
	}
	if err := d.Init(config); err != nil {
		fmt.Printf("Driver Init Failed: %v\n", err)
		return
	}
	if err := d.Connect(context.Background()); err != nil {
		fmt.Printf("Driver Connect Failed: %v\n", err)
		return
	}
	defer d.Disconnect()

	// 2. 定义目标设备列表（从 conf/channels.yaml 和 conf/devices/... 中提取）
	// 注意：所有设备都在 192.168.3.112 上，但端口不同。Instance ID 也不同。
	// 验证点：Setpoint.1 -> AnalogValue:1 -> 期望值分别为 316, 317, 318, 319
	devices := []struct {
		ID         string
		Name       string
		InstanceID int
		IP         string
		Port       int
		Expected   float32
	}{
		{
			ID:         "bacnet-16",
			Name:       "Device 16",
			InstanceID: 2228316,
			IP:         "192.168.3.112",
			Port:       63501,
			Expected:   316.0,
		},
		{
			ID:         "bacnet-17",
			Name:       "Device 17",
			InstanceID: 2228317,
			IP:         "192.168.3.112",
			Port:       63502,
			Expected:   317.0,
		},
		{
			ID:         "bacnet-18",
			Name:       "Device 18",
			InstanceID: 2228318,
			IP:         "192.168.3.112",
			Port:       63503,
			Expected:   318.0,
		},
		{
			ID:         "Room_FC_2014_19",
			Name:       "Room_FC_2014_19",
			InstanceID: 2228319,
			IP:         "192.168.3.112",
			Port:       57611,
			Expected:   319.0,
		},
	}

	// 3. 逐个验证设备读取
	fmt.Println("=== Starting BACnet Verification Test ===")
	fmt.Println("Target: 4 Devices on 192.168.3.112")
	fmt.Println("Point:  AnalogValue:1 (Setpoint.1)")
	fmt.Println("Check:  Value match Expected (316/317/318/319)")
	fmt.Println("------------------------------------------------")

	allPassed := true

	for _, devDef := range devices {
		fmt.Printf("\n[Testing Device: %s (Instance: %d)]\n", devDef.ID, devDef.InstanceID)

		// 3.1 构造设备模型并注入配置
		dev := &model.Device{
			ID:   devDef.ID,
			Name: devDef.Name,
			Config: map[string]any{
				"instance_id":         devDef.InstanceID,
				"ip":                  devDef.IP,
				"port":                devDef.Port,
				"_internal_device_id": devDef.ID, // 关键：注入内部 ID 映射
			},
			Points: []model.Point{
				{
					ID:       "Setpoint.1",
					DeviceID: devDef.ID,
					Address:  "AnalogValue:1",
					DataType: "float32",
				},
			},
		}

		// 3.2 调用 SetDeviceConfig 建立映射
		// 这会触发 discoverDevice (WhoIs)，如果设备在线，会建立 Context
		if err := d.SetDeviceConfig(dev.Config); err != nil {
			fmt.Printf("❌ SetDeviceConfig Failed: %v\n", err)
			allPassed = false
			continue
		}

		// 等待发现完成（WhoIs 是异步的，但在 SetDeviceConfig 内部如果是首次可能会阻塞一下或者直接返回）
		// 为了稳妥，稍微等一下，或者依赖 ReadPoints 的重试/等待机制（如果有的话）
		// 实际上 SetDeviceConfig 触发的是异步发现，但 ReadPoints 会检查 Context 是否存在。
		// 我们可以手动等一下，确保发现完成。
		fmt.Print("  Waiting for discovery...")
		time.Sleep(2 * time.Second) 
		fmt.Println(" Done.")

		// 3.3 读取点位
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		results, err := d.ReadPoints(ctx, dev.Points)
		cancel()

		if err != nil {
			fmt.Printf("❌ ReadPoints Failed: %v\n", err)
			allPassed = false
			continue
		}

		// 3.4 验证结果
		val, ok := results["Setpoint.1"]
		if !ok {
			fmt.Printf("❌ Point Not Found in Results\n")
			allPassed = false
			continue
		}

		// 解析值
		var floatVal float32
		switch v := val.Value.(type) {
		case float32:
			floatVal = v
		case float64:
			floatVal = float32(v)
		default:
			fmt.Printf("⚠️ Unexpected Type: %T (%v)\n", v, v)
			// 尝试继续
		}

		fmt.Printf("  Read Value: %.2f (Expected: %.2f)\n", floatVal, devDef.Expected)

		if floatVal == devDef.Expected {
			fmt.Printf("✅ Verification PASSED for %s\n", devDef.ID)
		} else {
			fmt.Printf("❌ Verification FAILED for %s: Value mismatch\n", devDef.ID)
			allPassed = false
		}
	}

	fmt.Println("\n------------------------------------------------")
	if allPassed {
		fmt.Println("🎉 All Devices Verified Successfully!")
	} else {
		fmt.Println("⚠️ Some Verifications Failed.")
	}
}
