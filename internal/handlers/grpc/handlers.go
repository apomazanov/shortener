package grpchandlers

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/apomazanov/shortener/internal/domain"
	"github.com/apomazanov/shortener/internal/validator"
	pb "github.com/apomazanov/shortener/pkg/api/proto/v1"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServer contains data needed for gRPC service.
type GRPCServer struct {
	pb.UnimplementedShortenerServiceServer
	business  domain.BusinessService
	cfg       domain.URLConfig
	log       *zerolog.Logger
	jwt       domain.JWT
	validator *validator.MyValidator
}

// New creates a GRPCServer instance.
func New(b domain.BusinessService, c domain.URLConfig, l *zerolog.Logger, j domain.JWT, v *validator.MyValidator) *GRPCServer {
	return &GRPCServer{
		business:  b,
		cfg:       c,
		log:       l,
		jwt:       j,
		validator: v,
	}
}

const aliasSize = 6

// ShortenURL provides functionality of long URL shortening.
func (s *GRPCServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {

	// check ctx for "user-id"

	authData := handleAuthData(ctx, s.jwt)
	if authData.err != nil {
		s.log.Error().
			Str("op", "grpc_handler.ShortenURL").
			Err(authData.err).
			Msg("Failed auth data handling")

		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	userID := authData.userID
	if !authData.valid {
		userID = authData.newUserID
	}

	// Reading request data
	original := req.GetUrl()

	// Validating
	originalToValidate := domain.JSONShortenRequest{URL: original}

	if err := s.validator.Validate(&originalToValidate); err != nil {
		return nil, status.Error(codes.InvalidArgument, codes.InvalidArgument.String())
	}

	// Getting alias

	var isDuplicate bool

	alias, err := s.business.CreateURLAlias(ctx, original, userID)
	if err != nil {

		if errors.Is(err, domain.ErrOriginalURLDuplicate) {
			isDuplicate = true
		} else {
			s.log.Error().
				Str("op", "grpc_handler.ShortenURL").
				Err(err).
				Str("alias", alias).
				Msg("alias creation failed")

			return nil, status.Error(codes.Internal, codes.Internal.String())
		}
	}

	// Auth data created for new users (user-id missing in ctx or invalid)
	if !authData.valid {
		header := metadata.Pairs("authorization", "Bearer "+authData.newToken)
		if err := grpc.SendHeader(ctx, header); err != nil {
			s.log.Error().
				Str("op", "grpc_handler.ShortenURL").
				Err(err).
				Msg("Failed to send metadata")

			return nil, status.Error(codes.Internal, codes.Internal.String())
		}
	}

	if isDuplicate {
		return nil, status.Error(codes.AlreadyExists, codes.AlreadyExists.String())
	}

	short := s.cfg.GetURLBase() + "/" + alias
	return &pb.URLShortenResponse{Result: &short}, nil
}

// ExpandURL provides functionality of getting original URL by alias.
func (s *GRPCServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {

	alias := req.GetId()

	if len(alias) != aliasSize {
		s.log.Info().
			Str("op", "grpc_handler.ExpandURL").
			Str("alias", alias).
			Msg("invalid alias")

		return nil, status.Error(codes.InvalidArgument, codes.InvalidArgument.String())
	}

	// Find original URL
	original, err := s.business.GetOriginalURL(ctx, alias)

	if err == nil {
		return &pb.URLExpandResponse{Result: &original}, nil
	}

	if errors.Is(err, domain.ErrFoundDeleted) {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	if errors.Is(err, domain.ErrNotFound) {
		s.log.Info().
			Str("op", "grpc_handler.ExpandURL").
			Err(err).
			Msg("alias not found")

		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	s.log.Error().
		Str("op", "grpc_handler.ExpandURL").
		Err(err).
		Msg("something went wrong")

	return nil, status.Error(codes.Internal, codes.Internal.String())
}

// ListUserURLs provides getting all URLs shortened by certain user.
func (s *GRPCServer) ListUserURLs(ctx context.Context, none *emptypb.Empty) (resp *pb.UserURLsResponse, err error) {

	// check ctx for "user-id"

	authData := handleAuthData(ctx, s.jwt)
	if authData.err != nil {
		s.log.Error().
			Str("op", "grpc_handler.ListUserURLs").
			Err(authData.err).
			Msg("Failed auth data handling")

		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	if !authData.exists {
		header := metadata.Pairs("authorization", "Bearer "+authData.newToken)
		if err := grpc.SendHeader(ctx, header); err != nil {
			s.log.Error().
				Str("op", "grpc_handler.ListUserURLs").
				Err(err).
				Msg("Failed to send metadata")

			return nil, status.Error(codes.Internal, codes.Internal.String())
		}
		return &pb.UserURLsResponse{}, nil
	}

	if !authData.valid {
		return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
	}

	// Find user's URLs
	data, err := s.business.GetUserURLs(ctx, authData.userID)
	if err != nil {

		if errors.Is(err, domain.ErrNotFound) {
			return &pb.UserURLsResponse{}, nil
		}

		s.log.Error().
			Str("op", "http_handler.ListUserURLs").
			Err(err).
			Msg("something went wrong")

		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	responseData := make([]*pb.URLData, 0, len(data))

	for k, v := range data {
		short := s.cfg.GetURLBase() + "/" + k
		responseData = append(responseData, &pb.URLData{
			ShortUrl:    &short,
			OriginalUrl: &v,
		})
	}

	return &pb.UserURLsResponse{
		Url: responseData,
	}, nil
}
