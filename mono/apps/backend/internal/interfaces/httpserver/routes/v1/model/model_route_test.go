package model

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"jan-server/mono/apps/backend/internal/domain"
	modelresponses "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/model"
)

func TestShouldHideExperimental(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	catalog := &modelresponses.ModelCatalogResponse{Experimental: true}

	// No principal -> hide
	if !shouldHideExperimental(ctx, catalog) {
		t.Fatalf("expected experimental catalog to be hidden without flag")
	}

	// With principal but without flag -> hide
	ctx.Set("principal", domain.Principal{})
	if !shouldHideExperimental(ctx, catalog) {
		t.Fatalf("expected experimental catalog to be hidden without feature flag")
	}

	// With feature flag -> visible
	ctx.Set("principal", domain.Principal{FeatureFlags: []string{"experimental_models"}})
	if shouldHideExperimental(ctx, catalog) {
		t.Fatalf("expected experimental catalog to be visible when flag present")
	}

	// Non experimental should never be hidden
	regular := &modelresponses.ModelCatalogResponse{Experimental: false}
	ctx.Set("principal", domain.Principal{})
	if shouldHideExperimental(ctx, regular) {
		t.Fatalf("expected non-experimental catalog to be visible")
	}
}
