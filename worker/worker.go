package worker

import (
	"errors"
	"fmt"
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

	time.Sleep(time.Duration(100+job.Priority*50) * time.Millisecond)

	if time.Now().UnixNano()%10 == 0 {
		result.Error = errors.New("random processing error")
	} else {
		result.Output = fmt.Sprintf("Processed job %d with priority %d", job.ID, job.Priority)
	}

	result.Duration = time.Since(startTime)
	return result
}

func (w *Worker) processJob(job job.Job) result.Result {
	var res result.Result
	var err error
	startTime := time.Now()

	err = w.RateLimiter.Wait(job.Ctx)
	if err != nil {
		return result.Result{
			JobID:    job.ID,
			WorkerID: w.ID,
			Error:    fmt.Errorf("rate limit error: %v", err),
			Duration: time.Since(startTime),
		}
	}

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
			if res.Error == nil || attempt == job.MaxRetry {
				res.Attempt = attempt
				return res
			}
			time.Sleep(time.Duration(attempt*attempt) * 100 * time.Millisecond)
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
