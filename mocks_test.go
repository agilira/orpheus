// mocks_test.go: Logger and Storage test doubles for the App tests
//
// The storage package has its own copy in storage/plugin_test.go: a shared
// package would need to import storage, which storage's white-box tests
// cannot import back.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/agilira/orpheus/observe"
	"github.com/agilira/orpheus/storage"
)

// MockLogger implements Logger interface for testing
type MockLogger struct {
	logs []LogEntry
	mu   sync.Mutex
}

type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

func (m *MockLogger) Trace(ctx context.Context, msg string, fields ...observe.Field) {
	m.addLog("TRACE", msg, fields)
}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...observe.Field) {
	m.addLog("DEBUG", msg, fields)
}

func (m *MockLogger) Info(ctx context.Context, msg string, fields ...observe.Field) {
	m.addLog("INFO", msg, fields)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...observe.Field) {
	m.addLog("WARN", msg, fields)
}

func (m *MockLogger) Error(ctx context.Context, msg string, fields ...observe.Field) {
	m.addLog("ERROR", msg, fields)
}

func (m *MockLogger) WithFields(fields ...observe.Field) observe.Logger {
	// For testing purposes, return the same logger
	// In a real implementation, this would return a new logger with additional fields
	return m
}

func (m *MockLogger) addLog(level, msg string, fields []observe.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fieldMap := make(map[string]interface{})
	for _, f := range fields {
		fieldMap[f.Key] = f.Value
	}

	m.logs = append(m.logs, LogEntry{
		Level:   level,
		Message: msg,
		Fields:  fieldMap,
	})
}

func (m *MockLogger) GetLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]LogEntry, len(m.logs))
	copy(result, m.logs)
	return result
}

func (m *MockLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = nil
}

type MockStorage struct {
	data        map[string][]byte
	mu          sync.RWMutex
	healthError error
	closed      bool
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string][]byte),
	}
}

func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, storage.ErrClosed
	}

	value, exists := m.data[key]
	if !exists {
		return nil, storage.ErrKeyNotFound
	}

	return value, nil
}

func (m *MockStorage) Set(ctx context.Context, key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return storage.ErrClosed
	}

	m.data[key] = value
	return nil
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return storage.ErrClosed
	}

	delete(m.data, key)
	return nil
}

func (m *MockStorage) List(ctx context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, storage.ErrClosed
	}

	var keys []string
	for key := range m.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (m *MockStorage) Health(ctx context.Context) error {
	if m.closed {
		return storage.ErrClosed
	}
	return m.healthError
}

func (m *MockStorage) Stats(ctx context.Context) (*storage.Stats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, storage.ErrClosed
	}

	totalSize := int64(0)
	for _, value := range m.data {
		totalSize += int64(len(value))
	}

	return &storage.Stats{
		TotalKeys: int64(len(m.data)),
		TotalSize: totalSize,
		Provider:  "mock",
		Uptime:    time.Hour,
	}, nil
}

func (m *MockStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

func (m *MockStorage) SetHealthError(err error) {
	m.healthError = err
}

// Test helper functions
