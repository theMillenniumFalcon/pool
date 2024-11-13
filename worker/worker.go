package worker

import (
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

func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				startTime := time.Now()
				result := result.Result{
					JobID:    job.ID,
					WorkerID: w.ID,
				}

				time.Sleep(time.Duration(100+job.Priority*50) * time.Millisecond)

				result.Output = fmt.Sprintf("Processed job %d with priority %d", job.ID, job.Priority)
				result.Duration = time.Since(startTime)

				w.ResultPool <- result

			case <-w.Quit:
				return
			}
		}
	}()
}
