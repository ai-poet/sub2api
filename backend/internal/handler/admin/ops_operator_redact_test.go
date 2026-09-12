package admin

import (
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func opsTestContext(t *testing.T, role string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/admin/ops/errors", nil)
	if role != "" {
		c.Set(string(middleware.ContextKeyUserRole), role)
	}
	return c
}

func TestOpsErrorLogsForViewer_OperatorGetsMaskedCopy(t *testing.T) {
	ip := "198.51.100.7"
	original := []*service.OpsErrorLog{{ID: 1, ClientIP: &ip, UserEmail: "alice@example.com"}, nil}

	c := opsTestContext(t, service.RoleOperator)
	out := opsErrorLogsForViewer(c, original)

	require.Len(t, out, 1, "nil entries are dropped")
	require.Equal(t, "198.51.100.x", *out[0].ClientIP)
	require.Equal(t, "alice@example.com", out[0].UserEmail, "email stays visible for troubleshooting")
	require.Equal(t, "198.51.100.7", *original[0].ClientIP, "original must not be mutated")
}

func TestOpsErrorLogsForViewer_AdminUnchanged(t *testing.T) {
	ip := "198.51.100.7"
	original := []*service.OpsErrorLog{{ID: 1, ClientIP: &ip}}

	out := opsErrorLogsForViewer(opsTestContext(t, service.RoleAdmin), original)
	require.Same(t, original[0], out[0])
	require.Equal(t, "198.51.100.7", *out[0].ClientIP)
}

func TestOpsErrorLogDetailForViewer(t *testing.T) {
	ip := "2001:db8:1:2:3:4:5:6"
	detail := &service.OpsErrorLogDetail{OpsErrorLog: service.OpsErrorLog{ID: 9, ClientIP: &ip}, ErrorBody: "{}"}

	masked := opsErrorLogDetailForViewer(opsTestContext(t, service.RoleOperator), detail)
	require.Equal(t, "2001:db8:1::x", *masked.ClientIP)
	require.Equal(t, "{}", masked.ErrorBody)
	require.Equal(t, "2001:db8:1:2:3:4:5:6", *detail.ClientIP)

	require.Same(t, detail, opsErrorLogDetailForViewer(opsTestContext(t, service.RoleAdmin), detail))
	require.Nil(t, opsErrorLogDetailForViewer(opsTestContext(t, service.RoleOperator), nil))

	bad := "garbage"
	dropped := opsErrorLogDetailForViewer(opsTestContext(t, service.RoleOperator), &service.OpsErrorLogDetail{OpsErrorLog: service.OpsErrorLog{ClientIP: &bad}})
	require.Nil(t, dropped.ClientIP, "unparseable IPs are dropped rather than leaked")
}

func TestOpsErrorLogDetailsForViewer(t *testing.T) {
	ip := "203.0.113.9"
	details := []*service.OpsErrorLogDetail{{OpsErrorLog: service.OpsErrorLog{ClientIP: &ip}}, nil}
	out := opsErrorLogDetailsForViewer(opsTestContext(t, service.RoleOperator), details)
	require.Len(t, out, 1)
	require.Equal(t, "203.0.113.x", *out[0].ClientIP)
}
