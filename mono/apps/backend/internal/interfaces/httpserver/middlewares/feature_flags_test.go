package middlewares

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"jan-server/mono/apps/backend/internal/domain"
)

func TestFeatureEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	if FeatureEnabled(ctx, "experimental_models") {
		t.Fatalf("expected false when no principal set")
	}

	ctx.Set(principalContextKey, domain.Principal{FeatureFlags: []string{"experimental_models"}})
	if !FeatureEnabled(ctx, "experimental_models") {
		t.Fatalf("expected true when feature flag present")
	}

	if FeatureEnabled(ctx, "other_flag") {
		t.Fatalf("expected false for non-matching flag")
	}
}
