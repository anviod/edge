package logger

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogBroadcaster handles broadcasting log messages to WebSocket subscribers
type LogBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

func NewLogBroadcaster() *LogBroadcaster {
	return &LogBroadcaster{
		subscribers: make(map[chan []byte]struct{}),
	}
}

// Write implements io.Writer to broadcast logs
func (b *LogBroadcaster) Write(p []byte) (n int, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Create a copy of the data to avoid race conditions
	data := make([]byte, len(p))
	copy(data, p)

	for ch := range b.subscribers {
		select {
		case ch <- data:
		default:
			// Drop message if subscriber is slow
		}
	}
	return len(p), nil
}

func (b *LogBroadcaster) Subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan []byte, 100) // Buffer 100 logs
	b.subscribers[ch] = struct{}{}
	return ch
}

func (b *LogBroadcaster) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// AsyncRotateWriter implements io.Writer with async, batch, and drop capabilities
type AsyncRotateWriter struct {
	filename string
	maxSize  int64

	ch      chan []byte
	file    *os.File
	size    int64
	dropCnt uint64

	closeCh chan struct{}
}

// NewAsyncRotateWriter creates a new AsyncRotateWriter
func NewAsyncRotateWriter(filename string, maxSizeMB int, buffer int) *AsyncRotateWriter {
	w := &AsyncRotateWriter{
		filename: filename,
		maxSize:  int64(maxSizeMB) * 1024 * 1024,
		ch:       make(chan []byte, buffer),
		closeCh:  make(chan struct{}),
	}

	w.openFile()
	go w.loop()

	return w
}

// openFile opens the log file for writing
func (w *AsyncRotateWriter) openFile() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(w.filename), 0755); err != nil {
		return err
	}

	// Open file in append mode
	f, err := os.OpenFile(w.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Get current file size
	info, err := f.Stat()
	if err == nil {
		w.size = info.Size()
	}

	// Close old file if exists
	if w.file != nil {
		w.file.Close()
	}

	w.file = f
	return nil
}

// rotate rotates the log file
func (w *AsyncRotateWriter) rotate() error {
	// Close current file
	if w.file != nil {
		w.file.Close()
		w.file = nil
	}

	// Create archive filename
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(w.filename)
	base := strings.TrimSuffix(w.filename, ext)
	archiveName := base + "_" + timestamp + ext + ".gz"

	// Compress the log file
	if err := w.compressLogFile(w.filename, archiveName); err != nil {
		// If compression fails, just continue with new file
		os.Remove(w.filename)
	} else {
		// Remove the original file after compression
		os.Remove(w.filename)
	}

	// Open new file
	return w.openFile()
}

// compressLogFile compresses a log file using gzip
func (w *AsyncRotateWriter) compressLogFile(source, target string) error {
	// Open the source file
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	// Create the target gzip file
	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Create a gzip writer
	gzw := gzip.NewWriter(dst)
	defer gzw.Close()

	// Copy the source file to the gzip writer
	buf := make([]byte, 1024*1024) // 1MB buffer
	for {
		n, err := src.Read(buf)
		if err != nil {
			break
		}
		if n > 0 {
			if _, err := gzw.Write(buf[:n]); err != nil {
				return err
			}
		}
	}

	return nil
}

// Write implements io.Writer (non-blocking, can drop)
func (w *AsyncRotateWriter) Write(p []byte) (int, error) {
	select {
	case w.ch <- append([]byte(nil), p...): // Copy to avoid race
	default:
		// Drop log if channel is full
		w.dropCnt++
		// Log drop warning occasionally
		if w.dropCnt%1000 == 0 {
			os.Stderr.WriteString("Log dropped: " + strconv.FormatUint(w.dropCnt, 10) + "\n")
		}
	}
	return len(p), nil
}

// loop processes log messages in background
func (w *AsyncRotateWriter) loop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	buffer := make([][]byte, 0, 100)

	for {
		select {
		case p := <-w.ch:
			buffer = append(buffer, p)

			// Batch write when buffer is full
			if len(buffer) >= 50 {
				w.flush(buffer)
				buffer = buffer[:0]
			}

		case <-ticker.C:
			if len(buffer) > 0 {
				w.flush(buffer)
				buffer = buffer[:0]
			}

		case <-w.closeCh:
			if len(buffer) > 0 {
				w.flush(buffer)
			}
			return
		}
	}
}

// flush writes buffered logs to file
func (w *AsyncRotateWriter) flush(buf [][]byte) {
	for _, p := range buf {
		if w.size+int64(len(p)) > w.maxSize {
			w.rotate()
		}

		if w.file != nil {
			n, _ := w.file.Write(p)
			w.size += int64(n)
		}
	}
}

// Close closes the writer
func (w *AsyncRotateWriter) Close() error {
	close(w.closeCh)
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Stats returns the number of dropped logs
func (w *AsyncRotateWriter) Stats() uint64 {
	return w.dropCnt
}

// InitLogger initializes the global logger
func InitLogger(logLevel string, logFile string, broadcaster *LogBroadcaster) (*zap.Logger, error) {
	// Parse log level
	level := zap.InfoLevel
	if logLevel != "" {
		if l, err := zapcore.ParseLevel(strings.ToLower(logLevel)); err == nil {
			level = l
		}
	}

	// 1. Console Encoder (Colorized)
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

	// 2. File Encoder (Standard text)
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewConsoleEncoder(fileEncoderConfig)

	// 3. JSON Encoder for WebSocket (Easier to parse in frontend)
	jsonEncoderConfig := zap.NewProductionEncoderConfig()
	jsonEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(jsonEncoderConfig)

	cores := []zapcore.Core{}

	// Console Core
	cores = append(cores, zapcore.NewCore(
		consoleEncoder,
		zapcore.Lock(os.Stdout),
		level,
	))

	// File Core
	if logFile != "" {
		// Create async rotate writer with 10MB max size and 1000 buffer
		rw := NewAsyncRotateWriter(logFile, 10, 1000)
		cores = append(cores, zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(rw),
			level,
		))
	}

	// Broadcaster Core (WebSocket)
	if broadcaster != nil {
		// Always allow Debug logs for WebSocket to support real-time monitoring
		// regardless of the file/console log level.
		cores = append(cores, zapcore.NewCore(
			jsonEncoder,
			zapcore.AddSync(broadcaster),
			zap.DebugLevel,
		))
	}

	// Combine all cores
	core := zapcore.NewTee(cores...)

	// Create logger
	logger := zap.New(core, zap.AddCaller())

	// Replace global logger
	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)

	return logger, nil
}
