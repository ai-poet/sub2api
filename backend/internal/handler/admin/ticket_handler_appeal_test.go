package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ticketUserLookupStub struct{ users map[int64]*service.User }

func (s ticketUserLookupStub) GetByID(_ context.Context, id int64) (*service.User, error) {
	if u, ok := s.users[id]; ok {
		return u, nil
	}
	return nil, service.ErrUserNotFound
}

// fork：申诉工单详情带上发起人当前账号状态（前端据此显示「恢复账户」）。
func TestAdminTicketDetailIncludesUserStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTicketRepoMem()
	svc := service.NewTicketService(repo)
	created, err := svc.CreateAppeal(context.Background(), service.TicketActor{UserID: 21, Email: "banned@example.com"}, "请求解封", "正文", "")
	require.NoError(t, err)

	h := NewTicketHandler(svc)
	h.users = ticketUserLookupStub{users: map[int64]*service.User{21: {ID: 21, Status: service.StatusDisabled}}}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		c.Next()
	})
	router.GET("/admin/tickets/:id", h.Get)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/tickets/"+strconv.FormatInt(created.ID, 10), nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Data struct {
			Ticket struct {
				Category string `json:"category"`
				User     struct {
					ID     int64  `json:"id"`
					Status string `json:"status"`
				} `json:"user"`
			} `json:"ticket"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, service.TicketCategoryAppeal, resp.Data.Ticket.Category)
	require.Equal(t, service.StatusDisabled, resp.Data.Ticket.User.Status)

	// 没有用户查询时不带状态，行为与原来一致
	h.users = nil
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/tickets/"+strconv.FormatInt(created.ID, 10), nil))
	require.NotContains(t, w.Body.String(), `"status":"disabled"`)
}
