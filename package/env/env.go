package env

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Get retrieves the value of the environment variable with the given key.
// If the variable is not set, it returns the fallback value.
func Get[T any](key string, fallback T) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	var result T

	switch any(result).(type) {
	case bool:
		return any(strings.ToLower(value) == "true").(T)
	case int:
		i, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}

		return any(i).(T)
	case float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(err)
		}

		return any(f).(T)
	case string:
		return any(value).(T)
	case time.Duration:
		d, err := time.ParseDuration(value)
		if err != nil {
			panic(err)
		}
		return any(d).(T)
	default:
		panic("unsupported type")
	}
}
