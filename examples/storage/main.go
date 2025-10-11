package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

type Logger struct{}

func (l *Logger) Trace(ctx context.Context, msg string, fields ...orpheus.Field) {}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...orpheus.Field) {
	fmt.Printf("[DEBUG] %s\n", msg)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...orpheus.Field) {
	fmt.Printf("[INFO] %s\n", msg)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...orpheus.Field) {
	fmt.Printf("[WARN] %s\n", msg)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...orpheus.Field) {
	fmt.Printf("[ERROR] %s", msg)
	for _, field := range fields {
		fmt.Printf(" %s=%v", field.Key, field.Value)
	}
	fmt.Println()
}

func (l *Logger) WithFields(fields ...orpheus.Field) orpheus.Logger { return l }

func main() {
	app := orpheus.New("storage-demo").
		SetDescription("Orpheus Storage Plugin Demo").
		SetVersion("1.0.0").
		SetLogger(&Logger{})

	// Convert relative path to absolute path
	pluginPath, _ := filepath.Abs("./plugins/memory.so")

	config := &orpheus.StorageConfig{
		Provider:   "memory",
		PluginPath: pluginPath,
		Config:     map[string]interface{}{},
	}

	fmt.Printf("Configuring storage with plugin: %s\n", config.PluginPath)
	app.ConfigureStorage(config)

	// Add all CLI commands
	app.Command("test", "Test plugin system", testCmd)
	app.Command("info", "Show plugin information", infoCmd)
	app.Command("set", "Set a key-value pair", setCmd)
	app.Command("get", "Get value by key", getCmd)
	app.Command("list", "List all keys", listCmd)
	app.Command("delete", "Delete a key", deleteCmd)
	app.Command("benchmark", "Run performance benchmark", benchmarkCmd)
	app.Command("security-test", "Run security tests", securityTestCmd)

	app.SetDefaultCommand("help")
	app.Run(os.Args[1:])
}

func testCmd(ctx *orpheus.Context) error {
	fmt.Println("Testing plugin system...")

	storage := ctx.Storage()
	if storage == nil {
		fmt.Println("No storage plugin loaded")
		return fmt.Errorf("no plugin loaded")
	}

	fmt.Println("Storage plugin loaded successfully!")

	// Test basic operations
	fmt.Println("Testing SET operation...")
	err := storage.Set(context.Background(), "test", []byte("hello world"))
	if err != nil {
		fmt.Printf("SET failed: %v\n", err)
		return err
	}
	fmt.Println("SET operation successful")

	fmt.Println("Testing GET operation...")
	val, err := storage.Get(context.Background(), "test")
	if err != nil {
		fmt.Printf("GET failed: %v\n", err)
		return err
	}

	fmt.Printf("GET operation successful: %s\n", string(val))
	fmt.Println("Plugin system working correctly!")
	return nil
}

func infoCmd(ctx *orpheus.Context) error {
	storage := ctx.Storage()
	if storage == nil {
		return fmt.Errorf("no storage plugin loaded")
	}

	fmt.Println("🔌 Orpheus Storage Plugin Demo")
	fmt.Println("===============================")

	err := storage.Health(context.Background())
	if err != nil {
		fmt.Printf("❌ Plugin health check failed: %v\n", err)
		return err
	}

	fmt.Println(" Plugin loaded successfully")
	fmt.Println(" Provider: memory")
	fmt.Println(" Plugin path: ./plugins/memory.so")

	stats, err := storage.Stats(context.Background())
	if err != nil {
		fmt.Printf("  Could not retrieve stats: %v\n", err)
	} else {
		fmt.Printf(" Statistics:\n")
		fmt.Printf("   Keys: %d\n", stats.TotalKeys)
		fmt.Printf("   Size: %d bytes\n", stats.TotalSize)
		fmt.Printf("   Operations: GET=%d SET=%d DELETE=%d LIST=%d\n",
			stats.GetOperations, stats.SetOperations, stats.DeleteOperations, stats.ListOperations)
	}

	return nil
}

func setCmd(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 2 {
		return fmt.Errorf("usage: set <key> <value>")
	}

	storage := ctx.Storage()
	if storage == nil {
		return fmt.Errorf("no storage plugin loaded")
	}

	key := ctx.GetArg(0)
	value := ctx.GetArg(1)

	err := storage.Set(context.Background(), key, []byte(value))
	if err != nil {
		return fmt.Errorf("failed to set key '%s': %v", key, err)
	}

	fmt.Printf(" Set '%s' = '%s'\n", key, value)
	return nil
}

func getCmd(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return fmt.Errorf("usage: get <key>")
	}

	storage := ctx.Storage()
	if storage == nil {
		return fmt.Errorf("no storage plugin loaded")
	}

	key := ctx.GetArg(0)
	value, err := storage.Get(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to get key '%s': %v", key, err)
	}

	fmt.Printf(" '%s' = '%s'\n", key, string(value))
	return nil
}

func listCmd(ctx *orpheus.Context) error {
	storage := ctx.Storage()
	if storage == nil {
		return fmt.Errorf("no storage plugin loaded")
	}

	prefix := ""
	if ctx.ArgCount() > 0 {
		prefix = ctx.GetArg(0)
	}

	keys, err := storage.List(context.Background(), prefix)
	if err != nil {
		return fmt.Errorf("failed to list keys: %v", err)
	}

	fmt.Printf(" Keys (prefix: '%s'):\n", prefix)
	for _, key := range keys {
		fmt.Printf("   - %s\n", key)
	}
	fmt.Printf("Total: %d keys\n", len(keys))
	return nil
}

func deleteCmd(ctx *orpheus.Context) error {
	if ctx.ArgCount() < 1 {
		return fmt.Errorf("usage: delete <key>")
	}

	storage := ctx.Storage()
	if storage == nil {
		return fmt.Errorf("no storage plugin loaded")
	}

	key := ctx.GetArg(0)
	err := storage.Delete(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to delete key '%s': %v", key, err)
	}

	fmt.Printf("  Deleted key '%s'\n", key)
	return nil
}

func benchmarkCmd(ctx *orpheus.Context) error {
	storage := ctx.Storage()
	if storage == nil {
		return fmt.Errorf("no storage plugin loaded")
	}

	fmt.Println("⚡ Running performance benchmark...")

	iterations := 10000
	if ctx.ArgCount() > 0 {
		if i, err := strconv.Atoi(ctx.GetArg(0)); err == nil {
			iterations = i
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

	fmt.Printf(" Benchmark Results (%d operations):\n", iterations)
	fmt.Printf("   SET: %v total, %v per operation\n", setDuration, setDuration/time.Duration(iterations))
	fmt.Printf("   GET: %v total, %v per operation\n", getDuration, getDuration/time.Duration(iterations))

	return nil
}

func securityTestCmd(ctx *orpheus.Context) error {
	storage := ctx.Storage()
	if storage == nil {
		return fmt.Errorf("no storage plugin loaded")
	}

	fmt.Println(" Running security tests...")

	// Test 1: Health check
	fmt.Print("   Health check... ")
	err := storage.Health(context.Background())
	if err != nil {
		fmt.Printf(" Failed: %v\n", err)
	} else {
		fmt.Println(" Pass")
	}

	// Test 2: Invalid key handling
	fmt.Print("   Invalid key handling... ")
	_, err = storage.Get(context.Background(), "nonexistent_key")
	if err != nil {
		fmt.Println(" Pass (correctly returns error)")
	} else {
		fmt.Println("  Warning (should return error for nonexistent key)")
	}

	// Test 3: Large value handling
	fmt.Print("   Large value handling... ")
	largeValue := make([]byte, 1024*1024) // 1MB
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}
	err = storage.Set(context.Background(), "large_key", largeValue)
	if err != nil {
		fmt.Printf(" Failed: %v\n", err)
	} else {
		fmt.Println(" Pass")
	}

	fmt.Println(" Security tests completed")
	return nil
}
