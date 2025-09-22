package env

import (
	"os"
	"testing"
	"time"
)

func TestGet_String(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
		shouldError  bool
	}{
		{
			name:         "string value exists",
			envValue:     "test_value",
			defaultValue: "default",
			expected:     "test_value",
			shouldError:  false,
		},
		{
			name:         "string value empty - use default",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
			shouldError:  false,
		},
		{
			name:         "string value not set - use default",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
			shouldError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_STRING_VAR"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result, err := Get(key, tt.defaultValue)

			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGet_Int(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
		shouldError  bool
	}{
		{
			name:         "valid int value",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
			shouldError:  false,
		},
		{
			name:         "negative int value",
			envValue:     "-15",
			defaultValue: 10,
			expected:     -15,
			shouldError:  false,
		},
		{
			name:         "invalid int value",
			envValue:     "not_a_number",
			defaultValue: 10,
			expected:     10,
			shouldError:  true,
		},
		{
			name:         "empty value - use default",
			envValue:     "",
			defaultValue: 10,
			expected:     10,
			shouldError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT_VAR"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result, err := Get(key, tt.defaultValue)

			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGet_Int64(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int64
		expected     int64
		shouldError  bool
	}{
		{
			name:         "valid int64 value",
			envValue:     "9223372036854775807",
			defaultValue: 100,
			expected:     9223372036854775807,
			shouldError:  false,
		},
		{
			name:         "invalid int64 value",
			envValue:     "not_a_number",
			defaultValue: 100,
			expected:     100,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT64_VAR"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result, err := Get(key, tt.defaultValue)

			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGet_Float64(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue float64
		expected     float64
		shouldError  bool
	}{
		{
			name:         "valid float value",
			envValue:     "3.14159",
			defaultValue: 1.0,
			expected:     3.14159,
			shouldError:  false,
		},
		{
			name:         "integer as float",
			envValue:     "42",
			defaultValue: 1.0,
			expected:     42.0,
			shouldError:  false,
		},
		{
			name:         "invalid float value",
			envValue:     "not_a_float",
			defaultValue: 1.0,
			expected:     1.0,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_FLOAT_VAR"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result, err := Get(key, tt.defaultValue)

			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGet_Bool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
		shouldError  bool
	}{
		{
			name:         "true value",
			envValue:     "true",
			defaultValue: false,
			expected:     true,
			shouldError:  false,
		},
		{
			name:         "false value",
			envValue:     "false",
			defaultValue: true,
			expected:     false,
			shouldError:  false,
		},
		{
			name:         "1 as true",
			envValue:     "1",
			defaultValue: false,
			expected:     true,
			shouldError:  false,
		},
		{
			name:         "0 as false",
			envValue:     "0",
			defaultValue: true,
			expected:     false,
			shouldError:  false,
		},
		{
			name:         "invalid bool value",
			envValue:     "maybe",
			defaultValue: false,
			expected:     false,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_BOOL_VAR"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result, err := Get(key, tt.defaultValue)

			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGet_Duration(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
		shouldError  bool
	}{
		{
			name:         "valid duration in seconds",
			envValue:     "30s",
			defaultValue: 10 * time.Second,
			expected:     30 * time.Second,
			shouldError:  false,
		},
		{
			name:         "valid duration in minutes",
			envValue:     "5m",
			defaultValue: 1 * time.Minute,
			expected:     5 * time.Minute,
			shouldError:  false,
		},
		{
			name:         "valid duration in hours",
			envValue:     "2h",
			defaultValue: 1 * time.Hour,
			expected:     2 * time.Hour,
			shouldError:  false,
		},
		{
			name:         "invalid duration",
			envValue:     "invalid_duration",
			defaultValue: 10 * time.Second,
			expected:     10 * time.Second,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_DURATION_VAR"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result, err := Get(key, tt.defaultValue)

			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGet_UnsupportedType(t *testing.T) {
	key := "TEST_UNSUPPORTED_VAR"
	os.Setenv(key, "some_value")
	defer os.Unsetenv(key)

	type CustomType struct {
		Value string
	}

	defaultValue := CustomType{Value: "default"}
	result, err := Get(key, defaultValue)

	if err == nil {
		t.Errorf("expected error for unsupported type but got none")
	}
	if result != defaultValue {
		t.Errorf("expected default value %v, got %v", defaultValue, result)
	}
}

func TestMustGet_Success(t *testing.T) {
	key := "TEST_MUST_VAR"
	os.Setenv(key, "42")
	defer os.Unsetenv(key)

	result := MustGet(key, 10)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestMustGet_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic but function didn't panic")
		}
	}()

	key := "TEST_MUST_PANIC_VAR"
	os.Setenv(key, "not_a_number")
	defer os.Unsetenv(key)

	MustGet(key, 10)
}

func TestGetWithValidator_Success(t *testing.T) {
	key := "TEST_VALIDATOR_VAR"
	os.Setenv(key, "100")
	defer os.Unsetenv(key)

	validator := func(value int) bool {
		return value > 50
	}

	result, err := GetWithValidator(key, 10, validator)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 100 {
		t.Errorf("expected 100, got %v", result)
	}
}

func TestGetWithValidator_FailValidation(t *testing.T) {
	key := "TEST_VALIDATOR_FAIL_VAR"
	os.Setenv(key, "25")
	defer os.Unsetenv(key)

	validator := func(value int) bool {
		return value > 50
	}

	result, err := GetWithValidator(key, 10, validator)
	if err == nil {
		t.Errorf("expected validation error but got none")
	}
	if result != 10 {
		t.Errorf("expected default value 10, got %v", result)
	}
}

func TestGetWithValidator_ParseError(t *testing.T) {
	key := "TEST_VALIDATOR_PARSE_ERROR_VAR"
	os.Setenv(key, "not_a_number")
	defer os.Unsetenv(key)

	validator := func(value int) bool {
		return value > 0
	}

	result, err := GetWithValidator(key, 10, validator)
	if err == nil {
		t.Errorf("expected parse error but got none")
	}
	if result != 10 {
		t.Errorf("expected default value 10, got %v", result)
	}
}
