package main

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	nodeID     = "edgex-node-001"
	broker     = "tcp://127.0.0.1:1883"
	clientID   = "edgex-integration-test"
)

type MessageHeader struct {
	MessageID     string `json:"message_id"`
	Timestamp     int64  `json:"timestamp"`
	Source        string `json:"source"`
	Destination   string `json:"destination,omitempty"`
	MessageType   string `json:"message_type"`
	Version       string `json:"version"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

type Message struct {
	Header MessageHeader `json:"header"`
	Body   any          `json:"body"`
}

var (
	receivedMessages = make(chan Message, 10)
	connected        = make(chan bool, 1)
	disconnected     = make(chan bool, 1)
)

func main() {
	fmt.Println("=== EdgeX-EdgeOS MQTT Integration Test ===")
	fmt.Println()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetConnectTimeout(10 * time.Second)

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		fmt.Println("[CONNECT] Connected to MQTT broker")
		connected <- true
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		fmt.Printf("[DISCONNECT] Connection lost: %v\n", err)
		disconnected <- true
	})

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("[RECV] Default handler - topic: %s\n", msg.Topic())
	})

	client := mqtt.NewClient(opts)

	fmt.Printf("[CONNECT] Connecting to %s...\n", broker)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("[ERROR] Connection failed: %v\n", token.Error())
		return
	}

	// Wait for connection
	select {
	case <-connected:
		fmt.Println("[OK] Connection established")
	case <-time.After(10 * time.Second):
		fmt.Println("[ERROR] Connection timeout")
		return
	}

	// Run tests
	testNodeRegistration(client)
	testDeviceReport(client)
	testPointReport(client)
	testPointSync(client)
	testRealTimeData(client)
	testHeartbeat(client)
	testSubscribeCommand(client)

	// Cleanup
	fmt.Println()
	fmt.Println("[CLEANUP] Disconnecting...")
	client.Disconnect(2000)

	fmt.Println()
	fmt.Println("=== Integration Test Complete ===")
}

func testNodeRegistration(client mqtt.Client) {
	fmt.Println()
	fmt.Println("--- Test 3.1.1: Node Registration ---")

	responseTopic := fmt.Sprintf("edgex/nodes/%s/response", nodeID)
	fmt.Printf("[SUBSCRIBE] Response topic: %s\n", responseTopic)

	token := client.Subscribe(responseTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		var message Message
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			fmt.Printf("[ERROR] Failed to parse response: %v\n", err)
			return
		}
		fmt.Printf("[RECV] Registration response - type: %s, source: %s\n",
			message.Header.MessageType, message.Header.Source)
		receivedMessages <- message
	})
	token.Wait()

	// Subscribe to online status
	onlineTopic := fmt.Sprintf("edgex/nodes/%s/online", nodeID)
	token = client.Subscribe(onlineTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		fmt.Printf("[RECV] Online status: %s\n", string(msg.Payload()))
	})
	token.Wait()

	// Publish registration
	regMessage := Message{
		Header: MessageHeader{
			MessageID:   generateMsgID(),
			Timestamp:   time.Now().UnixMilli(),
			Source:      nodeID,
			MessageType: "node_register",
			Version:     "1.0",
		},
		Body: map[string]any{
			"node_id":     nodeID,
			"node_name":   "EdgeX Gateway Node",
			"model":       "edge-gateway",
			"version":     "1.0.0",
			"api_version": "v1",
			"capabilities": []string{"shadow-sync", "heartbeat", "device-control", "task-execution"},
			"protocol":    "edgeOS(MQTT)",
			"endpoint": map[string]string{
				"host": "127.0.0.1",
				"port": "8082",
			},
		},
	}

	payload, _ := json.Marshal(regMessage)
	topic := "edgex/nodes/register"
	fmt.Printf("[PUBLISH] Topic: %s\n", topic)
	token = client.Publish(topic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("[ERROR] Publish failed: %v\n", token.Error())
	} else {
		fmt.Println("[OK] Registration message published")
	}

	// Wait for response
	select {
	case msg := <-receivedMessages:
		fmt.Printf("[OK] Received response: %s\n", msg.Header.MessageType)
		if body, ok := msg.Body.(map[string]any); ok {
			if status, ok := body["status"].(string); ok {
				fmt.Printf("[OK] Registration status: %s\n", status)
			}
		}
	case <-time.After(5 * time.Second):
		fmt.Println("[TIMEOUT] No response received (EdgeOS may not be running)")
	}
}

func testDeviceReport(client mqtt.Client) {
	fmt.Println()
	fmt.Println("--- Test 3.1.2: Device Report ---")

	// Note: Device report is published by EdgeX after registration
	// We'll verify by subscribing to the topic
	reportTopic := "edgex/devices/report"
	fmt.Printf("[SUBSCRIBE] Report topic: %s\n", reportTopic)

	token := client.Subscribe(reportTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		var message Message
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			fmt.Printf("[ERROR] Failed to parse device report: %v\n", err)
			return
		}
		fmt.Printf("[RECV] Device report - type: %s, source: %s\n",
			message.Header.MessageType, message.Header.Source)

		if body, ok := message.Body.(map[string]any); ok {
			if devices, ok := body["devices"].([]any); ok {
				fmt.Printf("[INFO] Device count in report: %d\n", len(devices))
			}
		}
	})
	token.Wait()

	// For this test, we'll publish a sample device report to verify format
	sampleReport := Message{
		Header: MessageHeader{
			MessageID:   generateMsgID(),
			Timestamp:   time.Now().UnixMilli(),
			Source:      nodeID,
			MessageType: "device_report",
			Version:     "1.0",
		},
		Body: map[string]any{
			"node_id": nodeID,
			"devices": []map[string]any{
				{
					"device_id":       "test-device-001",
					"device_name":     "Test Device",
					"device_profile":  "modbus",
					"service_name":    "Test Channel",
					"labels":          []string{},
					"description":     "",
					"admin_state":    "ENABLED",
					"operating_state": "ENABLED",
					"properties": map[string]any{
						"protocol":   "modbus",
						"channel_id": "channel-001",
					},
				},
			},
		},
	}

	payload, _ := json.Marshal(sampleReport)
	fmt.Printf("[PUBLISH] Publishing sample device report\n")
	token = client.Publish(reportTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("[ERROR] Publish failed: %v\n", token.Error())
	} else {
		fmt.Println("[OK] Device report published (self-test)")
	}

	time.Sleep(2 * time.Second)
}

func testPointReport(client mqtt.Client) {
	fmt.Println()
	fmt.Println("--- Test 3.1.3: Point Metadata Report ---")

	reportTopic := "edgex/points/report"
	fmt.Printf("[SUBSCRIBE] Report topic: %s\n", reportTopic)

	token := client.Subscribe(reportTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		var message Message
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			fmt.Printf("[ERROR] Failed to parse point report: %v\n", err)
			return
		}
		fmt.Printf("[RECV] Point report - type: %s, source: %s\n",
			message.Header.MessageType, message.Header.Source)

		if body, ok := message.Body.(map[string]any); ok {
			if points, ok := body["points"].([]any); ok {
				fmt.Printf("[INFO] Points in report: %d\n", len(points))
				for i, p := range points {
					if point, ok := p.(map[string]any); ok {
						pointID, _ := point["point_id"].(string)
						pointName, _ := point["point_name"].(string)
						dataType, _ := point["data_type"].(string)
						fmt.Printf("[INFO]   [%d] %s (%s) - %s\n", i+1, pointID, pointName, dataType)
					}
				}
			}
		}
	})
	token.Wait()

	// Publish test point report (metadata)
	pointReport := Message{
		Header: MessageHeader{
			MessageID:   generateMsgID(),
			Timestamp:   time.Now().UnixMilli(),
			Source:      nodeID,
			MessageType: "point_report",
			Version:     "1.0",
		},
		Body: map[string]any{
			"node_id":   nodeID,
			"device_id": "test-device-001",
			"points": []map[string]any{
				{
					"point_id":     "SupplyWaterTemp",
					"point_name":   "供水温度",
					"data_type":    "Float32",
					"access_mode":  "R",
					"unit":         "°C",
					"minimum":      -50.0,
					"maximum":       150.0,
					"address":       "AI-30001",
					"description":  "AHU Supply Water Temperature Sensor",
					"scale":         0.1,
					"offset":        0,
				},
				{
					"point_id":     "ReturnWaterTemp",
					"point_name":   "回水温度",
					"data_type":    "Float32",
					"access_mode":  "R",
					"unit":         "°C",
					"minimum":      -50.0,
					"maximum":       150.0,
					"address":       "AI-30002",
					"description":  "AHU Return Water Temperature Sensor",
					"scale":         0.1,
					"offset":        0,
				},
				{
					"point_id":     "ValvePosition",
					"point_name":   "阀门开度",
					"data_type":    "Float32",
					"access_mode":  "RW",
					"unit":         "%",
					"minimum":      0.0,
					"maximum":       100.0,
					"address":       "AO-30001",
					"description":  "Control Valve Position",
					"scale":         1.0,
					"offset":        0,
				},
			},
		},
	}

	payload, _ := json.Marshal(pointReport)
	fmt.Printf("[PUBLISH] Publishing point metadata report to %s\n", reportTopic)
	token = client.Publish(reportTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("[ERROR] Publish failed: %v\n", token.Error())
	} else {
		fmt.Println("[OK] Point metadata report published")
	}

	time.Sleep(2 * time.Second)
}

func testPointSync(client mqtt.Client) {
	fmt.Println()
	fmt.Println("--- Test 3.1.4: Device Point Value Sync ---")

	deviceID := "test-device-001"
	pointSyncTopic := fmt.Sprintf("edgex/points/%s/%s", nodeID, deviceID)
	fmt.Printf("[SUBSCRIBE] Point sync topic: %s\n", pointSyncTopic)

	token := client.Subscribe(pointSyncTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		var message Message
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			fmt.Printf("[ERROR] Failed to parse point sync: %v\n", err)
			return
		}
		fmt.Printf("[RECV] Point sync - type: %s\n", message.Header.MessageType)

		if body, ok := message.Body.(map[string]any); ok {
			if points, ok := body["points"].(map[string]any); ok {
				fmt.Printf("[INFO] Points received: %d\n", len(points))
				for k, v := range points {
					fmt.Printf("[INFO]   %s = %v\n", k, v)
				}
			}
		}
	})
	token.Wait()

	// Publish test point sync
	pointSync := Message{
		Header: MessageHeader{
			MessageID:   generateMsgID(),
			Timestamp:   time.Now().UnixMilli(),
			Source:      nodeID,
			MessageType: "point_sync",
			Version:     "1.0",
		},
		Body: map[string]any{
			"node_id":   nodeID,
			"device_id": deviceID,
			"timestamp": time.Now().UnixMilli(),
			"points": map[string]any{
				"Temperature": map[string]any{
					"value":    25.5,
					"quality":  "good",
					"timestamp": time.Now().UnixMilli() - 5000,
				},
				"Humidity": map[string]any{
					"value":    60.0,
					"quality":  "good",
					"timestamp": time.Now().UnixMilli() - 5000,
				},
				"Pressure": map[string]any{
					"value":    1013.25,
					"quality":  "good",
					"timestamp": time.Now().UnixMilli() - 5000,
				},
			},
			"quality": "good",
		},
	}

	payload, _ := json.Marshal(pointSync)
	fmt.Printf("[PUBLISH] Publishing point sync to %s\n", pointSyncTopic)
	token = client.Publish(pointSyncTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("[ERROR] Publish failed: %v\n", token.Error())
	} else {
		fmt.Println("[OK] Point sync published")
	}

	time.Sleep(2 * time.Second)
}

func testRealTimeData(client mqtt.Client) {
	fmt.Println()
	fmt.Println("--- Test 3.1.5: Real-time Data Push ---")

	deviceID := "test-device-001"
	dataTopic := fmt.Sprintf("edgex/data/%s/%s", nodeID, deviceID)
	fmt.Printf("[SUBSCRIBE] Data topic: %s\n", dataTopic)

	token := client.Subscribe(dataTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		var message Message
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			fmt.Printf("[ERROR] Failed to parse data: %v\n", err)
			return
		}
		fmt.Printf("[RECV] Real-time data - type: %s\n", message.Header.MessageType)

		if body, ok := message.Body.(map[string]any); ok {
			if points, ok := body["points"].(map[string]any); ok {
				fmt.Printf("[INFO] Points received: %d\n", len(points))
				for k, v := range points {
					fmt.Printf("[INFO]   %s = %v\n", k, v)
				}
			}
		}
	})
	token.Wait()

	// Publish test data
	testData := Message{
		Header: MessageHeader{
			MessageID:   generateMsgID(),
			Timestamp:   time.Now().UnixMilli(),
			Source:      nodeID,
			MessageType: "data",
			Version:     "1.0",
		},
		Body: map[string]any{
			"node_id":   nodeID,
			"device_id": deviceID,
			"timestamp": time.Now().UnixMilli(),
			"points": map[string]any{
				"temperature": 25.5,
				"humidity":    60.0,
				"pressure":    1013.25,
			},
			"quality": "good",
		},
	}

	payload, _ := json.Marshal(testData)
	fmt.Printf("[PUBLISH] Publishing real-time data\n")
	token = client.Publish(dataTopic, 0, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("[ERROR] Publish failed: %v\n", token.Error())
	} else {
		fmt.Println("[OK] Real-time data published")
	}

	time.Sleep(2 * time.Second)
}

func testHeartbeat(client mqtt.Client) {
	fmt.Println()
	fmt.Println("--- Test 3.1.6: Heartbeat ---")

	heartbeatTopic := fmt.Sprintf("edgex/heartbeat/%s", nodeID)
	fmt.Printf("[SUBSCRIBE] Heartbeat topic: %s\n", heartbeatTopic)

	token := client.Subscribe(heartbeatTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		var message Message
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			fmt.Printf("[ERROR] Failed to parse heartbeat: %v\n", err)
			return
		}
		fmt.Printf("[RECV] Heartbeat - type: %s, status: %s\n",
			message.Header.MessageType, message.Header.Source)
	})
	token.Wait()

	// Publish test heartbeat
	heartbeat := Message{
		Header: MessageHeader{
			MessageID:   generateMsgID(),
			Timestamp:   time.Now().UnixMilli(),
			Source:      nodeID,
			MessageType: "heartbeat",
			Version:     "1.0",
		},
		Body: map[string]any{
			"node_id":   nodeID,
			"status":    "active",
			"timestamp": time.Now().UnixMilli(),
			"metrics": map[string]any{
				"cpu_usage":    45.5,
				"mem_usage":    62.3,
				"device_count": 10,
			},
		},
	}

	payload, _ := json.Marshal(heartbeat)
	fmt.Printf("[PUBLISH] Publishing heartbeat\n")
	token = client.Publish(heartbeatTopic, 0, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("[ERROR] Publish failed: %v\n", token.Error())
	} else {
		fmt.Println("[OK] Heartbeat published")
	}

	time.Sleep(2 * time.Second)
}

func testSubscribeCommand(client mqtt.Client) {
	fmt.Println()
	fmt.Println("--- Test 3.1.7: Subscribe to Write Command ---")

	// Simulate EdgeOS sending a write command
	writeTopic := fmt.Sprintf("edgex/cmd/%s/test-device-001/write", nodeID)
	responseTopic := fmt.Sprintf("edgex/responses/%s/cmd-test-001", nodeID)

	fmt.Printf("[SUBSCRIBE] Response topic: %s\n", responseTopic)
	token := client.Subscribe(responseTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		var message Message
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			fmt.Printf("[ERROR] Failed to parse response: %v\n", err)
			return
		}
		fmt.Printf("[RECV] Write command response - success: %v\n", message.Header.MessageType)
	})
	token.Wait()

	// Publish write command (simulating EdgeOS)
	writeCmd := Message{
		Header: MessageHeader{
			MessageID:     "cmd-test-001",
			Timestamp:     time.Now().UnixMilli(),
			Source:        "edgeos-server",
			Destination:   nodeID,
			MessageType:   "write",
			Version:       "1.0",
			CorrelationID: "cmd-test-001",
		},
		Body: map[string]any{
			"points": map[string]any{
				"temperature": 30.0,
			},
		},
	}

	payload, _ := json.Marshal(writeCmd)
	fmt.Printf("[PUBLISH] Simulating EdgeOS write command to %s\n", writeTopic)
	token = client.Publish(writeTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("[ERROR] Publish failed: %v\n", token.Error())
	} else {
		fmt.Println("[OK] Write command simulated")
	}

	time.Sleep(3 * time.Second)
}

func generateMsgID() string {
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}
