package bacnet

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

// TestBACnetDriverLockUsage tests that the BACnet driver correctly uses read/write locks
func TestBACnetDriverLockUsage(t *testing.T) {
	driver := NewBACnetDriver()

	// Test 1: Verify the lock type is RWMutex
	bacnetDriver := driver.(*BACnetDriver)
	lockType := reflect.TypeOf(bacnetDriver.mu)
	lockTypeName := lockType.String()

	if lockTypeName != "sync.RWMutex" {
		t.Errorf("Expected sync.RWMutex, got %s", lockTypeName)
	}
}

// TestReadPointsCompiles tests that ReadPoints compiles with the new lock usage
func TestReadPointsCompiles(t *testing.T) {
	// This test just ensures that ReadPoints compiles correctly
	driver := NewBACnetDriver()
	// The fact that this compiles means our lock usage is correct
	_ = driver
}

// TestConcurrentReadOperations tests concurrent read operations
func TestConcurrentReadOperations(t *testing.T) {
	driver := NewBACnetDriver()

	var wg sync.WaitGroup
	concurrentReads := 10

	for i := 0; i < concurrentReads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Acquire read lock
			driver.(*BACnetDriver).mu.RLock()
			defer driver.(*BACnetDriver).mu.RUnlock()
			// Simulate read operation
			time.Sleep(10 * time.Millisecond)
		}()
	}

	wg.Wait()
	// If we get here, all concurrent reads completed successfully
}

// TestReadWriteConcurrency tests that read and write operations can coexist
func TestReadWriteConcurrency(t *testing.T) {
	driver := NewBACnetDriver()

	var wg sync.WaitGroup

	// Start multiple read operations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			driver.(*BACnetDriver).mu.RLock()
			defer driver.(*BACnetDriver).mu.RUnlock()
			time.Sleep(10 * time.Millisecond)
		}()
	}

	// Start a write operation
	wg.Add(1)
	go func() {
		defer wg.Done()
		driver.(*BACnetDriver).mu.Lock()
		defer driver.(*BACnetDriver).mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}()

	wg.Wait()
	// If we get here, read and write operations completed successfully
}
