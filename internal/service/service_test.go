package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/apomazanov/shortener/internal/service/mocks"
)

type serviceTestContext struct {
	ctx     context.Context
	repo    *mocks.MockRepo
	service *Service
}

func createServiceTestContext(t *testing.T) *serviceTestContext {
	t.Helper()

	ctrl := gomock.NewController(t)

	repo := mocks.NewMockRepo(ctrl)
	service := New(repo)

	return &serviceTestContext{
		ctx:     context.Background(),
		repo:    repo,
		service: service,
	}
}

func assertServiceErr(t *testing.T, err error, expectedErr error) {
	t.Helper()

	if expectedErr != nil {
		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		return
	}

	require.NoError(t, err)
}

func assertGeneratedAlias(t *testing.T, alias string) {
	t.Helper()

	assert.Len(t, alias, 6)
}

func assertBatchAliases(t *testing.T, toWrite map[string]string, originals []string) {
	t.Helper()

	require.Len(t, toWrite, len(originals))

	for _, original := range originals {
		alias, ok := toWrite[original]
		require.True(t, ok)
		assertGeneratedAlias(t, alias)
	}
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))

	for k, v := range src {
		dst[k] = v
	}

	return dst
}

func TestService_GetOriginalURL(t *testing.T) {
	const (
		alias       = "abcdef"
		originalURL = "http://example.com/original"
	)

	internalErr := errors.New("repo get error")

	tests := []struct {
		name             string
		mocksSetup       func(repo *mocks.MockRepo)
		expectedOriginal string
		expectedErr      error
	}{
		{
			name: "success",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), alias).
					Return(originalURL, nil).
					Times(1)
			},
			expectedOriginal: originalURL,
			expectedErr:      nil,
		},
		{
			name: "not found",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), alias).
					Return("", domain.ErrNotFound).
					Times(1)
			},
			expectedOriginal: "",
			expectedErr:      domain.ErrNotFound,
		},
		{
			name: "internal repo error",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					Get(gomock.Any(), alias).
					Return("", internalErr).
					Times(1)
			},
			expectedOriginal: "",
			expectedErr:      internalErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createServiceTestContext(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(tcx.repo)
			}

			original, err := tcx.service.GetOriginalURL(tcx.ctx, alias)

			assertServiceErr(t, err, tt.expectedErr)
			assert.Equal(t, tt.expectedOriginal, original)
		})
	}
}

func TestService_GetUserURLs(t *testing.T) {
	const userID = "123e4567-e89b-42d3-a456-426614174000"

	internalErr := errors.New("repo get by user error")

	userURLs := map[string]string{
		"http://shortener/alias1": "http://example.com/original1",
		"http://shortener/alias2": "http://example.com/original2",
	}

	tests := []struct {
		name         string
		mocksSetup   func(repo *mocks.MockRepo)
		expectedData map[string]string
		expectedErr  error
	}{
		{
			name: "success",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					GetByUser(gomock.Any(), userID).
					Return(userURLs, nil).
					Times(1)
			},
			expectedData: userURLs,
			expectedErr:  nil,
		},
		{
			name: "not found",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					GetByUser(gomock.Any(), userID).
					Return(nil, domain.ErrNotFound).
					Times(1)
			},
			expectedData: nil,
			expectedErr:  domain.ErrNotFound,
		},
		{
			name: "internal repo error",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					GetByUser(gomock.Any(), userID).
					Return(nil, internalErr).
					Times(1)
			},
			expectedData: nil,
			expectedErr:  internalErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createServiceTestContext(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(tcx.repo)
			}

			data, err := tcx.service.GetUserURLs(tcx.ctx, userID)

			assertServiceErr(t, err, tt.expectedErr)
			assert.Equal(t, tt.expectedData, data)
		})
	}
}

func TestService_CreateURLAlias(t *testing.T) {
	const (
		originalURL = "http://example.com/original"
		userID      = "123e4567-e89b-42d3-a456-426614174000"
	)

	internalErr := errors.New("repo save error")

	tests := []struct {
		name          string
		mocksSetup    func(repo *mocks.MockRepo)
		expectedAlias string
		checkAliasLen bool
		expectedErr   error
	}{
		{
			name: "success first try",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any(), originalURL, userID).
					DoAndReturn(func(ctx context.Context, alias string, original string, userID string) (string, error) {
						assertGeneratedAlias(t, alias)
						return alias, nil
					}).
					Times(1)
			},
			expectedAlias: "",
			checkAliasLen: true,
			expectedErr:   nil,
		},
		{
			name: "original url duplicate",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any(), originalURL, userID).
					Return("existing", domain.ErrOriginalURLDuplicate).
					Times(1)
			},
			expectedAlias: "existing",
			checkAliasLen: false,
			expectedErr:   domain.ErrOriginalURLDuplicate,
		},
		{
			name: "alias duplicate then success",
			mocksSetup: func(repo *mocks.MockRepo) {
				gomock.InOrder(
					repo.EXPECT().
						Save(gomock.Any(), gomock.Any(), originalURL, userID).
						Return("", domain.ErrAliasDuplicate).
						Times(1),

					repo.EXPECT().
						Save(gomock.Any(), gomock.Any(), originalURL, userID).
						DoAndReturn(func(ctx context.Context, alias string, original string, userID string) (string, error) {
							assertGeneratedAlias(t, alias)
							return alias, nil
						}).
						Times(1),
				)
			},
			expectedAlias: "",
			checkAliasLen: true,
			expectedErr:   nil,
		},
		{
			name: "internal repo error",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any(), originalURL, userID).
					Return("", internalErr).
					Times(1)
			},
			expectedAlias: "",
			checkAliasLen: false,
			expectedErr:   internalErr,
		},
		{
			name: "retry limit exceeded",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					Save(gomock.Any(), gomock.Any(), originalURL, userID).
					Return("", domain.ErrAliasDuplicate).
					Times(5)
			},
			expectedAlias: "",
			checkAliasLen: false,
			expectedErr:   domain.ErrSaveRetryLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createServiceTestContext(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(tcx.repo)
			}

			alias, err := tcx.service.CreateURLAlias(tcx.ctx, originalURL, userID)

			assertServiceErr(t, err, tt.expectedErr)

			if tt.checkAliasLen {
				assertGeneratedAlias(t, alias)
			} else {
				assert.Equal(t, tt.expectedAlias, alias)
			}
		})
	}
}

func TestService_CreateURLAliasBatch(t *testing.T) {
	const userID = "123e4567-e89b-42d3-a456-426614174000"

	originals := []string{
		"http://example.com/original1",
		"http://example.com/original2",
	}

	writtenData := map[string]string{
		"http://example.com/original1": "alias1",
		"http://example.com/original2": "alias2",
	}

	internalErr := errors.New("repo save batch error")

	tests := []struct {
		name             string
		mocksSetup       func(repo *mocks.MockRepo)
		expectedWritten  map[string]string
		expectedErr      error
		checkGeneratedIn bool
	}{
		{
			name: "success",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					SaveBatch(gomock.Any(), gomock.Any(), userID).
					DoAndReturn(func(ctx context.Context, toWrite map[string]string, userID string) (map[string]string, error) {
						assertBatchAliases(t, toWrite, originals)
						return cloneStringMap(toWrite), nil
					}).
					Times(1)
			},
			expectedWritten:  nil,
			expectedErr:      nil,
			checkGeneratedIn: true,
		},
		{
			name: "original url duplicate",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					SaveBatch(gomock.Any(), gomock.Any(), userID).
					DoAndReturn(func(ctx context.Context, toWrite map[string]string, userID string) (map[string]string, error) {
						assertBatchAliases(t, toWrite, originals)
						return writtenData, domain.ErrOriginalURLDuplicate
					}).
					Times(1)
			},
			expectedWritten:  writtenData,
			expectedErr:      domain.ErrOriginalURLDuplicate,
			checkGeneratedIn: false,
		},
		{
			name: "alias duplicate then success",
			mocksSetup: func(repo *mocks.MockRepo) {
				gomock.InOrder(
					repo.EXPECT().
						SaveBatch(gomock.Any(), gomock.Any(), userID).
						DoAndReturn(func(ctx context.Context, toWrite map[string]string, userID string) (map[string]string, error) {
							assertBatchAliases(t, toWrite, originals)
							return nil, domain.ErrAliasDuplicate
						}).
						Times(1),

					repo.EXPECT().
						SaveBatch(gomock.Any(), gomock.Any(), userID).
						DoAndReturn(func(ctx context.Context, toWrite map[string]string, userID string) (map[string]string, error) {
							assertBatchAliases(t, toWrite, originals)
							return cloneStringMap(toWrite), nil
						}).
						Times(1),
				)
			},
			expectedWritten:  nil,
			expectedErr:      nil,
			checkGeneratedIn: true,
		},
		{
			name: "internal repo error",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					SaveBatch(gomock.Any(), gomock.Any(), userID).
					Return(nil, internalErr).
					Times(1)
			},
			expectedWritten:  nil,
			expectedErr:      internalErr,
			checkGeneratedIn: false,
		},
		{
			name: "retry limit exceeded",
			mocksSetup: func(repo *mocks.MockRepo) {
				repo.EXPECT().
					SaveBatch(gomock.Any(), gomock.Any(), userID).
					Return(nil, domain.ErrAliasDuplicate).
					Times(5)
			},
			expectedWritten:  nil,
			expectedErr:      domain.ErrSaveRetryLimitExceeded,
			checkGeneratedIn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createServiceTestContext(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(tcx.repo)
			}

			written, err := tcx.service.CreateURLAliasBatch(tcx.ctx, originals, userID)

			assertServiceErr(t, err, tt.expectedErr)

			if tt.checkGeneratedIn {
				assertBatchAliases(t, written, originals)
			} else {
				assert.Equal(t, tt.expectedWritten, written)
			}
		})
	}
}
