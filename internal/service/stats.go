package service

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// statsUpdater contains data used for statistics updating routine.
type statsUpdater struct {
	// repo is a repository, where statistics is stored.
	repo Repo
	// interval is a statistics update period.
	interval time.Duration
	// log is a pointer to logger.
	log *zerolog.Logger
	// chOff is a channel for shutdown procedure initiation.
	chOff chan struct{}
	// stopOnce used for safe repeated Stop executing
	stopOnce sync.Once
}

// NewStatsUpdater constructs a new statsUpdater object.
func NewStatsUpdater(repo Repo,
	interval time.Duration,
	log *zerolog.Logger) *statsUpdater {
	return &statsUpdater{
		repo:     repo,
		interval: interval,
		log:      log,
		chOff:    make(chan struct{}),
	}
}

// Run provides statsUpdater operation: statistics is being updated according to interval value.
func (u *statsUpdater) Run(ctx context.Context) {
	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-u.chOff:
			return

		case <-ctx.Done():
			u.log.Error().
				Str("op", "statsUpdater.Run").
				Msg("service: stats updater stopped by context")

			return

		case <-ticker.C:
			err := u.repo.RefreshStats(ctx)
			if err != nil {
				u.log.Error().
					Str("op", "statsUpdater.Run").
					Err(err).
					Msg("service: failed to refresh stats in db")
			}
		}
	}
}

// Stop provides proper operation shutdown.
func (u *statsUpdater) Stop() {
	u.stopOnce.Do(func() {
		close(u.chOff)
	})
}
