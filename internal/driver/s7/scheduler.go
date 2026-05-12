package s7

import (
	"context"
	"fmt"
	"sync"
	"time"

	"edge-gateway/internal/model"

	"github.com/anviod/gos7"
	"go.uber.org/zap"
)

// S7Scheduler S7调度器，负责批量分组读写
type S7Scheduler struct {
	transport *S7Transport
	decoder   *S7Decoder

	// 配置
	batchReadMax int // 单次AGReadMulti最大读取点数（最大20）
	pduSize      int // PDU大小限制（字节）
	minInterval  time.Duration // 指令最小间隔

	// 统计
	totalRequests int64
	successCount  int64
	failureCount  int64
	mu            sync.Mutex
}

// NewS7Scheduler 创建S7调度器
func NewS7Scheduler(transport *S7Transport, decoder *S7Decoder, cfg map[string]any) *S7Scheduler {
	s := &S7Scheduler{
		transport:    transport,
		decoder:      decoder,
		batchReadMax: getCfgInt(cfg, "batch_read_max", 20),
		pduSize:      getCfgInt(cfg, "pdu_size", 4096),
		minInterval:  5 * time.Millisecond,
	}
	// AGReadMulti 最大支持20个项目
	if s.batchReadMax > 20 {
		s.batchReadMax = 20
	}
	return s
}

// PointGroup 点位分组，按Area+DBNumber分组
type PointGroup struct {
	Area     int
	DBNumber int
	Points   []pointWithArea
}

// pointWithArea 带解析地址的点位
type pointWithArea struct {
	Point model.Point
	Area  *S7Area
}

// ReadPoints 批量读取点位
func (s *S7Scheduler) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	results := make(map[string]model.Value)

	// 1. 解析所有点位地址
	var parsed []pointWithArea
	for _, p := range points {
		area, err := s.decoder.ParseAddress(p.Address)
		if err != nil {
			zap.L().Warn("[S7] Failed to parse address",
				zap.String("point", p.Name),
				zap.String("address", p.Address),
				zap.Error(err),
			)
			// 记录失败但继续处理其他点位
			results[p.ID] = model.Value{
				PointID: p.ID,
				Quality: "Bad",
				TS:      time.Now(),
			}
			s.incFailure()
			continue
		}
		parsed = append(parsed, pointWithArea{Point: p, Area: area})
	}

	// 2. 按Area+DBNumber分组
	groups := s.groupPoints(parsed)

	// 3. 逐组批量读取
	for _, group := range groups {
		if err := s.readGroup(ctx, group, results); err != nil {
			zap.L().Warn("[S7] Failed to read group",
				zap.Int("area", group.Area),
				zap.Int("dbNumber", group.DBNumber),
				zap.Error(err),
			)
		}
	}

	return results, nil
}

// groupPoints 按Area和DBNumber对点位分组
func (s *S7Scheduler) groupPoints(points []pointWithArea) []PointGroup {
	groupMap := make(map[string]*PointGroup)

	for _, p := range points {
		key := fmt.Sprintf("%d:%d", p.Area.Area, p.Area.DBNumber)
		if group, ok := groupMap[key]; ok {
			group.Points = append(group.Points, p)
		} else {
			groupMap[key] = &PointGroup{
				Area:     p.Area.Area,
				DBNumber: p.Area.DBNumber,
				Points:   []pointWithArea{p},
			}
		}
	}

	groups := make([]PointGroup, 0, len(groupMap))
	for _, g := range groupMap {
		// 拆分超大组（AGReadMulti最大支持20个项目）
		for len(g.Points) > s.batchReadMax {
			groups = append(groups, PointGroup{
				Area:     g.Area,
				DBNumber: g.DBNumber,
				Points:   g.Points[:s.batchReadMax],
			})
			g.Points = g.Points[s.batchReadMax:]
		}
		if len(g.Points) > 0 {
			groups = append(groups, *g)
		}
	}

	return groups
}

// readGroup 使用AGReadMulti批量读取一组点位
func (s *S7Scheduler) readGroup(ctx context.Context, group PointGroup, results map[string]model.Value) error {
	client := s.transport.GetClient()
	if client == nil {
		return fmt.Errorf("S7 client not connected")
	}

	if len(group.Points) == 0 {
		return nil
	}

	// 将点位转换为S7DataItem
	dataItems := make([]gos7.S7DataItem, 0, len(group.Points))
	pointIndexMap := make(map[int]int) // dataItems index -> group.Points index

	for i, pwa := range group.Points {
		s.incTotal()

		area := pwa.Area
		readSize := s.decoder.ReadSizeForArea(area)

		item := gos7.S7DataItem{
			Area:     area.Area,
			WordLen:  area.WordLen,
			DBNumber: area.DBNumber,
			Start:    area.ByteOff,
			Bit:      area.BitOff,
			Amount:   1, // 单个元素
			Data:     make([]byte, readSize),
		}

		// 字节类型使用readSize作为Amount
		if area.WordLen == S7WLByte {
			item.Amount = readSize
		}

		pointIndexMap[len(dataItems)] = i
		dataItems = append(dataItems, item)
	}

	// 调用AGReadMulti批量读取
	if err := client.AGReadMulti(dataItems, len(dataItems)); err != nil {
		// 整个批量读取失败，所有点位标记为Bad
		zap.L().Warn("[S7] AGReadMulti failed",
			zap.Int("area", group.Area),
			zap.Int("dbNumber", group.DBNumber),
			zap.Int("items", len(dataItems)),
			zap.Error(err),
		)
		for _, pwa := range group.Points {
			results[pwa.Point.ID] = model.Value{
				PointID: pwa.Point.ID,
				Quality: "Bad",
				TS:      time.Now(),
			}
			s.incFailure()
		}
		return err
	}

	// 解码每个数据项的结果
	for idx, item := range dataItems {
		pwa := group.Points[pointIndexMap[idx]]

		if item.Error != "" {
			zap.L().Debug("[S7] AGReadMulti item error",
				zap.String("point", pwa.Point.Name),
				zap.String("address", pwa.Point.Address),
				zap.String("error", item.Error),
			)
			results[pwa.Point.ID] = model.Value{
				PointID: pwa.Point.ID,
				Quality: "Bad",
				TS:      time.Now(),
			}
			s.incFailure()
			continue
		}

		// 解码值
		val, err := s.decoder.DecodeValue(item.Data, pwa.Area, pwa.Point.DataType)
		if err != nil {
			zap.L().Debug("[S7] Decode value failed",
				zap.String("point", pwa.Point.Name),
				zap.String("address", pwa.Point.Address),
				zap.Error(err),
			)
			results[pwa.Point.ID] = model.Value{
				PointID: pwa.Point.ID,
				Quality: "Bad",
				TS:      time.Now(),
			}
			s.incFailure()
			continue
		}

		results[pwa.Point.ID] = model.Value{
			PointID: pwa.Point.ID,
			Value:   val,
			Quality: "Good",
			TS:      time.Now(),
		}
		s.incSuccess()
	}

	// 指令间隔
	if s.minInterval > 0 {
		time.Sleep(s.minInterval)
	}

	return nil
}

// readSinglePoint 读取单个点位（用于写入前的读取或单点读取回退）
func (s *S7Scheduler) readSinglePoint(client gos7.Client, pwa pointWithArea) (interface{}, error) {
	area := pwa.Area
	readSize := s.decoder.ReadSizeForArea(area)
	buffer := make([]byte, readSize)

	var err error
	switch area.Area {
	case S7AreaDB:
		err = client.AGReadDB(area.DBNumber, area.ByteOff, readSize, buffer)
	case S7AreaMK:
		err = client.AGReadMB(area.ByteOff, readSize, buffer)
	case S7AreaPE:
		err = client.AGReadEB(area.ByteOff, readSize, buffer)
	case S7AreaPA:
		err = client.AGReadAB(area.ByteOff, readSize, buffer)
	case S7AreaTM:
		err = client.AGReadTM(area.ByteOff, readSize, buffer)
	case S7AreaCT:
		err = client.AGReadCT(area.ByteOff, readSize, buffer)
	default:
		return nil, fmt.Errorf("unsupported S7 area: %d", area.Area)
	}

	if err != nil {
		return nil, err
	}

	// 解码值
	return s.decoder.DecodeValue(buffer, area, pwa.Point.DataType)
}

// WritePoint 写入单个点位
func (s *S7Scheduler) WritePoint(ctx context.Context, p model.Point, value interface{}) error {
	client := s.transport.GetClient()
	if client == nil {
		return fmt.Errorf("S7 client not connected")
	}

	area, err := s.decoder.ParseAddress(p.Address)
	if err != nil {
		return fmt.Errorf("invalid S7 address %s: %w", p.Address, err)
	}

	writeSize := s.decoder.ReadSizeForArea(area)
	buffer := make([]byte, writeSize)

	// 编码值
	if err := s.decoder.EncodeValue(buffer, area, p.DataType, value); err != nil {
		return fmt.Errorf("encode value failed: %w", err)
	}

	s.incTotal()

	switch area.Area {
	case S7AreaDB:
		err = client.AGWriteDB(area.DBNumber, area.ByteOff, writeSize, buffer)
	case S7AreaMK:
		err = client.AGWriteMB(area.ByteOff, writeSize, buffer)
	case S7AreaPE:
		err = client.AGWriteEB(area.ByteOff, writeSize, buffer)
	case S7AreaPA:
		err = client.AGWriteAB(area.ByteOff, writeSize, buffer)
	case S7AreaTM:
		err = client.AGWriteTM(area.ByteOff, writeSize, buffer)
	case S7AreaCT:
		err = client.AGWriteCT(area.ByteOff, writeSize, buffer)
	default:
		return fmt.Errorf("unsupported S7 area: %d", area.Area)
	}

	if err != nil {
		s.incFailure()
		return err
	}

	s.incSuccess()
	return nil
}

// GetStats 获取统计信息
func (s *S7Scheduler) GetStats() (total, success, failure int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.totalRequests, s.successCount, s.failureCount
}

func (s *S7Scheduler) incTotal() {
	s.mu.Lock()
	s.totalRequests++
	s.mu.Unlock()
}

func (s *S7Scheduler) incSuccess() {
	s.mu.Lock()
	s.successCount++
	s.mu.Unlock()
}

func (s *S7Scheduler) incFailure() {
	s.mu.Lock()
	s.failureCount++
	s.mu.Unlock()
}
