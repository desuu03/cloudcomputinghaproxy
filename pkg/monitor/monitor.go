package monitor

import (
	"log"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

var (
	mu           sync.RWMutex
	lastUsage    float64
	upperLimit   = 80.0
	lowerLimit   = 20.0
	checkInterval = 5 * time.Second
)

type Config struct {
	UpperThreshold float64
	LowerThreshold float64
	Interval       time.Duration
}

func SetConfig(c Config) {
	mu.Lock()
	defer mu.Unlock()
	if c.UpperThreshold > 0 {
		upperLimit = c.UpperThreshold
	}
	if c.LowerThreshold > 0 {
		lowerLimit = c.LowerThreshold
	}
	if c.Interval > 0 {
		checkInterval = c.Interval
	}
	log.Printf("Monitor config: upper=%.1f, lower=%.1f, interval=%v", upperLimit, lowerLimit, checkInterval)
}

func GetConfig() Config {
	mu.RLock()
	defer mu.RUnlock()
	return Config{
		UpperThreshold: upperLimit,
		LowerThreshold: lowerLimit,
		Interval:       checkInterval,
	}
}

func GetCPUUsage() float64 {
	mu.RLock()
	defer mu.RUnlock()
	return lastUsage
}

func GetUsage() float64 {
	mu.Lock()
	defer mu.Unlock()
	usage, err := cpu.Percent(0, false)
	if err != nil {
		log.Printf("Error getting CPU: %v", err)
		return lastUsage
	}
	if len(usage) > 0 {
		lastUsage = usage[0]
	}
	return lastUsage
}

func IsOverloaded() bool {
	mu.RLock()
	defer mu.RUnlock()
	return lastUsage > upperLimit
}

func IsUnderutilized() bool {
	mu.RLock()
	defer mu.RUnlock()
	return lastUsage < lowerLimit
}

func StartMonitoring(onOverload, onUnderuse func()) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()
		
		highCount := 0
		lowCount := 0
		requiredCount := 3 // needing 3 consecutive readings before acting
		
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				usage := GetUsage()
				
				if usage > upperLimit {
					highCount++
					lowCount = 0
					if highCount >= requiredCount {
						log.Printf("CPU overload sustained: %.1f%% (x%d)", usage, highCount)
						onOverload()
						highCount = 0
					}
				} else if usage < lowerLimit {
					lowCount++
					highCount = 0
					if lowCount >= requiredCount {
						log.Printf("CPU underutilized sustained: %.1f%% (x%d)", usage, lowCount)
						onUnderuse()
						lowCount = 0
					}
				} else {
					highCount = 0
					lowCount = 0
				}
			}
		}
	}()
	return func() { close(stop) }
}