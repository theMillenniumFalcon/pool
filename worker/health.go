package worker

import (
	"context"
	"log"
	"sync"
	"time"
)

type WorkerHealth struct {
	JobsProcessed  int
	LastHeartbeat  time.Time
	AverageJobTime time.Duration
	FailureRate    float64
	Status         string
}

type HealthMonitor struct {
	workers       map[int]*WorkerHealth
	mutex         sync.RWMutex
	healthyCutoff time.Duration
	checkInterval time.Duration
}

func NewHealthMonitor(checkInterval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		workers:       make(map[int]*WorkerHealth),
		healthyCutoff: 5 * time.Second,
		checkInterval: checkInterval,
	}
}

func (hm *HealthMonitor) checkWorkersHealth() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	now := time.Now()
	for id, health := range hm.workers {
		timeSinceHeartbeat := now.Sub(health.LastHeartbeat)

		if timeSinceHeartbeat > hm.healthyCutoff {
			health.Status = "dead"
		} else if health.FailureRate > 0.5 {
			health.Status = "failing"
		} else {
			health.Status = "healthy"
		}

		log.Printf("Worker %d health status: %s (Processed: %d, Avg Time: %v, Failure Rate: %.2f)",
			id, health.Status, health.JobsProcessed, health.AverageJobTime, health.FailureRate)
	}
}

func (hm *HealthMonitor) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(hm.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hm.checkWorkersHealth()
			}
		}
	}()
}
