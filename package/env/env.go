package env

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func Get[T any](key string, defaultValue T) (T, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	var result any
	var err error

	switch any(defaultValue).(type) {
	case string:
		result = value
	case int:
		var intVal int64
		intVal, err = strconv.ParseInt(value, 10, 32)
		if err == nil {
			result = int(intVal)
		}
	case int64:
		result, err = strconv.ParseInt(value, 10, 64)
	case float64:
		result, err = strconv.ParseFloat(value, 64)
	case bool:
		result, err = strconv.ParseBool(value)
	case time.Duration:
		result, err = time.ParseDuration(value)
	default:
		return defaultValue, fmt.Errorf("unsupported type for environment variable %s", key)
	}

	if err != nil {
		return defaultValue, fmt.Errorf("failed to parse environment variable %s: %w", key, err)
	}

	return result.(T), nil
}

func MustGet[T any](key string, defaultValue T) T {
	value, err := Get(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("failed to get environment variable %s: %v", key, err))
	}
	return value
}

func GetWithValidator[T any](key string, defaultValue T, validator func(T) bool) (T, error) {
	value, err := Get(key, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	if !validator(value) {
		return defaultValue, fmt.Errorf("environment variable %s failed validation", key)
	}

	return value, nil
}
