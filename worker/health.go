package worker

import (
	"context"
	"log"
	"sync"
	"time"
)

type WorkerHealth struct {
	ID             int
	JobsProcessed  int
	LastHeartbeat  time.Time
	AverageJobTime time.Duration
	FailureRate    float64
	Status         string
}

type HealthMonitor struct {
	workers       []*WorkerHealth
	mutex         sync.RWMutex
	healthyCutoff time.Duration
	checkInterval time.Duration
}

func NewHealthMonitor(checkInterval time.Duration, numWorkers int) *HealthMonitor {
	monitor := &HealthMonitor{
		workers:       make([]*WorkerHealth, numWorkers),
		healthyCutoff: 5 * time.Second,
		checkInterval: checkInterval,
	}

	for i := 0; i < numWorkers; i++ {
		monitor.workers[i] = &WorkerHealth{
			ID:            i,
			LastHeartbeat: time.Now(),
			Status:        "healthy",
		}
	}

	return monitor
}

func (hm *HealthMonitor) RegisterWorker(workerHealth *WorkerHealth) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if workerHealth.ID < len(hm.workers) {
		hm.workers[workerHealth.ID] = workerHealth
	}
}

func (hm *HealthMonitor) checkWorkersHealth() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	now := time.Now()
	for _, health := range hm.workers {
		if health == nil {
			continue
		}

		timeSinceHeartbeat := now.Sub(health.LastHeartbeat)

		if timeSinceHeartbeat > hm.healthyCutoff {
			health.Status = "dead"
		} else if health.FailureRate > 0.5 {
			health.Status = "failing"
		} else {
			health.Status = "healthy"
		}

		log.Printf("Worker %d health status: %s (Processed: %d, Avg Time: %v, Failure Rate: %.2f)",
			health.ID, health.Status, health.JobsProcessed, health.AverageJobTime, health.FailureRate)
	}
}

func (w *Worker) updateHealth(duration time.Duration, err error) {
	w.healthMutex.Lock()
	defer w.healthMutex.Unlock()

	w.Health.LastHeartbeat = time.Now()
	w.Health.JobsProcessed++

	w.metrics = append(w.metrics, duration)
	if len(w.metrics) > 100 {
		w.metrics = w.metrics[1:]
	}

	var total time.Duration
	for _, d := range w.metrics {
		total += d
	}
	w.Health.AverageJobTime = total / time.Duration(len(w.metrics))

	if err != nil {
		w.Health.FailureRate = (w.Health.FailureRate*float64(w.Health.JobsProcessed-1) + 1) / float64(w.Health.JobsProcessed)
	} else {
		w.Health.FailureRate = w.Health.FailureRate * float64(w.Health.JobsProcessed-1) / float64(w.Health.JobsProcessed)
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
