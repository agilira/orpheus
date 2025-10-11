package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// TestLogger captures log output for testing
type TestLogger struct {
	logs []string
}

func (l *TestLogger) Trace(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.logs = append(l.logs, "[TRACE] "+msg)
}

func (l *TestLogger) Debug(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.logs = append(l.logs, "[DEBUG] "+msg)
}

func (l *TestLogger) Info(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.logs = append(l.logs, "[INFO] "+msg)
}

func (l *TestLogger) Warn(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.logs = append(l.logs, "[WARN] "+msg)
}

func (l *TestLogger) Error(ctx context.Context, msg string, fields ...orpheus.Field) {
	msg_with_fields := "[ERROR] " + msg
	for _, field := range fields {
		msg_with_fields += " " + field.Key + "=" + field.Value.(string)
	}
	l.logs = append(l.logs, msg_with_fields)
}

func (l *TestLogger) WithFields(fields ...orpheus.Field) orpheus.Logger { return l }

func (l *TestLogger) GetLogs() []string { return l.logs }

func (l *TestLogger) Clear() { l.logs = []string{} }

func setupTestApp(t *testing.T) (*orpheus.App, *TestLogger) {
	logger := &TestLogger{}
	app := orpheus.New("storage-demo-test").
		SetDescription("Orpheus Storage Plugin Demo Test").
		SetVersion("1.0.0").
		SetLogger(logger)

	// Check if plugin exists
	pluginPath, _ := filepath.Abs("./plugins/memory.so")
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		t.Skipf("Plugin not found at %s. Run build_plugins.sh first.", pluginPath)
	}

	config := &orpheus.StorageConfig{
		Provider:   "memory",
		PluginPath: pluginPath,
		Config:     map[string]interface{}{},
	}

	app.ConfigureStorage(config)

	// Add all commands
	app.Command("test", "Test plugin system", testCmd)
	app.Command("info", "Show plugin information", infoCmd)
	app.Command("set", "Set a key-value pair", setCmd)
	app.Command("get", "Get value by key", getCmd)
	app.Command("list", "List all keys", listCmd)
	app.Command("delete", "Delete a key", deleteCmd)
	app.Command("benchmark", "Run performance benchmark", benchmarkCmd)
	app.Command("security-test", "Run security tests", securityTestCmd)

	return app, logger
}

func TestStorageDemo_InfoCommand(t *testing.T) {
	app, logger := setupTestApp(t)
	logger.Clear()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := app.Run([]string{"info"})

	w.Close()
	os.Stdout = oldStdout

	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	if err != nil {
		t.Errorf("Info command failed: %v", err)
	}

	// Check output contains expected information
	if !strings.Contains(outputStr, "Orpheus Storage Plugin Demo") {
		t.Error("Output should contain demo title")
	}
	if !strings.Contains(outputStr, "Plugin loaded successfully") {
		t.Error("Output should indicate plugin loaded successfully")
	}
	if !strings.Contains(outputStr, "Provider: memory") {
		t.Error("Output should show memory provider")
	}
}

func TestStorageDemo_SetGetCommands(t *testing.T) {
	app, logger := setupTestApp(t)
	logger.Clear()

	tests := []struct {
		name        string
		setArgs     []string
		getArgs     []string
		expectError bool
		description string
	}{
		{
			name:        "ValidSetGet",
			setArgs:     []string{"set", "testkey", "testvalue"},
			getArgs:     []string{"get", "testkey"},
			expectError: false,
			description: "Valid set and get operations",
		},
		{
			name:        "SetWithSpaces",
			setArgs:     []string{"set", "spacekey", "value with spaces"},
			getArgs:     []string{"get", "spacekey"},
			expectError: false,
			description: "Set and get value with spaces",
		},
		{
			name:        "SetSpecialChars",
			setArgs:     []string{"set", "special_key", "value!@#$%^&*()"},
			getArgs:     []string{"get", "special_key"},
			expectError: false,
			description: "Set and get value with special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test SET command
			err := app.Run(tt.setArgs)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}

			if !tt.expectError {
				// Test GET command
				err = app.Run(tt.getArgs)
				if err != nil {
					t.Errorf("GET command failed for %s: %v", tt.description, err)
				}
			}
		})
	}
}

func TestStorageDemo_SetCommandErrors(t *testing.T) {
	app, _ := setupTestApp(t)

	tests := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "NoArguments",
			args:        []string{"set"},
			description: "Set command with no arguments",
		},
		{
			name:        "OneArgument",
			args:        []string{"set", "key"},
			description: "Set command with only key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run(tt.args)
			if err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !strings.Contains(err.Error(), "usage: set <key> <value>") {
				t.Errorf("Error message should contain usage information, got: %v", err)
			}
		})
	}
}

func TestStorageDemo_GetCommandErrors(t *testing.T) {
	app, _ := setupTestApp(t)

	tests := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "NoArguments",
			args:        []string{"get"},
			description: "Get command with no arguments",
		},
		{
			name:        "NonexistentKey",
			args:        []string{"get", "nonexistent_key_12345"},
			description: "Get command with nonexistent key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run(tt.args)
			if err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}

			if tt.name == "NoArguments" && !strings.Contains(err.Error(), "usage: get <key>") {
				t.Errorf("Error message should contain usage information, got: %v", err)
			}
		})
	}
}

func TestStorageDemo_ListCommand(t *testing.T) {
	app, _ := setupTestApp(t)

	// Set up test data
	testData := map[string]string{
		"prefix_key1": "value1",
		"prefix_key2": "value2",
		"other_key":   "value3",
	}

	// Insert test data
	for key, value := range testData {
		err := app.Run([]string{"set", key, value})
		if err != nil {
			t.Fatalf("Failed to set test data: %v", err)
		}
	}

	tests := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "ListAll",
			args:        []string{"list"},
			description: "List all keys",
		},
		{
			name:        "ListWithPrefix",
			args:        []string{"list", "prefix"},
			description: "List keys with prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := app.Run(tt.args)

			w.Close()
			os.Stdout = oldStdout

			output := make([]byte, 1024)
			n, _ := r.Read(output)
			outputStr := string(output[:n])

			if err != nil {
				t.Errorf("List command failed for %s: %v", tt.description, err)
			}

			if !strings.Contains(outputStr, "Total:") {
				t.Error("Output should contain total count")
			}
		})
	}
}

func TestStorageDemo_DeleteCommand(t *testing.T) {
	app, _ := setupTestApp(t)

	// Set up test data
	err := app.Run([]string{"set", "delete_test_key", "delete_test_value"})
	if err != nil {
		t.Fatalf("Failed to set test data: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "ValidDelete",
			args:        []string{"delete", "delete_test_key"},
			expectError: false,
			description: "Valid delete operation",
		},
		{
			name:        "NoArguments",
			args:        []string{"delete"},
			expectError: true,
			description: "Delete command with no arguments",
		},
		{
			name:        "DeleteNonexistent",
			args:        []string{"delete", "nonexistent_delete_key"},
			expectError: false, // Memory plugin is idempotent - no error for deleting nonexistent key
			description: "Delete nonexistent key (idempotent operation)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run(tt.args)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}

			if tt.name == "NoArguments" && !strings.Contains(err.Error(), "usage: delete <key>") {
				t.Errorf("Error message should contain usage information, got: %v", err)
			}
		})
	}
}

func TestStorageDemo_TestCommand(t *testing.T) {
	app, _ := setupTestApp(t)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := app.Run([]string{"test"})

	w.Close()
	os.Stdout = oldStdout

	output := make([]byte, 2048)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	if err != nil {
		t.Errorf("Test command failed: %v", err)
	}

	expectedStrings := []string{
		"Testing plugin system",
		"Storage plugin loaded successfully",
		"Testing SET operation",
		"SET operation successful",
		"Testing GET operation",
		"GET operation successful",
		"Plugin system working correctly",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Output should contain '%s', got: %s", expected, outputStr)
		}
	}
}

func TestStorageDemo_BenchmarkCommand(t *testing.T) {
	app, _ := setupTestApp(t)

	tests := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "DefaultBenchmark",
			args:        []string{"benchmark"},
			description: "Benchmark with default iterations",
		},
		{
			name:        "CustomBenchmark",
			args:        []string{"benchmark", "100"},
			description: "Benchmark with custom iterations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			start := time.Now()
			err := app.Run(tt.args)
			duration := time.Since(start)

			w.Close()
			os.Stdout = oldStdout

			output := make([]byte, 2048)
			n, _ := r.Read(output)
			outputStr := string(output[:n])

			if err != nil {
				t.Errorf("Benchmark command failed for %s: %v", tt.description, err)
			}

			// Check that benchmark completed in reasonable time
			if duration > 30*time.Second {
				t.Errorf("Benchmark took too long: %v", duration)
			}

			// Check output format
			if !strings.Contains(outputStr, "Benchmark Results") {
				t.Error("Output should contain benchmark results")
			}
			if !strings.Contains(outputStr, "SET:") {
				t.Error("Output should contain SET benchmark")
			}
			if !strings.Contains(outputStr, "GET:") {
				t.Error("Output should contain GET benchmark")
			}
		})
	}
}

func TestStorageDemo_SecurityTestCommand(t *testing.T) {
	app, _ := setupTestApp(t)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := app.Run([]string{"security-test"})

	w.Close()
	os.Stdout = oldStdout

	output := make([]byte, 2048)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	if err != nil {
		t.Errorf("Security test command failed: %v", err)
	}

	expectedStrings := []string{
		"Running security tests",
		"Health check",
		"Invalid key handling",
		"Large value handling",
		"Security tests completed",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Output should contain '%s', got: %s", expected, outputStr)
		}
	}
}

func TestStorageDemo_NoStoragePlugin(t *testing.T) {
	// Create app without storage configuration
	app := orpheus.New("storage-demo-test-no-plugin").
		SetDescription("Test without storage plugin").
		SetVersion("1.0.0")

	// Add commands
	app.Command("info", "Show plugin information", infoCmd)
	app.Command("set", "Set a key-value pair", setCmd)
	app.Command("get", "Get value by key", getCmd)

	tests := []struct {
		name string
		args []string
	}{
		{"InfoNoStorage", []string{"info"}},
		{"SetNoStorage", []string{"set", "key", "value"}},
		{"GetNoStorage", []string{"get", "key"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run(tt.args)
			if err == nil {
				t.Errorf("Expected error for %s when no storage plugin is loaded", tt.name)
			}
			if !strings.Contains(err.Error(), "no storage plugin loaded") {
				t.Errorf("Error should mention no storage plugin loaded, got: %v", err)
			}
		})
	}
}

func TestStorageDemo_IntegrationWorkflow(t *testing.T) {
	app, _ := setupTestApp(t)

	// Test a complete workflow
	workflow := []struct {
		command     []string
		description string
		expectError bool
	}{
		{[]string{"info"}, "Check plugin info", false},
		{[]string{"set", "workflow_key1", "workflow_value1"}, "Set first key", false},
		{[]string{"set", "workflow_key2", "workflow_value2"}, "Set second key", false},
		{[]string{"get", "workflow_key1"}, "Get first key", false},
		{[]string{"list", "workflow"}, "List workflow keys", false},
		{[]string{"delete", "workflow_key1"}, "Delete first key", false},
		{[]string{"get", "workflow_key1"}, "Try to get deleted key", true},
		{[]string{"list", "workflow"}, "List remaining keys", false},
		{[]string{"delete", "workflow_key2"}, "Delete second key", false},
	}

	for i, step := range workflow {
		t.Run(step.description, func(t *testing.T) {
			err := app.Run(step.command)
			if step.expectError && err == nil {
				t.Errorf("Step %d (%s): Expected error but got none", i+1, step.description)
			}
			if !step.expectError && err != nil {
				t.Errorf("Step %d (%s): Unexpected error: %v", i+1, step.description, err)
			}
		})
	}
}

func TestStorageDemo_ErrorHandling(t *testing.T) {
	app, _ := setupTestApp(t)

	// Test various error conditions
	errorTests := []struct {
		name        string
		args        []string
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name:        "InvalidCommand",
			args:        []string{"invalid_command"},
			expectError: true,
			errorCheck:  func(err error) bool { return err != nil },
		},
		{
			name:        "EmptyArgs",
			args:        []string{},
			expectError: false, // Should show help
			errorCheck:  func(err error) bool { return err == nil },
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run(tt.args)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
			if err != nil && tt.errorCheck != nil && !tt.errorCheck(err) {
				t.Errorf("Error check failed for %s: %v", tt.name, err)
			}
		})
	}
}

// Benchmark tests
func BenchmarkStorageDemo_SetOperation(b *testing.B) {
	app, _ := setupTestApp(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench_key"
		value := "bench_value"
		suppressOutput(func() {
			err := app.Run([]string{"set", key, value})
			if err != nil {
				b.Fatalf("SET operation failed: %v", err)
			}
		})
	}
}

func BenchmarkStorageDemo_GetOperation(b *testing.B) {
	app, _ := setupTestApp(&testing.T{})

	// Set up test data
	suppressOutput(func() {
		err := app.Run([]string{"set", "bench_key", "bench_value"})
		if err != nil {
			b.Fatalf("Failed to set test data: %v", err)
		}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		suppressOutput(func() {
			err := app.Run([]string{"get", "bench_key"})
			if err != nil {
				b.Fatalf("GET operation failed: %v", err)
			}
		})
	}
}

// Helper function to suppress output during tests
func suppressOutput(f func()) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Redirect to /dev/null equivalent
	devNull, _ := os.Open(os.DevNull)
	defer devNull.Close()

	os.Stdout = devNull
	os.Stderr = devNull

	f()

	os.Stdout = oldStdout
	os.Stderr = oldStderr
}
