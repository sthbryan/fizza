package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalGuardAllowsRequestsWithoutOrigin(t *testing.T) {
	ts, _ := newTestServer(t, "demo")
	resp, _ := doJSON(t, http.MethodGet, ts.URL+"/v1/projects", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLocalGuardRejectsCrossOriginWrite(t *testing.T) {
	ts, _ := newTestServer(t, "demo")

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/projects", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	env := decode(t, resp.Body)
	require.NotNil(t, env.Error)
	assert.Equal(t, "FORBIDDEN", env.Error.Code)

	listResp, body := doJSON(t, http.MethodGet, ts.URL+"/v1/projects", nil)
	defer listResp.Body.Close()
	assert.NotContains(t, string(body), "evil")
}

func TestLocalGuardRejectsCrossOriginRead(t *testing.T) {
	ts, _ := newTestServer(t, "demo")

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/projects", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://attacker.test")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestLocalGuardRejectsRebindingHost(t *testing.T) {
	ts, _ := newTestServer(t, "demo")

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/projects", nil)
	require.NoError(t, err)
	req.Host = "board.attacker.test"

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestOriginIsLocal(t *testing.T) {
	allowed := []string{
		"http://127.0.0.1:6500",
		"http://localhost:6500",
		"http://LOCALHOST:5173",
		"http://[::1]:6500",
		"https://127.0.0.1",
		"http://127.0.0.1:5173",
	}
	for _, origin := range allowed {
		assert.True(t, originIsLocal(origin), "expected %q to be allowed", origin)
	}

	rejected := []string{
		"",
		"null",
		"https://evil.example",
		"http://127.0.0.1.evil.example",
		"http://evil.example:6500",
		"file://",
		"http://10.0.0.5:6500",
		"http://[::2]:6500",
	}
	for _, origin := range rejected {
		assert.False(t, originIsLocal(origin), "expected %q to be rejected", origin)
	}
}

func TestHostIsRebindSafe(t *testing.T) {
	allowed := []string{
		"127.0.0.1:6500",
		"localhost:6500",
		"localhost",
		"[::1]:6500",
		"192.168.1.20:9090",
	}
	for _, host := range allowed {
		assert.True(t, hostIsRebindSafe(host), "expected %q to be allowed", host)
	}

	rejected := []string{
		"",
		"board.attacker.test",
		"board.attacker.test:6500",
		"fizza.local:6500",
	}
	for _, host := range rejected {
		assert.False(t, hostIsRebindSafe(host), "expected %q to be rejected", host)
	}
}

func TestLocalGuardPassesThroughOnLoopbackOrigin(t *testing.T) {
	handler := localGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:6500/v1/projects", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTeapot, rec.Code)
}
