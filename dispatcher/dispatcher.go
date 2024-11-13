package dispatcher

import (
	"context"
	"time"

	"github.com/themillenniumfalcon/pool/job"
	"github.com/themillenniumfalcon/pool/result"
	"github.com/themillenniumfalcon/pool/worker"
	"golang.org/x/time/rate"
)

type Dispatcher struct {
	WorkerPool    chan chan job.Job
	MaxWorkers    int
	JobQueue      chan job.Job
	Workers       []worker.Worker
	HealthMonitor *worker.HealthMonitor
	RateLimiter   *rate.Limiter
}

func NewDispatcher(maxWorkers int, jobQueue chan job.Job, resultPool chan result.Result) *Dispatcher {
	workerPool := make(chan chan job.Job, maxWorkers)
	workers := make([]worker.Worker, maxWorkers)

	healthMonitor := worker.NewHealthMonitor(1*time.Second, maxWorkers)

	for i := 0; i < maxWorkers; i++ {
		workers[i] = worker.NewWorker(i, resultPool, workerPool)
		healthMonitor.RegisterWorker(workers[i].Health)
	}

	return &Dispatcher{
		WorkerPool:    workerPool,
		MaxWorkers:    maxWorkers,
		JobQueue:      jobQueue,
		Workers:       workers,
		HealthMonitor: healthMonitor,
		RateLimiter:   rate.NewLimiter(rate.Every(50*time.Millisecond), 5), // Max 20 jobs per second
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	d.HealthMonitor.Start(ctx)

	for i := 0; i < d.MaxWorkers; i++ {
		d.Workers[i].Start()
	}

	go func() {
		for job := range d.JobQueue {
			if err := d.RateLimiter.Wait(ctx); err != nil {
				continue
			}

			select {
			case <-ctx.Done():
				return
			default:
				workerJobQueue := <-d.WorkerPool
				workerJobQueue <- job
			}
		}
	}()
}
