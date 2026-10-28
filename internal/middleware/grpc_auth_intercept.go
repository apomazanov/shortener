package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	my_jwt "github.com/apomazanov/shortener/internal/jwt"
)

// NewAuthInterceptor returns a pointer to interceptor that is used for
// authorization procedure in gRPC stack.
func NewAuthInterceptor(jwtData *my_jwt.Data, log *zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Skip auth check for GET request
		if info.FullMethod == "/shortener.v1.ShortenerService/ExpandURL" {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			ctx = context.WithValue(ctx, TokenExistsKey, false)
			return handler(ctx, req)
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			ctx = context.WithValue(ctx, TokenExistsKey, false)
			return handler(ctx, req)
		}

		authHeader := values[0]
		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			ctx = context.WithValue(ctx, TokenExistsKey, false)
			return handler(ctx, req)
		}

		ctx = context.WithValue(ctx, TokenExistsKey, true)
		tokenStr := strings.TrimPrefix(authHeader, prefix)

		// parsing token

		claims := my_jwt.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
				return jwtData.SigningKey, nil
			}
			return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
		})

		// Token is OK

		if err == nil && token.Valid {
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			return handler(ctx, req)
		}

		// Token signature is OK but expired, renewing

		if errors.Is(err, jwt.ErrTokenExpired) {
			userID := claims.UserID

			newToken, err := jwtData.CreateTokenWithUserID(userID)
			if err != nil {
				log.Error().
					Err(err).
					Msg("token creation failed")
				return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
			}

			header := metadata.Pairs("authorization", "Bearer "+newToken)
			if err := grpc.SetHeader(ctx, header); err != nil {
				return nil, status.Error(codes.Internal, codes.Internal.String())
			}

			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		}

		// If token is garbage, just skipping it

		return handler(ctx, req)
	}
}
