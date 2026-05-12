package pool

import (
	"context"
	"document-convert-service-new/internal/model"
	"document-convert-service-new/internal/usecase"
	"log/slog"
	"sync"
	"time"
)

const jobTimeout = 5 * time.Minute

func RunWorkerPool(dispatch <-chan model.Job, usecase *usecase.UseCase, workerCount int) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func(id int) {
			defer wg.Done()
			for job := range dispatch {
				ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
				defer cancel()
				if err := usecase.Process(ctx, job.Data); err != nil {
					slog.Error("job failed", "worker", id, "error", err)
				} else {
					job.Ack()
				}
			}
		}(i)
	}

	return wg
}
