package core

import (
	"edge-gateway/internal/model"
	"edge-gateway/internal/storage"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// DeviceStorageManager handles minute-level data persistence for devices
type DeviceStorageManager struct {
	storage      *storage.Storage
	pipeline     *DataPipeline
	mu           sync.Mutex
	snapshots    map[string]map[string]any // deviceID -> pointID -> value
	deviceCfgs   map[string]model.DeviceStorage
	tickers      map[string]*time.Ticker
	stopChans    map[string]chan struct{}
	intervalUnit time.Duration // For testing
}

func NewDeviceStorageManager(s *storage.Storage, dp *DataPipeline) *DeviceStorageManager {
	dsm := &DeviceStorageManager{
		storage:      s,
		pipeline:     dp,
		snapshots:    make(map[string]map[string]any),
		deviceCfgs:   make(map[string]model.DeviceStorage),
		tickers:      make(map[string]*time.Ticker),
		stopChans:    make(map[string]chan struct{}),
		intervalUnit: time.Minute,
	}

	// Register to pipeline
	dp.AddHandler(dsm.handleValue)

	return dsm
}

func (m *DeviceStorageManager) handleValue(val model.Value) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update snapshot regardless of storage strategy
	if _, ok := m.snapshots[val.DeviceID]; !ok {
		m.snapshots[val.DeviceID] = make(map[string]any)
	}
	m.snapshots[val.DeviceID][val.PointID] = val.Value

	// Check if "Realtime" strategy
	cfg, ok := m.deviceCfgs[val.DeviceID]
	if !ok || !cfg.Enable {
		return
	}

	if cfg.Strategy == "realtime" {
		// Save immediately (debouncing might be needed in real world, but requirement says "Every 1 record" which might mean every update)
		// For now, let's treat "realtime" as "save on every update"
		// However, requirement says "All points merged into one record".
		// If we save on every point update, we might have partial data or many records.
		// "每1条" usually implies "Record every point update separately" OR "Snapshot on every update".
		// Given "All points merged", "Realtime" might mean "Snapshot on every update".
		// Let's implement Snapshot on every update for realtime.
		go m.saveSnapshot(val.DeviceID, time.Now())
	}
}

// UpdateDeviceConfig updates the storage configuration for a device
func (m *DeviceStorageManager) UpdateDeviceConfig(deviceID string, cfg model.DeviceStorage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing ticker
	if stop, ok := m.stopChans[deviceID]; ok {
		close(stop)
		delete(m.stopChans, deviceID)
	}
	if ticker, ok := m.tickers[deviceID]; ok {
		ticker.Stop()
		delete(m.tickers, deviceID)
	}

	m.deviceCfgs[deviceID] = cfg

	if !cfg.Enable {
		return
	}

	if cfg.Strategy == "interval" && cfg.Interval > 0 {
		ticker := time.NewTicker(time.Duration(cfg.Interval) * m.intervalUnit)
		stop := make(chan struct{})
		m.tickers[deviceID] = ticker
		m.stopChans[deviceID] = stop

		go func() {
			for {
				select {
				case t := <-ticker.C:
					m.saveSnapshot(deviceID, t)
				case <-stop:
					return
				}
			}
		}()
	}
}

func (m *DeviceStorageManager) saveSnapshot(deviceID string, ts time.Time) {
	m.mu.Lock()
	snapshot, ok := m.snapshots[deviceID]
	if !ok || len(snapshot) == 0 {
		m.mu.Unlock()
		return
	}

	// Deep copy snapshot to avoid race conditions during async save
	data := make(map[string]any)
	for k, v := range snapshot {
		data[k] = v
	}
	cfg := m.deviceCfgs[deviceID]
	m.mu.Unlock()

	// Construct record
	record := map[string]any{
		"ts":   ts.Unix(), // Timestamp for sorting
		"data": data,
	}

	// Bucket name: device_history_<deviceID>
	bucket := fmt.Sprintf("device_history_%s", deviceID)

	// Key: timestamp string (RFC3339Nano for uniqueness and sortability)
	key := ts.Format(time.RFC3339Nano)

	// Save to DB
	if m.storage == nil {
		// Storage not initialized (e.g. timeout), skip saving
		return
	}
	if err := m.storage.SaveData(bucket, key, record); err != nil {
		log.Printf("[Storage] Failed to save history for device %s: %v", deviceID, err)
		return
	}

	max := cfg.MaxRecords
	if max <= 0 {
		max = 1000 // Default
	}

	// Run prune in background to avoid blocking
	go m.pruneHistory(bucket, max)
}

func (m *DeviceStorageManager) pruneHistory(bucket string, maxRecords int) {
	// Simple prune: Count and delete excess
	// Optimization: This can be expensive if done every time.
	// Maybe do it probabilistically or check count first.
	// For now, let's just get count and delete from head if needed.

	// NOTE: storage.Storage doesn't expose easy Count/Cursor.
	// We might need to extend Storage or use LoadAll.
	// Extending Storage to support Cursor would be better for performance.
	// But given the constraints, let's try to add a method to Storage or use what we have.
	// We have LoadAll.

	// Let's rely on bbolt's ordered keys.
	// We need to delete oldest. Oldest keys are lexicographically smaller (RFC3339).

	if m.storage == nil {
		return
	}

	m.storage.PruneOldest(bucket, maxRecords)
}

// RemoveDevice stops storage for a device and cleans up resources
func (m *DeviceStorageManager) RemoveDevice(deviceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stop, ok := m.stopChans[deviceID]; ok {
		close(stop)
		delete(m.stopChans, deviceID)
	}
	if ticker, ok := m.tickers[deviceID]; ok {
		ticker.Stop()
		delete(m.tickers, deviceID)
	}
	delete(m.deviceCfgs, deviceID)
	delete(m.snapshots, deviceID)
}

// GetHistory retrieves history for a device
// limit: max records to return
func (m *DeviceStorageManager) GetHistory(deviceID string, limit int) ([]map[string]any, error) {
	bucket := fmt.Sprintf("device_history_%s", deviceID)
	var records []map[string]any

	// We want latest first? User said "1000条历史数据查询". Usually latest first.
	// But bbolt stores sorted by key (time).
	// We can fetch all (up to limit from end).

	if m.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	err := m.storage.LoadLatest(bucket, limit, func(k, v []byte) error {
		var rec map[string]any
		if err := json.Unmarshal(v, &rec); err != nil {
			return nil // Skip invalid
		}
		records = append(records, rec)
		return nil
	})

	return records, err
}

func (m *DeviceStorageManager) GetHistoryByTimeRange(deviceID string, start, end time.Time) ([]map[string]any, error) {
	bucket := fmt.Sprintf("device_history_%s", deviceID)
	var records []map[string]any

	minKey := start.Format(time.RFC3339Nano)
	maxKey := end.Format(time.RFC3339Nano)

	// log.Printf("[DeviceStorage] Querying %s from %s to %s", bucket, minKey, maxKey)
	startTime := time.Now()

	err := m.storage.LoadRange(bucket, minKey, maxKey, func(k, v []byte) error {
		var rec map[string]any
		if err := json.Unmarshal(v, &rec); err != nil {
			return nil // Skip invalid
		}
		records = append(records, rec)
		return nil
	})

	duration := time.Since(startTime)
	if duration > 1*time.Second {
		log.Printf("[DeviceStorage] Slow query for %s: %v, records: %d", bucket, duration, len(records))
	}

	return records, err
}
