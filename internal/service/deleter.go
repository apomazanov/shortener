package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// AsyncDeleterConfig contains configuration data for async deleter.
type AsyncDeleterConfig struct {
	// BatchSize defines size of a batch for delete operation.
	BatchSize int
	// Timeout is a timeout for delete operation.
	Timeout time.Duration
}

// deleteTask contains data for single delete operation (batch of aliases).
type deleteTask struct {
	// userID is a user whose data is going to be deleted.
	userID string
	// aliases is a slice of aliases to be deleted.
	aliases []string
}

// asyncDeleter contains data for async deletion operation.
type asyncDeleter struct {
	// repo contains repository methods.
	repo Repo
	// batchSize defines size of a batch for delete operation.
	batchSize int
	// timeout is a timeout for delete operation.
	timeout time.Duration
	// log is a pointer to Logger object.
	log *zerolog.Logger
	// chTasks is a channel for delete operations queue.
	chTasks chan deleteTask
}

// newAsyncDeleter creates and returns a new async deleter object.
func newAsyncDeleter(repo Repo, cfg *AsyncDeleterConfig, log *zerolog.Logger) *asyncDeleter {
	d := &asyncDeleter{
		repo:      repo,
		batchSize: cfg.BatchSize,
		timeout:   cfg.Timeout,
		log:       log,
		chTasks:   make(chan deleteTask, 1000),
	}

	go d.collect()

	return d
}

// Enqueue adds delete operations to queue.
func (d *asyncDeleter) Enqueue(userID string, aliases []string) {
	d.chTasks <- deleteTask{
		userID:  userID,
		aliases: aliases,
	}
}

// collect provides getting delete requests from queue, packing them into
// batches and executing.
func (d *asyncDeleter) collect() {
	batch := make(map[string][]string)
	aliasesInBatch := 0

	timer := time.NewTimer(d.timeout)
	defer timer.Stop()

	flush := func() {
		if aliasesInBatch == 0 {
			return
		}

		d.flushWithRetries(batch)

		batch = make(map[string][]string)
		aliasesInBatch = 0

		timer.Reset(d.timeout)
	}

	for {
		select {
		case <-timer.C:
			flush()

		case task, open := <-d.chTasks:
			if !open {
				flush()
				return
			}

			batch[task.userID] = append(batch[task.userID], task.aliases...)
			aliasesInBatch += len(task.aliases)

			if aliasesInBatch >= d.batchSize {
				flush()
			}
		}
	}
}

// flushWithRetries provides data operations executing with retries mechanism.
func (d *asyncDeleter) flushWithRetries(data map[string][]string) {
	const maxRetries = 5

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var err error

	for range maxRetries {
		err = d.repo.DeleteBatch(ctx, data)

		if err == nil {
			return
		}

		time.Sleep(time.Millisecond * 200)
	}

	d.log.Error().
		Str("op", "deleter.flushWithRetries").
		Err(err).
		Msg("service: batch of aliases del retry limit exceeded")
}
