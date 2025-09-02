package env_test

import (
	"testing"
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/env"
)

func TestGetEnv_String(t *testing.T) {
	t.Setenv("TEST_VAR", "test_value")

	value := env.Get("TEST_VAR", "default_value")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}
}

func TestGetEnv_Int(t *testing.T) {
	t.Setenv("TEST_VAR", "123")

	value := env.Get("TEST_VAR", 0)
	if value != 123 {
		t.Errorf("Expected 123, got %d", value)
	}
}

func TestGetEnv_Bool(t *testing.T) {
	t.Setenv("TEST_VAR", "true")

	value := env.Get("TEST_VAR", false)
	if value != true {
		t.Errorf("Expected true, got %v", value)
	}
}

func TestGetEnv_Float(t *testing.T) {
	t.Setenv("TEST_VAR", "123.45")

	value := env.Get("TEST_VAR", 0.0)
	if value != 123.45 {
		t.Errorf("Expected 123.45, got %f", value)
	}
}

func TestGetEnv_Duration(t *testing.T) {
	t.Setenv("TEST_VAR", "1h30m")

	value := env.Get("TEST_VAR", time.Duration(0))
	if value != 1*time.Hour+30*time.Minute {
		t.Errorf("Expected 1h30m, got %v", value)
	}
}

func TestGetEnv_Fallback_String(t *testing.T) {
	t.Setenv("TEST_VAR", "")
	value := env.Get("UNSET_VAR", "fallback")
	if value != "fallback" {
		t.Errorf("Expected fallback, got %s", value)
	}
}

func TestGetEnv_Fallback_Int(t *testing.T) {
	value := env.Get("UNSET_VAR", 42)
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
}

func TestGetEnv_Fallback_Bool(t *testing.T) {
	value := env.Get("UNSET_VAR", true)
	if value != true {
		t.Errorf("Expected true, got %v", value)
	}
}

func TestGetEnv_Fallback_Float(t *testing.T) {
	value := env.Get("UNSET_VAR", 3.14)
	if value != 3.14 {
		t.Errorf("Expected 3.14, got %f", value)
	}
}

func TestGetEnv_Fallback_Duration(t *testing.T) {
	value := env.Get("UNSET_VAR", 5*time.Second)
	if value != 5*time.Second {
		t.Errorf("Expected 5s, got %v", value)
	}
}
