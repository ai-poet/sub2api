package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type payCatalogReaderStub struct {
	called bool
}

func (s *payCatalogReaderStub) GetCatalog(context.Context, int64) (*service.ModelCatalogResponse, error) {
	panic("must not use the buyer's accessible catalog")
}

func (s *payCatalogReaderStub) GetPayGroupCatalog(_ context.Context, group service.Group) (*service.ModelCatalogResponse, error) {
	s.called = true
	price := 1.5
	return &service.ModelCatalogResponse{Items: []service.ModelCatalogItem{{
		Model: "claude-sonnet", DisplayName: "Claude Sonnet", BillingMode: "token",
		BestGroup:           service.ModelCatalogGroupRef{ID: group.ID, Name: "internal group"},
		EffectivePricingUSD: service.ModelCatalogPricing{InputPerMTokUSD: &price, Source: "internal-price-source"},
	}}}, nil
}

type payCatalogGroupStub struct{ status string }

func (s payCatalogGroupStub) GetGroup(_ context.Context, id int64) (*service.Group, error) {
	return &service.Group{ID: id, Status: s.status}, nil
}

func TestPayModelCatalogProjection(t *testing.T) {
	reader := &payCatalogReaderStub{}
	h := &ModelCatalogHandler{modelCatalogService: reader, payGroupReader: payCatalogGroupStub{service.StatusActive}}
	r := gin.New()
	r.GET("/groups/:id/models", h.ListPayGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/groups/10/models", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, reader.called)
	var body struct {
		Data struct {
			Models []struct {
				Model   string `json:"model"`
				Pricing struct {
					Input float64 `json:"input_per_mtok_usd"`
				} `json:"effective_pricing_usd"`
			} `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Models, 1)
	require.Equal(t, "claude-sonnet", body.Data.Models[0].Model)
	require.Equal(t, 1.5, body.Data.Models[0].Pricing.Input)
	for _, field := range []string{"best_group", "other_groups", "internal-price-source", `"source"`} {
		require.NotContains(t, w.Body.String(), field)
	}
}

func TestPayModelCatalogRejectsInvalidAndInactiveGroups(t *testing.T) {
	for _, path := range []string{"/groups/invalid/models", "/groups/0/models", "/groups/10/models"} {
		t.Run(path, func(t *testing.T) {
			reader := &payCatalogReaderStub{}
			h := &ModelCatalogHandler{modelCatalogService: reader, payGroupReader: payCatalogGroupStub{"disabled"}}
			r := gin.New()
			r.GET("/groups/:id/models", h.ListPayGroup)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			require.GreaterOrEqual(t, w.Code, 400)
			require.False(t, reader.called)
		})
	}
}
