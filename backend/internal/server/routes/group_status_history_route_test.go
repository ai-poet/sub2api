package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fork：/group-status/history（批量）与 /group-status/:groupId/history 共存，
// 两条路径各自命中自己的处理函数。
func TestGroupStatusBatchHistoryRouteCoexistsWithPerGroupRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1/group-status")
	g.GET("", func(c *gin.Context) { c.String(http.StatusOK, "list") })
	g.GET("/history", func(c *gin.Context) { c.String(http.StatusOK, "batch") })
	g.GET("/:groupId/history", func(c *gin.Context) { c.String(http.StatusOK, "one:"+c.Param("groupId")) })

	for path, want := range map[string]string{
		"/api/v1/group-status/history":    "batch",
		"/api/v1/group-status/12/history": "one:12",
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, w.Code, path)
		require.Equal(t, want, w.Body.String(), path)
	}
}
