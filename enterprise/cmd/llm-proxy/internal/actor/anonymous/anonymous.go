package anonymous

import (
	"context"
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/llm-proxy/internal/actor"
)

type Source struct {
	allowAnonymous bool
}

func NewSource(allowAnonymous bool) *Source {
	return &Source{allowAnonymous: allowAnonymous}
}

var _ actor.Source = &Source{}

func (s *Source) Name() string { return "anonymous" }

func (s *Source) Get(ctx context.Context, token string) (*actor.Actor, error) {
	// This source only handles completely anonymous requests.
	if token != "" {
		return nil, actor.ErrNotFromSource{}
	}
	return &actor.Actor{
		ID:            "anonymous", // TODO: Make this IP-based?
		Key:           token,
		AccessEnabled: s.allowAnonymous,
		RateLimit: actor.RateLimit{
			Limit:    50,
			Interval: 60 * time.Minute,
		},
		Source: s,
	}, nil
}