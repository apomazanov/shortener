package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/apomazanov/shortener/internal/service/mocks"
)

type deleterTestContext struct {
	repo    *mocks.MockRepo
	deleter *asyncDeleter
}

func createDeleterTestContext(t *testing.T, cfg *AsyncDeleterConfig) *deleterTestContext {
	t.Helper()

	ctrl := gomock.NewController(t)

	repo := mocks.NewMockRepo(ctrl)
	logger := zerolog.Nop()

	deleter := newAsyncDeleter(repo, cfg, &logger)

	t.Cleanup(func() {
		close(deleter.chTasks)
	})

	return &deleterTestContext{
		repo:    repo,
		deleter: deleter,
	}
}

func TestAsyncDeleter_FlushOnBatchSize(t *testing.T) {
	cfg := &AsyncDeleterConfig{
		BatchSize: 3,
		Timeout:   time.Hour,
	}

	tcx := createDeleterTestContext(t, cfg)

	called := make(chan map[string][]string, 1)

	tcx.repo.EXPECT().
		DeleteBatch(gomock.Any(), map[string][]string{
			"user-1": {"alias-1", "alias-2"},
			"user-2": {"alias-3"},
		}).
		DoAndReturn(func(ctx context.Context, batch map[string][]string) error {
			called <- batch
			return nil
		}).
		Times(1)

	tcx.deleter.Enqueue("user-1", []string{"alias-1", "alias-2"})
	tcx.deleter.Enqueue("user-2", []string{"alias-3"})

	select {
	case batch := <-called:
		assert.ElementsMatch(t, []string{"alias-1", "alias-2"}, batch["user-1"])
		assert.ElementsMatch(t, []string{"alias-3"}, batch["user-2"])

	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected DeleteBatch to be called after batch size limit reached")
	}
}

func TestAsyncDeleter_FlushOnTimeout(t *testing.T) {
	cfg := &AsyncDeleterConfig{
		BatchSize: 100,
		Timeout:   10 * time.Millisecond,
	}

	tcx := createDeleterTestContext(t, cfg)

	called := make(chan map[string][]string, 1)

	tcx.repo.EXPECT().
		DeleteBatch(gomock.Any(), map[string][]string{
			"user-1": {"alias-1"},
		}).
		DoAndReturn(func(ctx context.Context, batch map[string][]string) error {
			called <- batch
			return nil
		}).
		Times(1)

	tcx.deleter.Enqueue("user-1", []string{"alias-1"})

	select {
	case batch := <-called:
		assert.ElementsMatch(t, []string{"alias-1"}, batch["user-1"])

	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected DeleteBatch to be called after timeout")
	}
}

func TestAsyncDeleter_GroupsAliasesByUser(t *testing.T) {
	cfg := &AsyncDeleterConfig{
		BatchSize: 4,
		Timeout:   time.Hour,
	}

	tcx := createDeleterTestContext(t, cfg)

	called := make(chan map[string][]string, 1)

	tcx.repo.EXPECT().
		DeleteBatch(gomock.Any(), map[string][]string{
			"user-1": {"alias-1", "alias-2", "alias-4"},
			"user-2": {"alias-3"},
		}).
		DoAndReturn(func(ctx context.Context, batch map[string][]string) error {
			called <- batch
			return nil
		}).
		Times(1)

	tcx.deleter.Enqueue("user-1", []string{"alias-1", "alias-2"})
	tcx.deleter.Enqueue("user-2", []string{"alias-3"})
	tcx.deleter.Enqueue("user-1", []string{"alias-4"})

	select {
	case batch := <-called:
		assert.ElementsMatch(t, []string{"alias-1", "alias-2", "alias-4"}, batch["user-1"])
		assert.ElementsMatch(t, []string{"alias-3"}, batch["user-2"])

	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected DeleteBatch to be called after batch size limit reached")
	}
}

func TestAsyncDeleter_FlushWithRetries_SuccessFirstTry(t *testing.T) {
	cfg := &AsyncDeleterConfig{
		BatchSize: 10,
		Timeout:   time.Hour,
	}

	tcx := createDeleterTestContext(t, cfg)

	data := map[string][]string{
		"user-1": {"alias-1"},
	}

	tcx.repo.EXPECT().
		DeleteBatch(gomock.Any(), data).
		Return(nil).
		Times(1)

	tcx.deleter.flushWithRetries(data)
}

func TestAsyncDeleter_FlushWithRetries_RetryThenSuccess(t *testing.T) {
	cfg := &AsyncDeleterConfig{
		BatchSize: 10,
		Timeout:   time.Hour,
	}

	tcx := createDeleterTestContext(t, cfg)

	data := map[string][]string{
		"user-1": {"alias-1"},
	}

	repoErr := errors.New("temporary repo error")

	gomock.InOrder(
		tcx.repo.EXPECT().
			DeleteBatch(gomock.Any(), data).
			Return(repoErr).
			Times(1),

		tcx.repo.EXPECT().
			DeleteBatch(gomock.Any(), data).
			Return(nil).
			Times(1),
	)

	tcx.deleter.flushWithRetries(data)
}

func TestAsyncDeleter_FlushWithRetries_RetryLimitExceeded(t *testing.T) {
	cfg := &AsyncDeleterConfig{
		BatchSize: 10,
		Timeout:   time.Hour,
	}

	tcx := createDeleterTestContext(t, cfg)

	data := map[string][]string{
		"user-1": {"alias-1"},
	}

	repoErr := errors.New("persistent repo error")

	tcx.repo.EXPECT().
		DeleteBatch(gomock.Any(), data).
		Return(repoErr).
		Times(5)

	tcx.deleter.flushWithRetries(data)
}
