package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/themillenniumfalcon/pool/dispatcher"
	"github.com/themillenniumfalcon/pool/job"
	"github.com/themillenniumfalcon/pool/result"
)

func main() {
	jobQueue := make(chan job.Job, 100)
	resultPool := make(chan result.Result, 100)

	dispatcher := dispatcher.NewDispatcher(5, jobQueue, resultPool)
	dispatcher.Start()

	var wg sync.WaitGroup

	go func() {
		for result := range resultPool {
			log.Printf("Job %d completed by Worker %d in %v: %v\n",
				result.JobID, result.WorkerID, result.Duration, result.Output)
			wg.Done()
		}
	}()

	numJobs := 20
	wg.Add(numJobs)

	for i := 0; i < numJobs; i++ {
		job := job.Job{
			ID:       i,
			Priority: i % 3, // Priority levels 0-2
			Data:     fmt.Sprintf("Job data %d", i),
		}
		jobQueue <- job
	}

	wg.Wait()
}
