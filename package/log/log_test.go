package log

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestPreforkHook_Interface(t *testing.T) {
	var _ zerolog.Hook = prefork{}
}
func TestPreforkHook_ChildProcess(t *testing.T) {
	var buf bytes.Buffer

	logger := zerolog.New(&buf).Hook(prefork{})

	hook := prefork{}

	event := logger.Info()

	hook.Run(event, zerolog.InfoLevel, "test message")

	if hook == (prefork{}) {

	}
}
func TestPreforkHook_Structure(t *testing.T) {
	hook := prefork{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("prefork.Run should not panic: %v", r)
		}
	}()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	event := logger.Info()

	hook.Run(event, zerolog.InfoLevel, "test message")
}
func TestNew_LoggerCreation(t *testing.T) {
	logger := New()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logger creation should not panic: %v", r)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logging should not panic: %v", r)
		}
	}()

	logger.Info().Msg("test message")
}
func TestNew_GlobalLevelConfiguration(t *testing.T) {
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)

	_ = New()

	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("expected global level to be InfoLevel, got %v", zerolog.GlobalLevel())
	}
}
func TestNew_TimeFormatConfiguration(t *testing.T) {
	originalTimeFormat := zerolog.TimeFieldFormat
	defer func() { zerolog.TimeFieldFormat = originalTimeFormat }()

	_ = New()

	expectedFormat := zerolog.TimeFormatUnixMs
	if zerolog.TimeFieldFormat != expectedFormat {
		t.Errorf("expected time format to be %s, got %s", expectedFormat, zerolog.TimeFieldFormat)
	}
}
func TestLogger_LevelsFiltering(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := New()

	logger.Debug().Msg("debug message")
	logger.Info().Msg("info message")
	logger.Warn().Msg("warn message")
	logger.Error().Msg("error message")

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if strings.Contains(output, "debug message") {
		t.Error("debug message should be filtered out at Info level")
	}

	if !strings.Contains(output, "info message") {
		t.Error("info message should be present")
	}
}
func TestLogger_ContextFields(t *testing.T) {
	var buf bytes.Buffer

	logger := zerolog.New(&buf).With().Timestamp().Caller().Logger()

	logger.Info().
		Str("service", "test").
		Int("port", 8080).
		Bool("debug", true).
		Msg("test message with context")

	output := buf.String()

	if !strings.Contains(output, "test message with context") {
		t.Error("log message should be present in output")
	}
	if !strings.Contains(output, "service") {
		t.Error("service field should be present in output")
	}
	if !strings.Contains(output, "test") {
		t.Error("service value should be present in output")
	}
}
func TestLogger_JSONOutput(t *testing.T) {
	var buf bytes.Buffer

	logger := zerolog.New(&buf).With().Timestamp().Logger()

	logger.Info().
		Str("component", "test").
		Int("value", 42).
		Msg("json test message")

	output := buf.String()

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("log output should be valid JSON: %v", err)
	}

	if logEntry["level"] != "info" {
		t.Error("level field should be 'info'")
	}
	if logEntry["message"] != "json test message" {
		t.Error("message field should match logged message")
	}
	if logEntry["component"] != "test" {
		t.Error("component field should be present")
	}
	if logEntry["value"] != float64(42) {
		t.Error("value field should be present and equal to 42")
	}

	if _, exists := logEntry["time"]; !exists {
		t.Error("timestamp field should be present")
	}
}
func TestNew_ConsoleWriterConfig(t *testing.T) {

	logger := New()

	subLogger := logger.With().Str("component", "test").Logger()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("console logging should not panic: %v", r)
		}
	}()

	subLogger.Info().Msg("console writer test")
}
func TestNew_HookAttachment(t *testing.T) {
	logger := New()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logging with hooks should not panic: %v", r)
		}
	}()

	logger.Info().Str("test", "hook").Msg("hook test message")
}
func TestLogger_CallerInfo(t *testing.T) {
	var buf bytes.Buffer

	logger := zerolog.New(&buf).With().Caller().Logger()

	logger.Info().Msg("caller test message")

	output := buf.String()

	if !strings.Contains(output, "log_test.go") {
		t.Error("caller information should include filename")
	}
}
func TestLogger_ErrorHandling(t *testing.T) {
	logger := New()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logging with nil should not panic: %v", r)
		}
	}()

	logger.Info().
		Str("nil_test", "").
		Interface("nil_interface", nil).
		Msg("error handling test")
}
func TestLogger_ConcurrentUsage(t *testing.T) {
	logger := New()

	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("concurrent logging should not panic: %v", r)
				}
				done <- true
			}()

			logger.Info().Int("goroutine", id).Msg("concurrent test")
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
func TestLogger_TimeFormat(t *testing.T) {
	var buf bytes.Buffer

	logger := zerolog.New(&buf).With().Timestamp().Logger()

	logger.Info().Msg("time format test")

	output := buf.String()

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("log output should be valid JSON: %v", err)
	}

	timeField, exists := logEntry["time"]
	if !exists {
		t.Error("time field should be present")
	}

	if timeStr, ok := timeField.(string); ok {
		if timeStr == "" {
			t.Error("timestamp should not be empty")
		}
	} else if timeNum, ok := timeField.(float64); ok {
		if timeNum <= 0 {
			t.Error("timestamp should be positive")
		}
	} else {
		t.Error("timestamp should be either string or number")
	}
}
