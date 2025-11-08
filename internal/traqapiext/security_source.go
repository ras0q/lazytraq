package traqapiext

import (
	"context"

	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/ras0q/lazytraq/internal/traqapi"
	traqoauth2 "github.com/traPtitech/go-traq-oauth2"
)

type SecuritySource struct {
	AccessToken string
}

var _ traqapi.SecuritySource = (*SecuritySource)(nil)

func NewSecuritySource(accessToken string) *SecuritySource {
	return &SecuritySource{
		AccessToken: accessToken,
	}
}

// BearerAuth implements traqapi.SecuritySource.
func (s *SecuritySource) BearerAuth(ctx context.Context, operationName traqapi.OperationName) (traqapi.BearerAuth, error) {
	return traqapi.BearerAuth{}, ogenerrors.ErrSkipClientSecurity
}

// OAuth2 implements traqapi.SecuritySource.
func (s *SecuritySource) OAuth2(ctx context.Context, operationName traqapi.OperationName) (traqapi.OAuth2, error) {
	return traqapi.OAuth2{
		Token:  s.AccessToken,
		Scopes: []string{traqoauth2.ScopeRead, traqoauth2.ScopeWrite},
	}, nil
}
