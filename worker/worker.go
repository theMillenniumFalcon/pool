package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/themillenniumfalcon/pool/job"
	"github.com/themillenniumfalcon/pool/result"
	"golang.org/x/time/rate"
)

type Worker struct {
	ID          int
	JobChannel  chan job.Job
	ResultPool  chan<- result.Result
	Quit        chan bool
	WorkerPool  chan<- chan job.Job
	Health      *WorkerHealth
	RateLimiter *rate.Limiter
	healthMutex sync.RWMutex
	metrics     []time.Duration
}

func NewWorker(id int, resultPool chan<- result.Result, workerPool chan<- chan job.Job) Worker {
	return Worker{
		ID:          id,
		JobChannel:  make(chan job.Job),
		ResultPool:  resultPool,
		Quit:        make(chan bool),
		WorkerPool:  workerPool,
		Health:      &WorkerHealth{ID: id, LastHeartbeat: time.Now()},
		RateLimiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 1),
		metrics:     make([]time.Duration, 0, 100),
	}
}

func (w *Worker) executeJob(job job.Job) result.Result {
	startTime := time.Now()
	result := result.Result{
		JobID:    job.ID,
		WorkerID: w.ID,
	}

	if rand.Float64() < 0.05 { // 5% chance of failure
		result.Error = errors.New("random processing error")
	} else {
		time.Sleep(time.Duration(100+job.Priority*50) * time.Millisecond)
		result.Output = fmt.Sprintf("Processed job %d with priority %d", job.ID, job.Priority)
	}

	result.Duration = time.Since(startTime)
	return result
}

func (w *Worker) processJob(job job.Job) result.Result {
	var res result.Result
	startTime := time.Now()

	if job.Ctx == nil {
		job.Ctx = context.Background()
	}

	// Wait for rate limiter
	if err := w.RateLimiter.Wait(job.Ctx); err != nil {
		return result.Result{
			JobID:    job.ID,
			WorkerID: w.ID,
			Error:    fmt.Errorf("rate limit error: %v", err),
			Duration: time.Since(startTime),
		}
	}

	// Set default MaxRetry if not specified
	if job.MaxRetry == 0 {
		job.MaxRetry = 3
	}

	// Process with retry logic
	for attempt := 0; attempt <= job.MaxRetry; attempt++ {
		select {
		case <-job.Ctx.Done():
			return result.Result{
				JobID:    job.ID,
				WorkerID: w.ID,
				Error:    job.Ctx.Err(),
				Duration: time.Since(startTime),
				Attempt:  attempt,
			}
		default:
			res = w.executeJob(job)
			if res.Error == nil {
				res.Attempt = attempt
				return res
			}

			if attempt < job.MaxRetry {
				log.Printf("Job %d failed attempt %d/%d: %v. Retrying...", job.ID, attempt+1, job.MaxRetry, res.Error)
				time.Sleep(time.Duration(attempt*attempt) * 100 * time.Millisecond)
				continue
			}

			// Max retries reached
			res.Attempt = attempt
			return res
		}
	}

	return res
}

func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				result := w.processJob(job)

				w.updateHealth(result.Duration, result.Error)

				w.ResultPool <- result
			case <-w.Quit:
				return
			}
		}
	}()
}
