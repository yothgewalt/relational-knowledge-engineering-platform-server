package config

import "time"

func init() {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	time.Local = loc
}
