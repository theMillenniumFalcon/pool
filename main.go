package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/themillenniumfalcon/pool/dispatcher"
	"github.com/themillenniumfalcon/pool/job"
	"github.com/themillenniumfalcon/pool/result"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobQueue := make(chan job.Job, 100)
	resultPool := make(chan result.Result, 100)

	dispatcher := dispatcher.NewDispatcher(5, jobQueue, resultPool)
	dispatcher.Start(ctx)

	var wg sync.WaitGroup

	go func() {
		for result := range resultPool {
			if result.Error != nil {
				log.Printf("Job %d failed on attempt %d: %v\n",
					result.JobID, result.Attempt, result.Error)
			} else {
				log.Printf("Job %d completed successfully by Worker %d in %v (attempt %d): %v\n",
					result.JobID, result.WorkerID, result.Duration, result.Attempt, result.Output)
			}
			wg.Done()
		}
	}()

	numJobs := 20
	wg.Add(numJobs)

	for i := 0; i < numJobs; i++ {
		jobCtx, jobCancel := context.WithTimeout(ctx, 10*time.Second)
		defer jobCancel()

		job := job.Job{
			ID:       i,
			Priority: i % 3,
			Data:     fmt.Sprintf("Job data %d", i),
			Ctx:      jobCtx,
			MaxRetry: 3,
		}
		jobQueue <- job
	}

	wg.Wait()
}
