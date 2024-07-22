package utils

import (
	"log"
	"os"
	"strconv"
	"time"
)

func GetDurationEnv(name string) time.Duration {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		log.Fatalf("[MAIN] cannot parse %v to integer: %v", name, err)
	}
	return time.Duration(value) * time.Second
}
