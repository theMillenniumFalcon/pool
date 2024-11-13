package result

import "time"

type Result struct {
	JobID    int
	Output   interface{}
	WorkerID int
	Error    error
	Duration time.Duration
	Attempt  int
}
