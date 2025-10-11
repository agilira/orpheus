// Memory Storage Provider Plugin - Ultra-fast in-memory storage
//
// This demonstrates a simple in-memory storage provider implementation
// as a dynamic plugin for the Orpheus framework.
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// MemoryStorage provides in-memory key-value storage
type MemoryStorage struct {
	data  map[string][]byte
	mutex sync.RWMutex
	stats *orpheus.StorageStats
}

// NewMemoryStorage creates a new memory storage instance
func NewMemoryStorage(config map[string]interface{}) (orpheus.Storage, error) {
	return &MemoryStorage{
		data: make(map[string][]byte),
		stats: &orpheus.StorageStats{
			TotalKeys: 0,
			TotalSize: 0,
		},
	}, nil
}

// Get retrieves a value by key
func (m *MemoryStorage) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	defer func() {
		m.mutex.Lock()
		m.stats.GetOperations++
		m.mutex.Unlock()
	}()

	m.mutex.RLock()
	value, exists := m.data[key]
	m.mutex.RUnlock()

	if !exists {
		m.mutex.Lock()
		m.stats.GetErrors++
		m.mutex.Unlock()
		return nil, orpheus.NewError("NOT_FOUND", "key", fmt.Sprintf("Key '%s' not found", key))
	}

	// Create a copy to prevent external modification
	result := make([]byte, len(value))
	copy(result, value)

	_ = time.Since(start) // Could be used for latency tracking
	return result, nil
}

// Set stores a value with the given key
func (m *MemoryStorage) Set(ctx context.Context, key string, value []byte) error {
	start := time.Now()
	defer func() {
		m.mutex.Lock()
		m.stats.SetOperations++
		m.mutex.Unlock()
	}()

	// Create a copy to prevent external modification
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	oldValue, existed := m.data[key]
	m.data[key] = valueCopy

	// Update statistics
	if existed {
		m.stats.TotalSize -= int64(len(oldValue))
	} else {
		m.stats.TotalKeys++
	}
	m.stats.TotalSize += int64(len(valueCopy))

	_ = time.Since(start) // Could be used for latency tracking
	return nil
}

// Delete removes a key and its value
func (m *MemoryStorage) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		m.mutex.Lock()
		m.stats.DeleteOperations++
		m.mutex.Unlock()
	}()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if value, existed := m.data[key]; existed {
		delete(m.data, key)
		m.stats.TotalKeys--
		m.stats.TotalSize -= int64(len(value))
	}

	_ = time.Since(start) // Could be used for latency tracking
	return nil
}

// List returns all keys matching the given prefix
func (m *MemoryStorage) List(ctx context.Context, prefix string) ([]string, error) {
	start := time.Now()
	defer func() {
		m.mutex.Lock()
		m.stats.ListOperations++
		m.mutex.Unlock()
	}()

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var keys []string
	for key := range m.data {
		if prefix == "" || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)

	_ = time.Since(start) // Could be used for latency tracking
	return keys, nil
}

// Health performs a health check
func (m *MemoryStorage) Health(ctx context.Context) error {
	// Memory storage is always healthy unless there's a catastrophic failure
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.data == nil {
		return orpheus.NewError("HEALTH", "storage", "Memory storage data map is nil")
	}

	return nil
}

// Stats returns operational statistics
func (m *MemoryStorage) Stats(ctx context.Context) (*orpheus.StorageStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent external modification
	return &orpheus.StorageStats{
		TotalKeys:        m.stats.TotalKeys,
		TotalSize:        m.stats.TotalSize,
		GetOperations:    m.stats.GetOperations,
		SetOperations:    m.stats.SetOperations,
		DeleteOperations: m.stats.DeleteOperations,
		ListOperations:   m.stats.ListOperations,
		GetErrors:        m.stats.GetErrors,
		SetErrors:        m.stats.SetErrors,
		DeleteErrors:     m.stats.DeleteErrors,
		ListErrors:       m.stats.ListErrors,
	}, nil
}

// Close releases resources (no-op for memory storage)
func (m *MemoryStorage) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear the data map for garbage collection
	m.data = nil
	return nil
}

// MemoryStoragePlugin implements the storage plugin interface
type MemoryStoragePlugin struct{}

// Name returns the plugin name
func (p *MemoryStoragePlugin) Name() string {
	return "memory"
}

// Description returns the plugin description
func (p *MemoryStoragePlugin) Description() string {
	return "Ultra-fast in-memory key-value storage"
}

// Version returns the plugin version
func (p *MemoryStoragePlugin) Version() string {
	return "1.0.0"
}

// Validate validates the configuration
func (p *MemoryStoragePlugin) Validate(config map[string]interface{}) error {
	// Memory storage doesn't need any configuration validation
	return nil
}

// DefaultConfig returns the default configuration
func (p *MemoryStoragePlugin) DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"initial_capacity": 1000,
		"enable_stats":     true,
	}
}

// New creates a new storage instance
func (p *MemoryStoragePlugin) New(config map[string]interface{}) (orpheus.Storage, error) {
	return NewMemoryStorage(config)
}

// NewStoragePlugin creates a new memory storage plugin (required for plugin loading)
func NewStoragePlugin() orpheus.StoragePlugin {
	return &MemoryStoragePlugin{}
}
