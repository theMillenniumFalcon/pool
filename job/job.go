package job

import "context"

type Job struct {
	ID       int
	Data     interface{}
	Priority int
	Ctx      context.Context
	Retries  int
	MaxRetry int
}
