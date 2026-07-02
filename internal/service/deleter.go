package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type AsyncDeleterConfig struct {
	BatchSize int
	Timeout   time.Duration
}

type deleteTask struct {
	userID  string
	aliases []string
}

type asyncDeleter struct {
	repo      Repo
	batchSize int
	timeout   time.Duration
	log       *zerolog.Logger
	chTasks   chan deleteTask
}

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

func (d *asyncDeleter) Enqueue(userID string, aliases []string) {
	d.chTasks <- deleteTask{
		userID:  userID,
		aliases: aliases,
	}
}

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
