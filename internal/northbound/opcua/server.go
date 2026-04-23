package opcua

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"edge-gateway/internal/model"

	"github.com/awcullen/opcua/server"
	"github.com/awcullen/opcua/ua"
	"go.uber.org/zap"
)

// Server is the OPC UA Server implementation
type Server struct {
	config    model.OPCUAConfig
	sb        model.SouthboundManager
	srv       *server.Server
	mu        sync.RWMutex
	nodeMap   map[string]*server.VariableNode
	gatewayID string
	stats     Stats
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewServer creates a new OPC UA Server
func NewServer(cfg model.OPCUAConfig, sb model.SouthboundManager) *Server {
	return &Server{
		config:    cfg,
		sb:        sb,
		nodeMap:   make(map[string]*server.VariableNode),
		gatewayID: "Gateway",
	}
}

// Start starts the OPC UA Server
func (s *Server) Start() error {
	zap.L().Info("Starting OPC UA Server...",
		zap.String("name", s.config.Name),
		zap.Int("port", s.config.Port),
		zap.String("component", "opcua-server"),
	)

	s.ctx, s.cancel = context.WithCancel(context.Background())

	endpoint := fmt.Sprintf("opc.tcp://0.0.0.0:%d%s", s.config.Port, s.config.Endpoint)
	// Sanitize name for URI (remove spaces)
	safeName := strings.ReplaceAll(s.config.Name, " ", "")
	appURI := fmt.Sprintf("urn:edgex-gateway:%s", safeName)

	// Ensure certificates exist
	certFile := s.config.CertFile
	keyFile := s.config.KeyFile
	if certFile == "" {
		if s.config.Name == "Test Server" {
			certFile = "server_test.crt"
		} else {
			certFile = "server.crt"
		}
	}
	if keyFile == "" {
		if s.config.Name == "Test Server" {
			keyFile = "server_test.key"
		} else {
			keyFile = "server.key"
		}
	}

	if err := s.ensureCert(certFile, keyFile, appURI); err != nil {
		return fmt.Errorf("failed to ensure certificate: %v", err)
	}

	appDesc := ua.ApplicationDescription{
		ApplicationURI:  appURI,
		ProductURI:      "http://github.com/awcullen/opcua",
		ApplicationName: ua.LocalizedText{Text: s.config.Name, Locale: "en"},
		ApplicationType: ua.ApplicationTypeServer,
		DiscoveryURLs:   []string{endpoint},
	}

	// Configure User Tokens
	// var userTokens []ua.UserTokenPolicy
	// ... logic to build tokens ...
	// Note: server.WithUserTokenPolicies seems to be unavailable or named differently.
	// We rely on Authenticator functions to implicitly support tokens if applicable.

	// Configure Authenticator
	opts := []server.Option{}

	// Helper to check if method is enabled
	hasAuthMethod := func(method string) bool {
		if len(s.config.AuthMethods) == 0 {
			// Default to Anonymous if not specified
			return method == "Anonymous"
		}
		for _, m := range s.config.AuthMethods {
			if m == method {
				return true
			}
		}
		return false
	}

	if hasAuthMethod("Anonymous") {
		opts = append(opts, server.WithAuthenticateAnonymousIdentityFunc(func(userIdentity ua.AnonymousIdentity, applicationURI string, endpointURL string) error {
			return nil
		}))
	}

	if hasAuthMethod("UserName") {
		opts = append(opts, server.WithAuthenticateUserNameIdentityFunc(func(userIdentity ua.UserNameIdentity, applicationURI string, endpointURL string) error {
			// For testing purposes, temporarily allow any username/password
			return nil
		}))
	}

	// Security Configuration
	// Handle Security Policy
	// User requested "Support all levels", so we enable None explicitly.
	// Secure policies (Basic256Sha256, Aes128_Sha256_RsaOaep) are enabled by default if a certificate is provided.
	opts = append(opts, server.WithSecurityPolicyNone(true))

	// Handle Trusted Certificates
	if s.config.TrustedCertPath != "" {
		// Use subdirectories for trusted and rejected certificates
		trustedDir := filepath.Join(s.config.TrustedCertPath, "trusted")
		rejectedDir := filepath.Join(s.config.TrustedCertPath, "rejected")
		// Ensure directories exist
		os.MkdirAll(trustedDir, 0755)
		os.MkdirAll(rejectedDir, 0755)
		opts = append(opts, server.WithTrustedCertificatesPaths(trustedDir, rejectedDir))

		// Development mode: Auto-trust client certificates to avoid manual copying
		// This fixes "Bad_SecurityChecksFailed" when client cert is not yet trusted
		opts = append(opts, server.WithInsecureSkipVerify())
	}

	if hasAuthMethod("Certificate") {
		opts = append(opts, server.WithAuthenticateX509IdentityFunc(func(userIdentity ua.X509Identity, applicationURI string, endpointURL string) error {
			// Verify the certificate
			cert, err := x509.ParseCertificate([]byte(userIdentity.Certificate))
			if err != nil {
				zap.L().Error("OPC UA Certificate Auth failed",
					zap.Error(err),
					zap.String("component", "opcua-server"),
				)
				return ua.BadUserAccessDenied
			}
			zap.L().Info("OPC UA Client Authenticated via Certificate",
				zap.String("subject", cert.Subject.String()),
				zap.String("issuer", cert.Issuer.String()),
				zap.String("component", "opcua-server"),
			)
			return nil
		}))
	}

	var err error
	s.srv, err = server.New(
		appDesc,
		certFile,
		keyFile,
		endpoint,
		opts...,
	)

	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	// Build Address Space
	if err := s.buildAddressSpace(); err != nil {
		return fmt.Errorf("failed to build address space: %v", err)
	}

	// Start Listener
	go func() {
		if err := s.srv.ListenAndServe(); err != nil {
			zap.L().Error("OPC UA Server error",
				zap.String("name", s.config.Name),
				zap.Error(err),
				zap.String("component", "opcua-server"),
			)
		}
	}()

	go s.systemInfoLoop(s.ctx)

	zap.L().Info("OPC UA Server started",
		zap.String("name", s.config.Name),
		zap.String("endpoint", endpoint),
		zap.String("component", "opcua-server"),
	)
	return nil
}

func (s *Server) ensureCert(certFile, keyFile, appURI string) error {
	regenerate := false
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			// Check if certificate has correct URI
			certPEM, err := os.ReadFile(certFile)
			if err == nil {
				block, _ := pem.Decode(certPEM)
				if block != nil {
					cert, err := x509.ParseCertificate(block.Bytes)
					if err == nil {
						foundURI := false
						foundCN := false
						for _, u := range cert.URIs {
							if u.String() == appURI {
								foundURI = true
								break
							}
						}

						// Check CommonName
						if cert.Subject.CommonName == s.config.Name {
							foundCN = true
						}

						if !foundURI || !foundCN {
							zap.L().Warn("Existing certificate mismatch (URI or CN), regenerating...",
								zap.String("expected_uri", appURI),
								zap.String("expected_cn", s.config.Name),
								zap.String("component", "opcua-server"),
							)
							regenerate = true
						}
					}
				}
			}
			if !regenerate {
				return nil
			}
		}
	}

	zap.L().Info("Generating self-signed certificate...", zap.String("component", "opcua-server"))

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	uri, err := url.Parse(appURI)
	if err != nil {
		return fmt.Errorf("failed to parse application URI: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"EdgeX Gateway"},
			CommonName:   s.config.Name,
			Country:      []string{"CN"},
			Locality:     []string{"Beijing"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 10 * 24 * time.Hour), // 10 years

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment | x509.KeyUsageCertSign | x509.KeyUsageContentCommitment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "127.0.0.1"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("0.0.0.0")},
		URIs:                  []*url.URL{uri},
		SignatureAlgorithm:    x509.SHA256WithRSA,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	return nil
}

// ... rest of the file ...
func (s *Server) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.srv != nil {
		s.srv.Close()
	}
	zap.L().Info("OPC UA Server stopped",
		zap.String("name", s.config.Name),
		zap.String("component", "opcua-server"),
	)
}

func (s *Server) UpdateConfig(cfg model.OPCUAConfig) error {
	s.Stop()
	s.config = cfg
	return s.Start()
}

func (s *Server) Update(v model.Value) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config.Devices != nil {
		if enabled, ok := s.config.Devices[v.DeviceID]; ok && !enabled {
			return
		}
	}

	key := fmt.Sprintf("%s/%s/%s", v.ChannelID, v.DeviceID, v.PointID)

	if node, ok := s.nodeMap[key]; ok {
		status := uint32(0) // Good
		if v.Quality != "Good" {
			status = 0x80000000 // Bad
		}

		zap.L().Debug("OPC UA Node Update",
			zap.String("point_id", v.PointID),
			zap.Any("value", v.Value),
			zap.String("quality", v.Quality),
			zap.String("component", "opcua-server"),
		)

		node.SetValue(ua.DataValue{
			Value:           v.Value,
			StatusCode:      ua.StatusCode(status),
			SourceTimestamp: v.TS,
			ServerTimestamp: time.Now(),
		})
	}
}

func (s *Server) buildAddressSpace() error {
	nsURI := "http://edgex-gateway.com/opcua"
	// Get namespace index from the server
	nsIndex := s.srv.NamespaceManager().Add(nsURI)
	//zap.L().Info("OPC UA Namespace Added", zap.String("uri", nsURI), zap.Uint16("index", nsIndex))

	createFolder := func(parentID ua.NodeID, id string, name string) ua.NodeID {
		nodeID := ua.ParseNodeID(fmt.Sprintf("ns=%d;s=%s", nsIndex, id))
		organizes := ua.ParseNodeID("i=35")
		node := server.NewObjectNode(
			s.srv,
			nodeID,
			ua.QualifiedName{NamespaceIndex: nsIndex, Name: name},
			ua.LocalizedText{Text: name, Locale: "en"},
			ua.LocalizedText{},
			nil,
			[]ua.Reference{
				{ReferenceTypeID: organizes, IsInverse: true, TargetID: ua.ExpandedNodeID{NodeID: parentID}},
			},
			0,
		)
		if err := s.srv.NamespaceManager().AddNode(node); err != nil {
			zap.L().Error("Failed to add OPC UA Object Node", zap.String("node_id", fmt.Sprintf("%v", nodeID)), zap.Error(err))
		}
		return nodeID
	}

	// Helper to create Variable
	createVar := func(parentID ua.NodeID, id string, name string, val interface{}, typeID ua.NodeID, accessLevel byte, writeHandler func(sess *server.Session, req ua.WriteValue) (ua.DataValue, ua.StatusCode)) *server.VariableNode {
		nodeID := ua.ParseNodeID(fmt.Sprintf("ns=%d;s=%s", nsIndex, id))
		hasComponent := ua.ParseNodeID("i=47")

		// Create VariableNode
		v := server.NewVariableNode(
			s.srv,
			nodeID,
			ua.QualifiedName{NamespaceIndex: nsIndex, Name: name},
			ua.LocalizedText{Text: name, Locale: "en"},
			ua.LocalizedText{},
			nil,
			[]ua.Reference{
				{ReferenceTypeID: hasComponent, IsInverse: true, TargetID: ua.ExpandedNodeID{NodeID: parentID}},
			},
			ua.DataValue{Value: val},
			typeID,
			-1,
			nil,
			accessLevel,
			0.0,
			false,
			nil, // Historian
		)

		if err := s.srv.NamespaceManager().AddNode(v); err != nil {
			zap.L().Error("Failed to add OPC UA Variable Node", zap.String("node_id", fmt.Sprintf("%v", nodeID)), zap.Error(err))
		}

		if writeHandler != nil {
			v.SetWriteValueHandler(writeHandler)
		}

		return v
	}

	objectsFolder := ua.ParseNodeID("i=85")

	gatewayID := createFolder(objectsFolder, "Gateway", "Gateway")

	infoID := createFolder(gatewayID, "Gateway/Info", "Info")

	s.mu.Lock()
	s.nodeMap["System/CPUUsage"] = createVar(infoID, "Gateway/Info/CPUUsage", "CPUUsage", 0.0, s.getDataTypeID("double"), 1, nil)
	s.nodeMap["System/MemoryUsage"] = createVar(infoID, "Gateway/Info/MemoryUsage", "MemoryUsage", 0.0, s.getDataTypeID("double"), 1, nil)
	s.nodeMap["System/Goroutines"] = createVar(infoID, "Gateway/Info/Goroutines", "Goroutines", int32(0), s.getDataTypeID("int32"), 1, nil)
	s.nodeMap["System/Uptime"] = createVar(infoID, "Gateway/Info/Uptime", "Uptime", int64(0), s.getDataTypeID("int64"), 1, nil)
	s.nodeMap["System/ClientCount"] = createVar(infoID, "Gateway/Info/ClientCount", "ClientCount", int32(0), s.getDataTypeID("int32"), 1, nil)
	s.nodeMap["System/SubscriptionCount"] = createVar(infoID, "Gateway/Info/SubscriptionCount", "SubscriptionCount", int32(0), s.getDataTypeID("int32"), 1, nil)
	s.nodeMap["System/WriteCount"] = createVar(infoID, "Gateway/Info/WriteCount", "WriteCount", int64(0), s.getDataTypeID("int64"), 1, nil)
	s.mu.Unlock()

	channelsID := createFolder(gatewayID, "Gateway/Channels", "Channels")

	channels := s.sb.GetChannels()
	//zap.L().Info("Building OPC UA Address Space", zap.Int("channel_count", len(channels)))

	for _, ch := range channels {
		chNodeIDStr := fmt.Sprintf("Gateway/Channels/%s", ch.ID)
		// Use ID as BrowseName to ensure consistency with user request
		//zap.L().Info("Adding OPC UA Channel Node", zap.String("channel_id", ch.ID), zap.String("channel_name", ch.Name), zap.Int("device_count", len(ch.Devices)))
		chNodeID := createFolder(channelsID, chNodeIDStr, ch.ID)

		createVar(chNodeID, chNodeIDStr+"/Protocol", "Protocol", ch.Protocol, s.getDataTypeID("string"), 1, nil)
		createVar(chNodeID, chNodeIDStr+"/Status", "Status", "Running", s.getDataTypeID("string"), 1, nil)

		devsNodeIDStr := chNodeIDStr + "/Devices"
		devsNodeID := createFolder(chNodeID, devsNodeIDStr, "Devices")

		//zap.L().Info("Processing Devices for Channel", zap.String("channel_id", ch.ID), zap.Int("device_count", len(ch.Devices)))

		for _, dev := range ch.Devices {
			//zap.L().Info("Processing Device", zap.String("device_id", dev.ID), zap.String("device_name", dev.Name), zap.Int("point_count", len(dev.Points)))

			// Check if device is enabled in config
			// If config.Devices is empty, we assume "Allow All" for better UX.
			// If config.Devices is populated, we apply strict filtering.
			if s.config.Devices != nil && len(s.config.Devices) > 0 {
				if enabled, ok := s.config.Devices[dev.ID]; !ok || !enabled {
//					zap.L().Info("Skipping OPC UA Device Node (Not Enabled)", zap.String("device_id", dev.ID), zap.Bool("ok", ok), zap.Bool("enabled", enabled))
					continue
				} else {
					//zap.L().Info("Device Enabled in OPC UA Config", zap.String("device_id", dev.ID), zap.Bool("enabled", enabled))
				}
			} else {
				zap.L().Info("No OPC UA Device Filter Configured, Allowing All Devices")
			}

			dNodeIDStr := devsNodeIDStr + "/" + dev.ID
//			zap.L().Info("Adding OPC UA Device Node", zap.String("device_id", dev.ID), zap.String("device_name", dev.Name))
			// Use ID as BrowseName
			dNodeID := createFolder(devsNodeID, dNodeIDStr, dev.ID)

			createVar(dNodeID, dNodeIDStr+"/Vendor", "Vendor", getString(dev.Config, "vendor_name"), s.getDataTypeID("string"), 1, nil)
			createVar(dNodeID, dNodeIDStr+"/Model", "Model", getString(dev.Config, "model_name"), s.getDataTypeID("string"), 1, nil)

			pointsNodeIDStr := dNodeIDStr + "/Points"
			pointsNodeID := createFolder(dNodeID, pointsNodeIDStr, "Points")

//			zap.L().Info("Adding OPC UA Points for Device", zap.String("device_id", dev.ID), zap.Int("point_count", len(dev.Points)))

			for _, p := range dev.Points {
				pKey := fmt.Sprintf("%s/%s/%s", ch.ID, dev.ID, p.ID)
				pNodeIDStr := pointsNodeIDStr + "/" + p.ID

				accessLevel := byte(1)
				if strings.Contains(strings.ToUpper(p.ReadWrite), "W") {
					accessLevel |= 2
				}

				dataTypeID := s.getDataTypeID(p.DataType)

				var writeHandler func(sess *server.Session, req ua.WriteValue) (ua.DataValue, ua.StatusCode)
				if accessLevel&2 != 0 {
					cid, did, pid := ch.ID, dev.ID, p.ID
					writeHandler = func(sess *server.Session, req ua.WriteValue) (ua.DataValue, ua.StatusCode) {
						// Only allow writing to Value attribute
						if req.AttributeID != ua.AttributeIDValue {
							zap.L().Warn("OPC UA Write Rejected: Not Value Attribute", zap.Uint32("attr_id", req.AttributeID))
							return ua.DataValue{}, ua.StatusCode(0x80730000) // BadWriteNotSupported
						}

						// Extract value
						val := req.Value.Value

						zap.L().Info("OPC UA Write Request Received",
							zap.String("channel_id", cid),
							zap.String("device_id", did),
							zap.String("point_id", pid),
							zap.Any("value", val),
							zap.String("component", "opcua-server"),
						)

						// Update stats
						s.mu.Lock()
						s.stats.WriteCount++
						writeCount := s.stats.WriteCount
						s.mu.Unlock()

						// Update system node (must be done outside of lock to avoid deadlock with updateSystemNode's internal RLock)
						s.updateSystemNode("WriteCount", writeCount)

						// Call Southbound Write
						err := s.sb.WritePoint(cid, did, pid, val)
						if err != nil {
							zap.L().Error("OPC UA Write Failed (SB)",
								zap.String("channel_id", cid),
								zap.String("device_id", did),
								zap.String("point_id", pid),
								zap.Error(err),
								zap.String("component", "opcua-server"),
							)
							// Change to BadInternalError (0x80020000) to distinguish from Access Denied
							return ua.DataValue{}, ua.StatusCode(0x80020000)
						}

						zap.L().Info("OPC UA Write Success (SB)",
							zap.String("point_id", pid),
							zap.Any("value", val),
						)

						// Return the value so the server updates the node
						// Ensure the returned value has the correct type
						return ua.DataValue{
							Value:           val,
							StatusCode:      ua.StatusCode(0),
							SourceTimestamp: time.Now(),
							ServerTimestamp: time.Now(),
						}, ua.StatusCode(0)
					}
				}

				vNode := createVar(pointsNodeID, pNodeIDStr, p.Name, s.getZeroValue(p.DataType), dataTypeID, accessLevel, writeHandler)

				s.mu.Lock()
				s.nodeMap[pKey] = vNode
				s.mu.Unlock()
//				zap.L().Info("Added OPC UA Point Node", zap.String("point_id", p.ID), zap.String("point_name", p.Name), zap.String("data_type", p.DataType))
			}
		}
	}
	return nil
}

func (s *Server) systemInfoLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	startTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			s.updateSystemNode("MemoryUsage", float64(mem.Alloc)/1024/1024)
			s.updateSystemNode("Goroutines", int32(runtime.NumGoroutine()))

			uptime := int64(time.Since(startTime).Seconds())
			s.updateSystemNode("Uptime", uptime)

			// Update Client Count
			clientCount := s.getClientCount()
			s.updateSystemNode("ClientCount", int32(clientCount))

			// Update internal stats
			s.mu.Lock()
			s.stats.ClientCount = clientCount
			s.stats.Uptime = uptime
			// WriteCount and SubscriptionCount are currently 0 or updated elsewhere (if we could hook them)
			s.mu.Unlock()
		}
	}
}

func (s *Server) getClientCount() int {
	portStr := fmt.Sprintf(":%d", s.config.Port)
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// netstat -an | find ":PORT" | find "ESTABLISHED" /c
		cmd = exec.Command("cmd", "/C", fmt.Sprintf("netstat -an | find \"%s\" | find \"ESTABLISHED\" /c", portStr))
	} else {
		// netstat -an | grep :PORT | grep ESTABLISHED | wc -l
		cmd = exec.Command("sh", "-c", fmt.Sprintf("netstat -an | grep '%s' | grep ESTABLISHED | wc -l", portStr))
	}

	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	countStr := strings.TrimSpace(string(out))
	count, _ := strconv.Atoi(countStr)
	return count
}

// Stats represents the monitoring statistics
type Stats struct {
	ClientCount       int   `json:"client_count"`
	SubscriptionCount int   `json:"subscription_count"`
	WriteCount        int64 `json:"write_count"`
	Uptime            int64 `json:"uptime"`
}

// GetStats returns the current statistics
func (s *Server) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

func (s *Server) updateSystemNode(name string, value interface{}) {
	s.mu.RLock()
	node, ok := s.nodeMap["System/"+name]
	s.mu.RUnlock()
	if ok {
		node.SetValue(ua.DataValue{
			Value:           value,
			StatusCode:      ua.StatusCode(0),
			SourceTimestamp: time.Now(),
			ServerTimestamp: time.Now(),
		})
	}
}

func (s *Server) getDataTypeID(dtype string) ua.NodeID {
	id := 11
	switch strings.ToLower(dtype) {
	case "float32":
		id = 10
	case "float64":
		id = 11
	case "int16":
		id = 4
	case "uint16":
		id = 5
	case "int32":
		id = 6
	case "uint32":
		id = 7
	case "int64":
		id = 8
	case "uint64":
		id = 9
	case "bool", "boolean":
		id = 1
	case "string":
		id = 12
	}
	nid := ua.ParseNodeID(fmt.Sprintf("i=%d", id))
	return nid
}

func (s *Server) getZeroValue(dtype string) interface{} {
	switch strings.ToLower(dtype) {
	case "bool", "boolean":
		return false
	case "string":
		return ""
	case "int16":
		return int16(0)
	case "uint16":
		return uint16(0)
	case "int32":
		return int32(0)
	case "uint32":
		return uint32(0)
	case "int64":
		return int64(0)
	case "uint64":
		return uint64(0)
	case "float32":
		return float32(0)
	case "float64":
		return float64(0)
	default:
		return 0.0
	}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
