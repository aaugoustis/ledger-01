package auth

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
)

type actorKey struct{}

// WithActor stores the authenticated caller in the context.
func WithActor(ctx context.Context, a ledger.Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// ActorFrom returns the authenticated caller. Every Connect handler runs behind the interceptor,
// so a missing actor is a programming error reported as unauthenticated.
func ActorFrom(ctx context.Context) (ledger.Actor, error) {
	a, ok := ctx.Value(actorKey{}).(ledger.Actor)
	if !ok {
		return ledger.Actor{}, connect.NewError(connect.CodeUnauthenticated, errors.New("sign in required"))
	}
	return a, nil
}

// Interceptor authenticates every unary RPC from the session cookie.
func (s *Sessions) Interceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			actor, err := s.Resolve(ctx, TokenFromHeader(req.Header()))
			if errors.Is(err, ErrNoSession) {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("sign in required"))
			}
			if err != nil {
				return nil, connect.NewError(connect.CodeUnavailable, err)
			}
			return next(WithActor(ctx, actor), req)
		}
	})
}
