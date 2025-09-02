package config

import "time"

func init() {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	time.Local = loc
}

type Config struct{}

func New() *Config {
	c := &Config{}
	return c
}

func Load() *Config {
	c := New()
	return c
}
