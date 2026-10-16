package grpc_handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/apomazanov/shortener/internal/domain/mocks"
	"github.com/apomazanov/shortener/internal/validator"
	pb "github.com/apomazanov/shortener/pkg/api/proto/v1"
)

const (
	validUserID = "123e4567-e89b-42d3-a456-426614174000"
	invalidUser = "not-a-uuid"
	validAlias  = "abcdef"
	validURL    = "http://example.com/original"
	baseURL     = "http://shortener"
)

type grpcTestContext struct {
	business *mocks.MockBusinessService
	cfg      *mocks.MockURLConfig
	jwt      *mocks.MockJWT
	server   *GRPCServer
}

func createGRPCTestContext(t *testing.T) *grpcTestContext {
	t.Helper()

	ctrl := gomock.NewController(t)

	business := mocks.NewMockBusinessService(ctrl)
	cfg := mocks.NewMockURLConfig(ctrl)
	jwt := mocks.NewMockJWT(ctrl)

	logger := zerolog.Nop()
	server := New(business, cfg, &logger, jwt, validator.New())

	return &grpcTestContext{
		business: business,
		cfg:      cfg,
		jwt:      jwt,
		server:   server,
	}
}

// ctxWithAuth builds a context with auth-related values used by handleAuthData.
func ctxWithAuth(userID any, tokenExists any) context.Context {
	ctx := context.Background()

	if userID != nil {
		ctx = context.WithValue(ctx, "user-id", userID)
	}
	if tokenExists != nil {
		ctx = context.WithValue(ctx, "token-exists", tokenExists)
	}

	return ctx
}

func strPtr(s string) *string {
	return &s
}

func assertGRPCCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error")
	assert.Equal(t, expected, st.Code())
}

func TestGRPCServer_ShortenURL(t *testing.T) {
	tests := []struct {
		name          string
		userID        any
		tokenExists   any
		url           string
		mocksSetup    func(tcx *grpcTestContext)
		expectedCode  codes.Code
		expectedShort string
		expectError   bool
	}{
		{
			name:        "success existing user",
			userID:      validUserID,
			tokenExists: true,
			url:         validURL,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return(validAlias, nil).
					Times(1)

				tcx.cfg.EXPECT().
					GetURLBase().
					Return(baseURL).
					Times(1)
			},
			expectedShort: baseURL + "/" + validAlias,
		},
		{
			name:        "invalid url returns InvalidArgument",
			userID:      validUserID,
			tokenExists: true,
			url:         "not-a-url",
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					CreateURLAlias(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectError:  true,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:        "duplicate original returns AlreadyExists",
			userID:      validUserID,
			tokenExists: true,
			url:         validURL,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return(validAlias, domain.ErrOriginalURLDuplicate).
					Times(1)
			},
			expectError:  true,
			expectedCode: codes.AlreadyExists,
		},
		{
			name:        "business error returns Internal",
			userID:      validUserID,
			tokenExists: true,
			url:         validURL,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					CreateURLAlias(gomock.Any(), validURL, validUserID).
					Return("", errors.New("internal error")).
					Times(1)
			},
			expectError:  true,
			expectedCode: codes.Internal,
		},
		{
			name:        "token creation error returns Internal",
			userID:      nil,
			tokenExists: false,
			url:         validURL,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.jwt.EXPECT().
					CreateTokenWithUserID(gomock.Any()).
					Return("", errors.New("token error")).
					Times(1)
			},
			expectError:  true,
			expectedCode: codes.Internal,
		},
		{
			name:        "invalid user-id type returns Internal",
			userID:      123,
			tokenExists: true,
			url:         validURL,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					CreateURLAlias(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectError:  true,
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createGRPCTestContext(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(tcx)
			}

			ctx := ctxWithAuth(tt.userID, tt.tokenExists)
			req := &pb.URLShortenRequest{Url: strPtr(tt.url)}

			resp, err := tcx.server.ShortenURL(ctx, req)

			if tt.expectError {
				assertGRPCCode(t, err, tt.expectedCode)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.expectedShort, resp.GetResult())
		})
	}
}

func TestGRPCServer_ExpandURL(t *testing.T) {
	tests := []struct {
		name           string
		alias          string
		mocksSetup     func(tcx *grpcTestContext)
		expectError    bool
		expectedCode   codes.Code
		expectedResult string
	}{
		{
			name:  "success",
			alias: validAlias,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetOriginalURL(gomock.Any(), validAlias).
					Return(validURL, nil).
					Times(1)
			},
			expectedResult: validURL,
		},
		{
			name:  "invalid alias size returns InvalidArgument",
			alias: "abc",
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetOriginalURL(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectError:  true,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:  "not found returns NotFound",
			alias: validAlias,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetOriginalURL(gomock.Any(), validAlias).
					Return("", domain.ErrNotFound).
					Times(1)
			},
			expectError:  true,
			expectedCode: codes.NotFound,
		},
		{
			name:  "deleted returns NotFound",
			alias: validAlias,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetOriginalURL(gomock.Any(), validAlias).
					Return("", domain.ErrFoundDeleted).
					Times(1)
			},
			expectError:  true,
			expectedCode: codes.NotFound,
		},
		{
			name:  "internal error returns Internal",
			alias: validAlias,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetOriginalURL(gomock.Any(), validAlias).
					Return("", errors.New("db error")).
					Times(1)
			},
			expectError:  true,
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createGRPCTestContext(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(tcx)
			}

			req := &pb.URLExpandRequest{Id: strPtr(tt.alias)}

			resp, err := tcx.server.ExpandURL(context.Background(), req)

			if tt.expectError {
				assertGRPCCode(t, err, tt.expectedCode)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.expectedResult, resp.GetResult())
		})
	}
}

func TestGRPCServer_ListUserURLs(t *testing.T) {
	tests := []struct {
		name          string
		userID        any
		tokenExists   any
		mocksSetup    func(tcx *grpcTestContext)
		expectError   bool
		expectedCode  codes.Code
		expectedPairs map[string]string
		expectEmpty   bool
	}{
		{
			name:        "success returns user urls",
			userID:      validUserID,
			tokenExists: true,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetUserURLs(gomock.Any(), validUserID).
					Return(map[string]string{
						"aaa111": "http://example.com/1",
						"bbb222": "http://example.com/2",
					}, nil).
					Times(1)

				tcx.cfg.EXPECT().
					GetURLBase().
					Return(baseURL).
					AnyTimes()
			},
			expectedPairs: map[string]string{
				baseURL + "/aaa111": "http://example.com/1",
				baseURL + "/bbb222": "http://example.com/2",
			},
		},
		{
			name:        "invalid user-id returns Unauthenticated",
			userID:      invalidUser,
			tokenExists: true,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.jwt.EXPECT().
					CreateTokenWithUserID(gomock.Any()).
					Return("new-token", nil).
					Times(1)

				tcx.business.EXPECT().
					GetUserURLs(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectError:  true,
			expectedCode: codes.Unauthenticated,
		},
		{
			name:        "not found returns empty response",
			userID:      validUserID,
			tokenExists: true,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetUserURLs(gomock.Any(), validUserID).
					Return(nil, domain.ErrNotFound).
					Times(1)
			},
			expectEmpty: true,
		},
		{
			name:        "internal error returns Internal",
			userID:      validUserID,
			tokenExists: true,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetUserURLs(gomock.Any(), validUserID).
					Return(nil, errors.New("db error")).
					Times(1)
			},
			expectError:  true,
			expectedCode: codes.Internal,
		},
		{
			name:        "invalid user-id type returns Internal",
			userID:      123,
			tokenExists: true,
			mocksSetup: func(tcx *grpcTestContext) {
				tcx.business.EXPECT().
					GetUserURLs(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectError:  true,
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcx := createGRPCTestContext(t)

			if tt.mocksSetup != nil {
				tt.mocksSetup(tcx)
			}

			ctx := ctxWithAuth(tt.userID, tt.tokenExists)

			resp, err := tcx.server.ListUserURLs(ctx, &emptypb.Empty{})

			if tt.expectError {
				assertGRPCCode(t, err, tt.expectedCode)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.expectEmpty {
				assert.Empty(t, resp.GetUrl())
				return
			}

			require.Len(t, resp.GetUrl(), len(tt.expectedPairs))

			for _, item := range resp.GetUrl() {
				original, ok := tt.expectedPairs[item.GetShortUrl()]
				require.True(t, ok, "unexpected short url: %s", item.GetShortUrl())
				assert.Equal(t, original, item.GetOriginalUrl())
			}
		})
	}
}
