package opcua

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"edge-gateway/internal/driver"
	"edge-gateway/internal/model"
	"edge-gateway/internal/pkg/dataformat"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
)

func init() {
	driver.RegisterDriver("opc-ua", NewOpcUaDriver)
}

type OpcUaDriver struct {
	mu                   sync.Mutex
	clients              map[string]*ClientWrapper // Key: Endpoint URL
	activeClient         *ClientWrapper
	useDataformatDecoder bool
}

type DeviceSubscription struct {
	mu         sync.RWMutex
	Sub        *opcua.Subscription
	Cache      map[string]model.Value
	PointIDs   []string
	Points     map[string]model.Point
	HandleMap  map[uint32]string
	NextHandle uint32
	NotifyCh   chan *opcua.PublishNotificationData
	Ctx        context.Context
	Cancel     context.CancelFunc
	lastUpdate time.Time
}

type ClientWrapper struct {
	Client        *opcua.Client
	Endpoint      string
	Connected     bool
	Config        map[string]any
	mu            sync.Mutex
	Subscriptions map[string]*DeviceSubscription // DeviceID -> Subscription
}

func NewOpcUaDriver() driver.Driver {
	return &OpcUaDriver{
		clients: make(map[string]*ClientWrapper),
	}
}

func (d *OpcUaDriver) Init(cfg model.DriverConfig) error {
	return nil
}

func (d *OpcUaDriver) Connect(ctx context.Context) error {
	return nil
}

func (d *OpcUaDriver) Disconnect() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.clients {
		if c.Client != nil {
			c.Client.Close(context.Background())
		}
	}
	d.clients = make(map[string]*ClientWrapper)
	return nil
}

func (d *OpcUaDriver) Health() driver.HealthStatus {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.activeClient != nil && d.activeClient.Connected {
		return driver.HealthStatusGood
	}
	return driver.HealthStatusUnknown
}

func (d *OpcUaDriver) SetSlaveID(slaveID uint8) error {
	return nil
}

func (d *OpcUaDriver) SetDeviceConfig(config map[string]any) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if v, ok := config["use_dataformat_decoder"]; ok {
		switch val := v.(type) {
		case bool:
			d.useDataformatDecoder = val
		case string:
			if val == "true" || val == "1" {
				d.useDataformatDecoder = true
			} else {
				d.useDataformatDecoder = false
			}
		case float64:
			d.useDataformatDecoder = val != 0
		}
	}

	endpoint, ok := config["endpoint"].(string)
	if !ok || endpoint == "" {
		return fmt.Errorf("endpoint is required in device config")
	}

	// Check if client exists
	if wrapper, exists := d.clients[endpoint]; exists {
		d.activeClient = wrapper
		// Check connection state
		if wrapper.Client.State() == opcua.Closed {
			wrapper.Connected = false
			// Try reconnect
			go d.reconnect(wrapper)
		}
		return nil
	}

	// Create new client
	wrapper := &ClientWrapper{
		Endpoint:      endpoint,
		Config:        config,
		Subscriptions: make(map[string]*DeviceSubscription),
	}

	opts, err := d.buildClientOptions(config)
	if err != nil {
		return err
	}

	c, err := opcua.NewClient(endpoint, opts...)
	if err != nil {
		return fmt.Errorf("failed to create opcua client: %v", err)
	}
	wrapper.Client = c

	// Initial connect
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		zap.L().Warn("[OPC UA] Failed to connect", zap.String("endpoint", endpoint), zap.Error(err))
		// We still register the client, but it's disconnected
		wrapper.Connected = false
	} else {
		wrapper.Connected = true
		zap.L().Info("[OPC UA] Connected", zap.String("endpoint", endpoint))
	}

	d.clients[endpoint] = wrapper
	d.activeClient = wrapper
	return nil
}

func (d *OpcUaDriver) GetConnectionMetrics() (connectionSeconds int64, reconnectCount int64, localAddr string, remoteAddr string, lastDisconnectTime time.Time) {
	return 0, 0, "", "", time.Time{}
}

func (d *OpcUaDriver) reconnect(w *ClientWrapper) {
	d.mu.Lock()
	if w.Connected {
		d.mu.Unlock()
		return
	}
	d.mu.Unlock()

	zap.L().Info("[OPC UA] Reconnecting", zap.String("endpoint", w.Endpoint))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := w.Client.Connect(ctx); err == nil {
		d.mu.Lock()
		w.Connected = true
		d.mu.Unlock()
		zap.L().Info("[OPC UA] Reconnected", zap.String("endpoint", w.Endpoint))
	} else {
		zap.L().Warn("[OPC UA] Reconnection failed", zap.Error(err))
	}
}

func (d *OpcUaDriver) buildClientOptions(config map[string]any) ([]opcua.Option, error) {
	opts := []opcua.Option{
		opcua.RequestTimeout(10 * time.Second),
		// opcua.SessionTimeout(30 * time.Minute),
	}

	/*
		// Security Policy
		policy, _ := config["security_policy"].(string)
		if policy == "" {
			policy = "None"
		}
		opts = append(opts, opcua.SecurityPolicy(policy))

		// Security Mode
		modeStr, _ := config["security_mode"].(string)
		mode := ua.MessageSecurityModeNone
		switch modeStr {
		case "Sign":
			mode = ua.MessageSecurityModeSign
		case "SignAndEncrypt":
			mode = ua.MessageSecurityModeSignAndEncrypt
		}
		opts = append(opts, opcua.SecurityMode(mode))

		// Auth Method
		authMethod, _ := config["auth_method"].(string)
		switch authMethod {
		case "UserName":
			user, _ := config["username"].(string)
			pass, _ := config["password"].(string)
			opts = append(opts, opcua.AuthUsername(user, pass))
		case "Certificate":
			certFile, _ := config["certificate_file"].(string)
			keyFile, _ := config["private_key_file"].(string)
			certBytes, err := os.ReadFile(certFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read cert file: %v", err)
			}
			opts = append(opts, opcua.AuthCertificate(certBytes))
			opts = append(opts, opcua.PrivateKeyFile(keyFile))
		default:
			opts = append(opts, opcua.AuthAnonymous())
		}
	*/

	return opts, nil
}

func (d *OpcUaDriver) ReadPoints(ctx context.Context, points []model.Point) (map[string]model.Value, error) {
	d.mu.Lock()
	client := d.activeClient
	d.mu.Unlock()

	if client == nil {
		return nil, fmt.Errorf("no active client")
	}
	if !client.Connected {
		// Try to reconnect synchronously
		if err := client.Client.Connect(ctx); err != nil {
			return nil, fmt.Errorf("client not connected: %v", err)
		}
		d.mu.Lock()
		client.Connected = true
		d.mu.Unlock()
	}

	if len(points) == 0 {
		return nil, nil
	}

	deviceID := points[0].DeviceID
	// Identify if we should use subscription or direct read.
	// For now, default to subscription as requested.

	// Get Subscription
	sub := d.ensureSubscription(ctx, client, deviceID, points)

	// If subscription failed, fallback to direct read?
	// For now, let's try to return from cache.
	if sub != nil {
		sub.mu.RLock()
		defer sub.mu.RUnlock()

		result := make(map[string]model.Value)
		// Check if we have values
		missing := false
		for _, p := range points {
			if v, ok := sub.Cache[p.ID]; ok && v.Value != nil {
				result[p.ID] = v
				zap.L().Debug("[OPC UA] Read (Cache)", zap.String("point", p.ID), zap.Any("value", v.Value), zap.String("quality", v.Quality))
			} else {
				missing = true
				// Return Bad quality if missing
				result[p.ID] = model.Value{
					PointID: p.ID,
					Quality: "Bad",
					Value:   0,
					TS:      time.Now(),
				}
				zap.L().Warn("[OPC UA] Cache Miss or Nil", zap.String("point", p.ID))
			}
		}

		if !missing {
			return result, nil
		}

		// If missing, log it and fallback to direct read for ALL points to ensure consistency
		zap.L().Warn("[OPC UA] Cache missing or incomplete", zap.Int("count", len(points)))
	} else {
		zap.L().Debug("[OPC UA] No subscription, using direct read")
	}

	// Fallback to direct read (also used for initial value population)
	return d.readDirect(ctx, client, points)
}

func (d *OpcUaDriver) ensureSubscription(ctx context.Context, w *ClientWrapper, deviceID string, points []model.Point) *DeviceSubscription {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if subscription exists
	sub, exists := w.Subscriptions[deviceID]

	// Check if points changed
	currentIDs := make([]string, len(points))
	for i, p := range points {
		currentIDs[i] = p.ID
	}
	sort.Strings(currentIDs)

	if exists {
		// Compare IDs
		if len(sub.PointIDs) == len(currentIDs) {
			match := true
			for i := range currentIDs {
				if sub.PointIDs[i] != currentIDs[i] {
					match = false
					break
				}
			}
			if match {
				return sub
			}
		}
		// Changed: cancel old
		sub.Cancel()
	}

	// Create new subscription
	notifyCh := make(chan *opcua.PublishNotificationData)
	subCtx, cancel := context.WithCancel(context.Background())

	opcuaSub, err := w.Client.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval: 1000 * time.Millisecond, // 1s interval
	}, notifyCh)

	if err != nil {
		zap.L().Error("[OPC UA] Failed to create subscription", zap.String("device_id", deviceID), zap.Error(err))
		cancel()
		return nil
	}

	newSub := &DeviceSubscription{
		Sub:        opcuaSub,
		Cache:      make(map[string]model.Value),
		PointIDs:   currentIDs,
		Points:     make(map[string]model.Point),
		HandleMap:  make(map[uint32]string),
		NextHandle: 1,
		NotifyCh:   notifyCh,
		Ctx:        subCtx,
		Cancel:     cancel,
	}

	for _, p := range points {
		newSub.Points[p.ID] = p
	}

	// Create monitored items
	requests := make([]*ua.MonitoredItemCreateRequest, len(points))
	for i, p := range points {
		id, err := ua.ParseNodeID(p.Address)
		if err != nil {
			zap.L().Error("[OPC UA] Invalid node id", zap.String("address", p.Address), zap.Error(err))
			continue
		}

		handle := newSub.NextHandle
		newSub.NextHandle++
		newSub.HandleMap[handle] = p.ID

		requests[i] = opcua.NewMonitoredItemCreateRequestWithDefaults(
			id,
			ua.AttributeIDValue,
			handle,
		)
	}

	if len(requests) > 0 {
		resp, err := opcuaSub.Monitor(ctx, ua.TimestampsToReturnBoth, requests...)
		if err != nil {
			zap.L().Error("[OPC UA] Monitor failed", zap.Error(err))
		} else {
			// Check results
			for i, res := range resp.Results {
				if res.StatusCode != ua.StatusOK {
					zap.L().Error("[OPC UA] Monitor item failed", zap.String("address", points[i].Address), zap.Any("status", res.StatusCode))
				}
			}
		}
	}

	// Start processing loop
	go d.subscriptionLoop(newSub)

	w.Subscriptions[deviceID] = newSub
	return newSub
}

func (d *OpcUaDriver) subscriptionLoop(sub *DeviceSubscription) {
	for {
		select {
		case <-sub.Ctx.Done():
			return
		case res, ok := <-sub.NotifyCh:
			if !ok {
				return
			}
			if res.Error != nil {
				zap.L().Error("[OPC UA] Subscription error", zap.Error(res.Error))
				continue
			}

			switch x := res.Value.(type) {
			case *ua.DataChangeNotification:
				sub.mu.Lock()
				for _, item := range x.MonitoredItems {
					pointID, ok := sub.HandleMap[item.ClientHandle]
					if !ok {
						continue
					}

					val := model.Value{
						PointID: pointID,
						TS:      time.Now(),
					}

					if item.Value != nil {
						if item.Value.Status == ua.StatusOK {
							val.Quality = "Good"
							raw := item.Value.Value.Value()
							if d.useDataformatDecoder {
								if p, ok := sub.Points[pointID]; ok {
									if formatted, err := dataformat.FormatScalar(p, "ABCD", raw); err == nil {
										raw = formatted
									}
								}
							}
							val.Value = raw
							if !item.Value.SourceTimestamp.IsZero() {
								val.TS = item.Value.SourceTimestamp
							}
						} else {
							val.Quality = "Bad"
							zap.L().Warn("[OPC UA] Subscription update bad status", zap.String("point_id", pointID), zap.Any("status", item.Value.Status))
						}
					}

					sub.Cache[pointID] = val
				}
				sub.mu.Unlock()
			}
		}
	}
}

func (d *OpcUaDriver) readDirect(ctx context.Context, client *ClientWrapper, points []model.Point) (map[string]model.Value, error) {
	req := &ua.ReadRequest{
		MaxAge:             2000,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		NodesToRead:        make([]*ua.ReadValueID, len(points)),
	}

	for i, p := range points {
		id, err := ua.ParseNodeID(p.Address)
		if err != nil {
			return nil, fmt.Errorf("invalid node id %s: %v", p.Address, err)
		}
		req.NodesToRead[i] = &ua.ReadValueID{
			NodeID:      id,
			AttributeID: ua.AttributeIDValue,
		}
	}

	resp, err := client.Client.Read(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Results == nil || len(resp.Results) != len(points) {
		return nil, fmt.Errorf("invalid read response")
	}

	result := make(map[string]model.Value)
	now := time.Now()

	for i, res := range resp.Results {
		p := points[i]
		val := model.Value{
			PointID: p.ID,
			TS:      now,
		}

		if res.Status != ua.StatusOK {
			val.Quality = "Bad"
			zap.L().Warn("[OPC UA] Read failed", zap.String("point_id", p.ID), zap.Any("status", res.Status))
		} else {
			val.Quality = "Good"
			raw := res.Value.Value()
			if d.useDataformatDecoder {
				if formatted, err := dataformat.FormatScalar(p, "ABCD", raw); err == nil {
					raw = formatted
				}
			}
			val.Value = raw
			// Use SourceTimestamp if available
			if !res.SourceTimestamp.IsZero() {
				val.TS = res.SourceTimestamp
			}
		}
		result[p.ID] = val
	}

	return result, nil
}

func (d *OpcUaDriver) WritePoint(ctx context.Context, point model.Point, value any) error {
	d.mu.Lock()
	client := d.activeClient
	d.mu.Unlock()

	if client == nil || !client.Connected {
		return fmt.Errorf("client not connected")
	}

	id, err := ua.ParseNodeID(point.Address)
	if err != nil {
		return fmt.Errorf("invalid node id: %v", err)
	}

	valToWrite, err := castValue(value, point.DataType)
	if err != nil {
		return fmt.Errorf("value conversion failed: %v", err)
	}

	v, err := ua.NewVariant(valToWrite)
	if err != nil {
		return fmt.Errorf("invalid value: %v", err)
	}

	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      id,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					Value: v,
				},
			},
		},
	}

	resp, err := client.Client.Write(ctx, req)
	if err != nil {
		return err
	}
	if len(resp.Results) > 0 && resp.Results[0] != ua.StatusOK {
		return fmt.Errorf("write failed: %s (0x%X)", resp.Results[0], uint32(resp.Results[0]))
	}
	zap.L().Info("[OPC UA] Write success", zap.String("point_id", point.ID), zap.Any("value", valToWrite))

	return nil
}

func (d *OpcUaDriver) ScanObjects(ctx context.Context, config map[string]any) (any, error) {
	endpoint, ok := config["endpoint"].(string)
	if !ok || endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	opts, err := d.buildClientOptions(config)
	if err != nil {
		return nil, err
	}

	// Start browsing from Objects folder
	rootID := ua.NewNumericNodeID(0, 85)

	if rootNodeIDStr, ok := config["root_node_id"].(string); ok && rootNodeIDStr != "" {
		id, err := ua.ParseNodeID(rootNodeIDStr)
		if err == nil {
			rootID = id
			zap.L().Info("Starting scan from custom node", zap.String("node_id", rootID.String()))
		}
	}

	// Create client
	c, err := opcua.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create scan client: %v", err)
	}

	scanCtx := ctx
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok {
		scanCtx, cancel = context.WithTimeout(ctx, 60*time.Second)
	}
	defer cancel()

	if err := c.Connect(scanCtx); err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	defer c.Close(context.Background())

	// Use recursive helper with concurrency control
	results, err := d.browseNode(scanCtx, c, rootID, 0)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %v", err)
	}

	return results, nil
}

// browseNode recursively browses the OPC UA address space
// It now supports concurrent browsing for children to speed up scanning
func (d *OpcUaDriver) browseNode(ctx context.Context, c *opcua.Client, nodeID *ua.NodeID, depth int) ([]map[string]any, error) {
	if depth > 10 { // Limit depth
		return nil, nil
	}

	// Fetch references
	refs, err := d.fetchReferences(ctx, c, nodeID)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	var variableNodeIDs []*ua.ReadValueID
	var variableIndices []int
	var childrenToBrowse []*ua.NodeID
	var childrenIndices []int

	for _, ref := range refs {
		// Convert NodeID to string
		nodeIDStr := ref.NodeID.NodeID.String()
		parsedID, parseErr := ua.ParseNodeID(nodeIDStr)
		if parseErr != nil {
			continue
		}

		// Skip standard "Server" object (ns=0;i=2253)
		if parsedID.Namespace() == 0 && (parsedID.IntID() == 2253 || parsedID.IntID() == 23470 || parsedID.IntID() == 31915) {
			continue
		}

		item := map[string]any{
			"node_id": nodeIDStr,
			"name":    ref.DisplayName.Text,
			"class":   ref.NodeClass.String(),
		}

		if ref.NodeClass == ua.NodeClassVariable {
			item["type"] = "Variable"
			item["address"] = nodeIDStr

			// Queue for DataType reading
			variableNodeIDs = append(variableNodeIDs, &ua.ReadValueID{
				NodeID:      parsedID,
				AttributeID: ua.AttributeIDDataType,
			})
			variableIndices = append(variableIndices, len(results))
			results = append(results, item)

		} else if ref.NodeClass == ua.NodeClassObject {
			item["type"] = "Folder"
			// Queue for recursive browsing
			childrenToBrowse = append(childrenToBrowse, parsedID)
			childrenIndices = append(childrenIndices, len(results))
			results = append(results, item)
		}
	}

	// 1. Batch Read DataTypes (Sequential, fast enough)
	if len(variableNodeIDs) > 0 {
		d.batchReadDataTypes(ctx, c, variableNodeIDs, results, variableIndices)
	}

	// 2. Browse Children (sequential)
	if len(childrenToBrowse) > 0 {
		for i, childID := range childrenToBrowse {
			idx := childrenIndices[i]
			children, err := d.browseNode(ctx, c, childID, depth+1)
			if err != nil {
				zap.L().Warn("Browse child failed", zap.String("node", childID.String()), zap.Error(err))
				results[idx]["browse_error"] = err.Error()
				continue
			}
			if len(children) > 0 {
				results[idx]["children"] = children
			}
		}
	}

	return results, nil
}

// fetchReferences handles the Browse request with retries
func (d *OpcUaDriver) fetchReferences(ctx context.Context, c *opcua.Client, nodeID *ua.NodeID) ([]ua.ReferenceDescription, error) {
	// Initial request
	req := &ua.BrowseRequest{
		RequestedMaxReferencesPerNode: 50,
		NodesToBrowse: []*ua.BrowseDescription{
			{
				NodeID:          nodeID,
				BrowseDirection: ua.BrowseDirectionForward,
				ReferenceTypeID: ua.NewNumericNodeID(0, 33), // HierarchicalReferences
				IncludeSubtypes: true,
				NodeClassMask:   uint32(ua.NodeClassObject | ua.NodeClassVariable),
				ResultMask:      uint32(ua.BrowseResultMaskAll),
			},
		},
	}

	resp, err := c.Browse(ctx, req)
	if err != nil && isOPCUAConnError(err) {
		_ = c.Close(context.Background())
		if err2 := c.Connect(ctx); err2 == nil {
			resp, err = c.Browse(ctx, req)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(resp.Results) == 0 || resp.Results[0].StatusCode != ua.StatusOK {
		return nil, fmt.Errorf("bad status code: %v", resp.Results[0].StatusCode)
	}

	var refs []ua.ReferenceDescription
	for _, r := range resp.Results[0].References {
		refs = append(refs, *r)
	}

	// Handle ContinuationPoint
	continuationPoint := resp.Results[0].ContinuationPoint
	for len(continuationPoint) > 0 {
		reqNext := &ua.BrowseNextRequest{
			ReleaseContinuationPoints: false,
			ContinuationPoints:        [][]byte{continuationPoint},
		}

		respNext, err := c.BrowseNext(ctx, reqNext)
		if err != nil {
			return nil, err
		}
		if len(respNext.Results) == 0 || respNext.Results[0].StatusCode != ua.StatusOK {
			return nil, fmt.Errorf("browse next bad status code: %v", respNext.Results[0].StatusCode)
		}

		for _, r := range respNext.Results[0].References {
			refs = append(refs, *r)
		}

		continuationPoint = respNext.Results[0].ContinuationPoint
	}

	return refs, nil
}

func isOPCUAConnError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection") || strings.Contains(msg, "EOF")
}

// batchReadDataTypes reads data types in batches
func (d *OpcUaDriver) batchReadDataTypes(ctx context.Context, c *opcua.Client, nodeIDs []*ua.ReadValueID, results []map[string]any, indices []int) {
	// Split into smaller chunks if necessary (e.g., 100 items)
	chunkSize := 50
	for i := 0; i < len(nodeIDs); i += chunkSize {
		end := i + chunkSize
		if end > len(nodeIDs) {
			end = len(nodeIDs)
		}

		chunkIDs := nodeIDs[i:end]
		chunkIndices := indices[i:end]

		req := &ua.ReadRequest{
			NodesToRead: chunkIDs,
			MaxAge:      2000,
		}

		resp, err := c.Read(ctx, req)
		if err != nil {
			zap.L().Warn("Read DataTypes chunk failed", zap.Error(err))
			continue
		}

		for j, res := range resp.Results {
			if res.Status == ua.StatusOK && res.Value != nil {
				if typeID, ok := res.Value.Value().(*ua.NodeID); ok {
					results[chunkIndices[j]]["data_type"] = lookupDataType(typeID)
				}
			}
		}
	}
}

func lookupDataType(id *ua.NodeID) string {
	if id.Namespace() != 0 {
		return id.String()
	}
	switch id.IntID() {
	case 1:
		return "Boolean"
	case 2:
		return "SByte"
	case 3:
		return "Byte"
	case 4:
		return "Int16"
	case 5:
		return "UInt16"
	case 6:
		return "Int32"
	case 7:
		return "UInt32"
	case 8:
		return "Int64"
	case 9:
		return "UInt64"
	case 10:
		return "Float"
	case 11:
		return "Double"
	case 12:
		return "String"
	case 13:
		return "DateTime"
	default:
		return fmt.Sprintf("ns=%d;i=%d", id.Namespace(), id.IntID())
	}
}

func castValue(val any, dataType string) (any, error) {
	dt := strings.ToLower(dataType)
	asString := func(v any) string {
		return fmt.Sprintf("%v", v)
	}

	switch {
	case dt == "bool" || dt == "boolean":
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			return strconv.ParseBool(v)
		default:
			s := asString(v)
			if b, err := strconv.ParseBool(s); err == nil {
				return b, nil
			}
			// Numeric fallback: != 0 is true
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f != 0, nil
			}
			return nil, fmt.Errorf("cannot cast %v to bool", val)
		}

	case strings.Contains(dt, "uint16") || dt == "unsignedshort":
		switch v := val.(type) {
		case uint16:
			return v, nil
		case float64:
			return uint16(v), nil
		case int:
			return uint16(v), nil
		}
		s := asString(val)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return uint16(f), nil
		}
		return nil, fmt.Errorf("cannot cast %v to uint16", val)

	case strings.Contains(dt, "int16") || dt == "short":
		switch v := val.(type) {
		case int16:
			return v, nil
		case float64:
			return int16(v), nil
		case int:
			return int16(v), nil
		}
		s := asString(val)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int16(f), nil
		}
		return nil, fmt.Errorf("cannot cast %v to int16", val)

	case dt == "sbyte" || dt == "int8":
		switch v := val.(type) {
		case int8:
			return v, nil
		case float64:
			return int8(v), nil
		case int:
			return int8(v), nil
		}
		s := asString(val)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int8(f), nil
		}
		return nil, fmt.Errorf("cannot cast %v to sbyte", val)

	case dt == "byte" || dt == "uint8":
		switch v := val.(type) {
		case uint8:
			return v, nil
		case float64:
			return uint8(v), nil
		case int:
			return uint8(v), nil
		}
		s := asString(val)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return uint8(f), nil
		}
		return nil, fmt.Errorf("cannot cast %v to byte", val)

	case strings.Contains(dt, "uint32") || dt == "unsignedint":
		switch v := val.(type) {
		case uint32:
			return v, nil
		case float64:
			return uint32(v), nil
		case int:
			return uint32(v), nil
		}
		s := asString(val)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return uint32(f), nil
		}
		return nil, fmt.Errorf("cannot cast %v to uint32", val)

	case strings.Contains(dt, "int32") || dt == "int":
		switch v := val.(type) {
		case int32:
			return v, nil
		case float64:
			return int32(v), nil
		case int:
			return int32(v), nil
		}
		s := asString(val)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int32(f), nil
		}
		return nil, fmt.Errorf("cannot cast %v to int32", val)

	case strings.Contains(dt, "uint64") || dt == "unsignedlong":
		switch v := val.(type) {
		case uint64:
			return v, nil
		case float64:
			return uint64(v), nil
		case int:
			return uint64(v), nil
		}
		s := asString(val)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return uint64(f), nil
		}
		return nil, fmt.Errorf("cannot cast %v to uint64", val)

	case strings.Contains(dt, "int64") || dt == "long":
		switch v := val.(type) {
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		case int:
			return int64(v), nil
		}
		s := asString(val)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f), nil
		}
		return nil, fmt.Errorf("cannot cast %v to int64", val)

	case strings.Contains(dt, "float32") || dt == "float":
		s := asString(val)
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return nil, err
		}
		return float32(f), nil

	case strings.Contains(dt, "float64") || dt == "double":
		s := asString(val)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		return f, nil

	case dt == "string":
		return asString(val), nil
	}

	return val, nil
}
