package dispatcher

import (
	"github.com/themillenniumfalcon/pool/job"
	"github.com/themillenniumfalcon/pool/result"
	"github.com/themillenniumfalcon/pool/worker"
)

type Dispatcher struct {
	WorkerPool chan chan job.Job
	MaxWorkers int
	JobQueue   chan job.Job
	Workers    []worker.Worker
}

func NewDispatcher(maxWorkers int, jobQueue chan job.Job, resultPool chan result.Result) *Dispatcher {
	workerPool := make(chan chan job.Job, maxWorkers)
	workers := make([]worker.Worker, maxWorkers)

	for i := 0; i < maxWorkers; i++ {
		workers[i] = worker.NewWorker(i, resultPool, workerPool)
	}

	return &Dispatcher{
		WorkerPool: workerPool,
		MaxWorkers: maxWorkers,
		JobQueue:   jobQueue,
		Workers:    workers,
	}
}

func (d *Dispatcher) Start() {
	for i := 0; i < d.MaxWorkers; i++ {
		d.Workers[i].Start()
	}

	go func() {
		for job := range d.JobQueue {
			workerJobQueue := <-d.WorkerPool
			workerJobQueue <- job
		}
	}()
}
