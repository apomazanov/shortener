package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/apomazanov/shortener/internal/service/mocks"
	"github.com/rs/zerolog"
	"go.uber.org/mock/gomock"
)

func TestStatsUpdater(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name     string
		interval time.Duration
		setup    func(m *mocks.MockRepo)
		action   func(cancel context.CancelFunc, updater *statsUpdater)
	}{
		{
			name:     "successful ticker cycles",
			interval: 10 * time.Millisecond,
			setup: func(m *mocks.MockRepo) {
				m.EXPECT().
					RefreshStats(gomock.Any()).
					Return(nil).
					MinTimes(2)
			},
			action: func(cancel context.CancelFunc, updater *statsUpdater) {
				time.Sleep(30 * time.Millisecond)
				cancel()
			},
		},
		{
			name:     "repository error is handled without crashing",
			interval: 10 * time.Millisecond,
			setup: func(m *mocks.MockRepo) {
				m.EXPECT().
					RefreshStats(gomock.Any()).
					Return(errors.New("db connection failure")).
					MinTimes(1)
			},
			action: func(cancel context.CancelFunc, updater *statsUpdater) {
				time.Sleep(25 * time.Millisecond)
				cancel()
			},
		},
		{
			name:     "graceful stop via Stop() method",
			interval: 10 * time.Millisecond,
			setup: func(m *mocks.MockRepo) {
				m.EXPECT().
					RefreshStats(gomock.Any()).
					Return(nil).
					MinTimes(1)
			},
			action: func(cancel context.CancelFunc, updater *statsUpdater) {
				time.Sleep(15 * time.Millisecond)
				updater.Stop()
			},
		},
		{
			name:     "immediate exit on pre-canceled context",
			interval: 100 * time.Millisecond,
			setup: func(m *mocks.MockRepo) {
				m.EXPECT().
					RefreshStats(gomock.Any()).
					Times(0)
			},
			action: func(cancel context.CancelFunc, updater *statsUpdater) {
				cancel()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepo(ctrl)
			if tt.setup != nil {
				tt.setup(mockRepo)
			}

			updater := NewStatsUpdater(mockRepo, tt.interval, &logger)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan struct{})
			go func() {
				updater.Run(ctx)
				close(done)
			}()

			tt.action(cancel, updater)

			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
				t.Fatal("statsUpdater.Run did not stop within expected time")
			}
		})
	}
}
