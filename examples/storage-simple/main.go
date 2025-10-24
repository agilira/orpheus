package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// SimpleLogger implements Logger interface for demo
type SimpleLogger struct{}

func (l *SimpleLogger) Trace(ctx context.Context, msg string, fields ...orpheus.Field) {}

func (l *SimpleLogger) Debug(ctx context.Context, msg string, fields ...orpheus.Field) {
	fmt.Printf("[DEBUG] %s\n", msg)
}

func (l *SimpleLogger) Info(ctx context.Context, msg string, fields ...orpheus.Field) {
	fmt.Printf("[INFO] %s\n", msg)
}

func (l *SimpleLogger) Warn(ctx context.Context, msg string, fields ...orpheus.Field) {
	fmt.Printf("[WARN] %s\n", msg)
}

func (l *SimpleLogger) Error(ctx context.Context, msg string, fields ...orpheus.Field) {
	fmt.Printf("[ERROR] %s", msg)
	for _, field := range fields {
		fmt.Printf(" %s=%v", field.Key, field.Value)
	}
	fmt.Println()
}

func (l *SimpleLogger) WithFields(fields ...orpheus.Field) orpheus.Logger { return l }

// SimpleMemoryStorage provides a basic in-memory storage implementation
type SimpleMemoryStorage struct {
	data  map[string][]byte
	stats *orpheus.StorageStats
}

func NewSimpleMemoryStorage() *SimpleMemoryStorage {
	return &SimpleMemoryStorage{
		data: make(map[string][]byte),
		stats: &orpheus.StorageStats{
			Provider: "simple-memory",
		},
	}
}

func (s *SimpleMemoryStorage) Get(ctx context.Context, key string) ([]byte, error) {
	s.stats.GetOperations++
	if value, exists := s.data[key]; exists {
		// Return a copy to prevent external modification
		result := make([]byte, len(value))
		copy(result, value)
		return result, nil
	}
	s.stats.GetErrors++
	return nil, orpheus.ErrKeyNotFound
}

func (s *SimpleMemoryStorage) Set(ctx context.Context, key string, value []byte) error {
	s.stats.SetOperations++
	if key == "" {
		s.stats.SetErrors++
		return orpheus.ErrKeyEmpty
	}

	// Create a copy to prevent external modification
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	oldValue, existed := s.data[key]
	s.data[key] = valueCopy

	// Update statistics
	if existed {
		s.stats.TotalSize -= int64(len(oldValue))
	} else {
		s.stats.TotalKeys++
	}
	s.stats.TotalSize += int64(len(valueCopy))

	return nil
}

func (s *SimpleMemoryStorage) Delete(ctx context.Context, key string) error {
	s.stats.DeleteOperations++
	if value, existed := s.data[key]; existed {
		delete(s.data, key)
		s.stats.TotalKeys--
		s.stats.TotalSize -= int64(len(value))
	}
	return nil
}

func (s *SimpleMemoryStorage) List(ctx context.Context, prefix string) ([]string, error) {
	s.stats.ListOperations++
	var keys []string
	for key := range s.data {
		if prefix == "" || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *SimpleMemoryStorage) Health(ctx context.Context) error {
	return nil // Always healthy for simple in-memory storage
}

func (s *SimpleMemoryStorage) Stats(ctx context.Context) (*orpheus.StorageStats, error) {
	// Return a copy to prevent external modification
	stats := *s.stats
	return &stats, nil
}

func (s *SimpleMemoryStorage) Close() error {
	s.data = nil
	return nil
}

func main() {
	// Create storage instance directly (no plugin needed)
	storage := NewSimpleMemoryStorage()

	app := orpheus.New("storage-simple-demo").
		SetDescription("Orpheus Storage Simple Demo (No Plugins)").
		SetVersion("1.0.0").
		SetLogger(&SimpleLogger{}).
		SetStorage(storage) // Use direct storage instead of plugin system

	// Add all CLI commands
	app.Command("test", "Test storage system", testCmd)
	app.Command("info", "Show storage information", infoCmd)
	app.Command("set", "Set a key-value pair", setCmd)
	app.Command("get", "Get value by key", getCmd)
	app.Command("list", "List all keys", listCmd)
	app.Command("delete", "Delete a key", deleteCmd)
	app.Command("benchmark", "Run performance benchmark", benchmarkCmd)

	app.SetDefaultCommand("help")

	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func testCmd(ctx *orpheus.Context) error {
	fmt.Println("✅ Testing storage system...")

	storage, err := ctx.RequireStorage()
	if err != nil {
		return fmt.Errorf("storage not available: %v", err)
	}

	fmt.Println("✅ Storage instance found!")

	// Test basic operations
	fmt.Println("✅ Testing SET operation...")
	err = storage.Set(context.Background(), "test", []byte("hello world"))
	if err != nil {
		return fmt.Errorf("SET failed: %v", err)
	}
	fmt.Println("✅ SET operation successful")

	fmt.Println("✅ Testing GET operation...")
	val, err := storage.Get(context.Background(), "test")
	if err != nil {
		return fmt.Errorf("GET failed: %v", err)
	}

	fmt.Printf("✅ GET operation successful: %s\n", string(val))
	fmt.Println("✅ Storage system working correctly!")
	return nil
}

func infoCmd(ctx *orpheus.Context) error {
	storage, err := ctx.RequireStorage()
	if err != nil {
		return fmt.Errorf("storage not available: %v", err)
	}

	fmt.Println("🔌 Orpheus Storage Simple Demo")
	fmt.Println("===============================")

	err = storage.Health(context.Background())
	if err != nil {
		fmt.Printf("❌ Storage health check failed: %v\n", err)
		return err
	}

	fmt.Println("✅ Storage healthy")
	fmt.Println("📦 Provider: simple-memory (built-in)")

	stats, err := storage.Stats(context.Background())
	if err != nil {
		fmt.Printf("⚠️  Could not retrieve stats: %v\n", err)
	} else {
		fmt.Printf("📊 Statistics:\n")
		fmt.Printf("   Keys: %d\n", stats.TotalKeys)
		fmt.Printf("   Size: %d bytes\n", stats.TotalSize)
		fmt.Printf("   Operations: GET=%d SET=%d DELETE=%d LIST=%d\n",
			stats.GetOperations, stats.SetOperations, stats.DeleteOperations, stats.ListOperations)
		if stats.GetErrors > 0 || stats.SetErrors > 0 || stats.DeleteErrors > 0 || stats.ListErrors > 0 {
			fmt.Printf("   Errors: GET=%d SET=%d DELETE=%d LIST=%d\n",
				stats.GetErrors, stats.SetErrors, stats.DeleteErrors, stats.ListErrors)
		}
	}

	return nil
}

func setCmd(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 2 {
		return fmt.Errorf("usage: set <key> <value>")
	}

	storage, err := ctx.RequireStorage()
	if err != nil {
		return fmt.Errorf("storage not available: %v", err)
	}

	key := ctx.GetArg(0)
	value := ctx.GetArg(1)

	err = storage.Set(context.Background(), key, []byte(value))
	if err != nil {
		return fmt.Errorf("failed to set key '%s': %v", key, err)
	}

	fmt.Printf("✅ Set '%s' = '%s'\n", key, value)
	return nil
}

func getCmd(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return fmt.Errorf("usage: get <key>")
	}

	storage, err := ctx.RequireStorage()
	if err != nil {
		return fmt.Errorf("storage not available: %v", err)
	}

	key := ctx.GetArg(0)
	value, err := storage.Get(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to get key '%s': %v", key, err)
	}

	fmt.Printf("🔍 '%s' = '%s'\n", key, string(value))
	return nil
}

func listCmd(ctx *orpheus.Context) error {
	storage, err := ctx.RequireStorage()
	if err != nil {
		return fmt.Errorf("storage not available: %v", err)
	}

	prefix := ""
	if ctx.ArgCount() > 0 {
		prefix = ctx.GetArg(0)
	}

	keys, err := storage.List(context.Background(), prefix)
	if err != nil {
		return fmt.Errorf("failed to list keys: %v", err)
	}

	fmt.Printf("📋 Keys (prefix: '%s'):\n", prefix)
	for _, key := range keys {
		fmt.Printf("   - %s\n", key)
	}
	fmt.Printf("📊 Total: %d keys\n", len(keys))
	return nil
}

func deleteCmd(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return fmt.Errorf("usage: delete <key>")
	}

	storage, err := ctx.RequireStorage()
	if err != nil {
		return fmt.Errorf("storage not available: %v", err)
	}

	key := ctx.GetArg(0)
	err = storage.Delete(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to delete key '%s': %v", key, err)
	}

	fmt.Printf("🗑️  Deleted key '%s'\n", key)
	return nil
}

func benchmarkCmd(ctx *orpheus.Context) error {
	storage, err := ctx.RequireStorage()
	if err != nil {
		return fmt.Errorf("storage not available: %v", err)
	}

	fmt.Println("⚡ Running performance benchmark...")

	iterations := 1000
	if ctx.ArgCount() > 0 {
		// Simple string to int conversion
		if arg := ctx.GetArg(0); arg != "" {
			var parsed int
			for _, r := range arg {
				if r >= '0' && r <= '9' {
					parsed = parsed*10 + int(r-'0')
				} else {
					break
				}
			}
			if parsed > 0 {
				iterations = parsed
			}
		}
	}

	// SET benchmark
	start := time.Now()
	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		value := fmt.Sprintf("bench_value_%d", i)
		err := storage.Set(context.Background(), key, []byte(value))
		if err != nil {
			return fmt.Errorf("benchmark SET failed: %v", err)
		}
	}
	setDuration := time.Since(start)

	// GET benchmark
	start = time.Now()
	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		_, err := storage.Get(context.Background(), key)
		if err != nil {
			return fmt.Errorf("benchmark GET failed: %v", err)
		}
	}
	getDuration := time.Since(start)

	fmt.Printf("🏃 Benchmark Results (%d operations):\n", iterations)
	fmt.Printf("   SET: %v total, %v per operation\n", setDuration, setDuration/time.Duration(iterations))
	fmt.Printf("   GET: %v total, %v per operation\n", getDuration, getDuration/time.Duration(iterations))

	return nil
}
