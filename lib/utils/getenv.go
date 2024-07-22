package utils

import (
	"log"
	"os"
	"strconv"
	"time"
)

func GetNumericEnv(name string, defValue int) time.Duration {
	stringValue, found := os.LookupEnv(name)
	if found {
		value, parseErr := strconv.Atoi(stringValue)
		if parseErr == nil {
			return time.Duration(value) * time.Second
		} else {
			log.Printf("[MAIN] cannot parse %v to integer: %v", name, parseErr)
		}
	}
	return time.Duration(defValue) * time.Second
}
