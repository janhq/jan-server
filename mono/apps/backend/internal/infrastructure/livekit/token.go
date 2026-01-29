package livekit

import (
	"time"

	"github.com/livekit/protocol/auth"

	"jan-server/mono/apps/backend/internal/config"
)

// TokenGenerator generates LiveKit access tokens.
type TokenGenerator struct {
	apiKey    string
	apiSecret string
}

// NewTokenGenerator creates a new token generator.
func NewTokenGenerator(cfg *config.Config) *TokenGenerator {
	return &TokenGenerator{
		apiKey:    cfg.LiveKitAPIKey,
		apiSecret: cfg.LiveKitAPISecret,
	}
}

// Generate creates a LiveKit access token for the given room and identity.
func (g *TokenGenerator) Generate(room, identity string, ttl time.Duration) (string, error) {
	at := auth.NewAccessToken(g.apiKey, g.apiSecret)

	canPublish := true
	canSubscribe := true
	canPublishData := true

	grant := &auth.VideoGrant{
		RoomJoin:       true,
		Room:           room,
		CanPublish:     &canPublish,
		CanSubscribe:   &canSubscribe,
		CanPublishData: &canPublishData,
	}

	at.AddGrant(grant).
		SetIdentity(identity).
		SetValidFor(ttl)

	return at.ToJWT()
}
