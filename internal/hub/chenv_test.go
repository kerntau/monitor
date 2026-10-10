//go:build testing

package hub_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bt "github.com/henrygd/beszel/internal/tests"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func chenvRequest(handler http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
func chenvTestHub(t *testing.T) (*bt.TestHub, http.Handler, string) {
	t.Helper()
	t.Setenv("CHENV_MODE", "true")
	t.Setenv("AUTO_LOGIN", "public@example.com")
	h, mux := firstUserTestMux(t)
	su, err := bt.CreateSuperuser(h, "admin@example.com", "original-password-123")
	require.NoError(t, err)
	token, err := su.NewAuthToken()
	require.NoError(t, err)
	return h, mux, token
}
func chenvPublish(t *testing.T, h core.App, id, name string, published bool) {
	t.Helper()
	_, err := bt.CreateRecord(h, "chenv_systems", map[string]any{"system": id, "name": name, "public": published})
	require.NoError(t, err)
}
func TestChenvPublicPrivacyAndIsolation(t *testing.T) {
	h, mux, adminToken := chenvTestHub(t)
	defer h.Cleanup()
	public, err := bt.CreateUserWithRole(h, "public@example.com", "password123", "readonly")
	require.NoError(t, err)
	legacyToken, err := public.NewAuthToken()
	require.NoError(t, err)
	a, err := bt.CreateRecord(h, "systems", map[string]any{"name": "Oracle", "host": "198.51.100.42", "port": "45876", "status": "paused", "info": map[string]any{"cpu": 42.5, "mp": 23, "dp": 44, "la": []float64{1, 2, 3}, "wf": map[string]any{"secret": "ssid"}}})
	require.NoError(t, err)
	b, err := bt.CreateRecord(h, "systems", map[string]any{"name": "Hidden", "host": "2001:db8::1", "status": "paused"})
	require.NoError(t, err)
	chenvPublish(t, h, a.Id, "Oracle 198.51.100.42", true)
	chenvPublish(t, h, b.Id, "Hidden", false)
	_, err = bt.CreateRecord(h, "system_details", map[string]any{"system": a.Id, "hostname": "chen", "cpu": "Ampere", "cores": 4})
	require.NoError(t, err)
	_, err = bt.CreateRecord(h, "system_stats", map[string]any{"system": a.Id, "type": "1m", "stats": map[string]any{"cpu": 42.5, "ni": map[string]any{"198.51.100.42": []int{1, 2}}, "secret": "never-export"}})
	require.NoError(t, err)
	mon, err := bt.CreateRecord(h, "network_monitors", map[string]any{"system": a.Id, "target": "https://198.51.100.42/private", "protocol": "http", "interval": 30, "enabled": false})
	require.NoError(t, err)
	_, err = bt.CreateRecord(h, "chenv_monitors", map[string]any{"monitor": mon.Id, "name": "198.51.100.42 探测", "public": true})
	require.NoError(t, err)
	_, err = bt.CreateRecord(h, "network_monitor_stats", map[string]any{"system": a.Id, "monitor": mon.Id, "type": "1m", "created": time.Now().UnixMilli(), "total_count": 10, "success_count": 8, "res_sum": 16000})
	require.NoError(t, err)
	res := chenvRequest(mux, "GET", "/api/chenv/public/systems/"+a.Id, "", nil)
	require.Equal(t, 200, res.Code, res.Body.String())
	text := res.Body.String()
	for _, secret := range []string{"198.51.100.42", "45876", "private", "never-export", "ssid", "\"users\"", "\"token\""} {
		require.NotContains(t, text, secret)
	}
	require.Contains(t, text, "Ampere")
	require.Contains(t, text, "chen")
	require.Contains(t, text, "42.5")
	require.Contains(t, text, "\"loss\":20")
	require.Contains(t, text, "\"latency\":2")
	probes := chenvRequest(mux, "GET", "/api/chenv/public/monitors", "", nil)
	require.Equal(t, 200, probes.Code, probes.Body.String())
	require.Contains(t, probes.Body.String(), "comparisonKey")
	for _, secret := range []string{"198.51.100.42", "private", "server", "port"} {
		require.NotContains(t, probes.Body.String(), "\""+secret+"\"")
	}
	require.NotContains(t, probes.Body.String(), "198.51.100.42")
	stats := chenvRequest(mux, "GET", "/api/chenv/public/monitor-stats?ids="+mon.Id, "", nil)
	require.Equal(t, 200, stats.Code, stats.Body.String())
	require.Contains(t, stats.Body.String(), "res_sum")
	require.Equal(t, 404, chenvRequest(mux, "GET", "/api/chenv/public/monitor-stats?ids=unknown", "", nil).Code)
	hidden, err := bt.CreateRecord(h, "network_monitors", map[string]any{"system": b.Id, "target": "hidden.example", "protocol": "icmp", "interval": 30})
	require.NoError(t, err)
	_, err = bt.CreateRecord(h, "chenv_monitors", map[string]any{"monitor": hidden.Id, "name": "隐藏", "public": true})
	require.NoError(t, err)
	require.Equal(t, 404, chenvRequest(mux, "GET", "/api/chenv/public/monitor-stats?ids="+mon.Id+","+hidden.Id, "", nil).Code)
	require.NotContains(t, chenvRequest(mux, "GET", "/api/chenv/public/monitors", "", nil).Body.String(), hidden.Id)
	list := chenvRequest(mux, "GET", "/api/chenv/public/systems", "", nil)
	require.Equal(t, 200, list.Code)
	require.NotContains(t, list.Body.String(), b.Id)
	require.Equal(t, 404, chenvRequest(mux, "GET", "/api/chenv/public/systems/"+b.Id, "", nil).Code)
	require.Equal(t, 400, chenvRequest(mux, "GET", "/api/chenv/public/systems/"+a.Id+"?range=unbounded", "", nil).Code)
	for _, token := range []string{"", legacyToken} {
		for _, path := range []string{"/api/collections/systems/records", "/api/collections/system_stats/records", "/api/collections/network_monitors/records", "/api/chenv/admin/systems", "/api/beszel/info", "/api/collections/users/auth-refresh"} {
			require.Equal(t, 403, chenvRequest(mux, "GET", path, token, nil).Code, path)
		}
	}
	require.Equal(t, 403, chenvRequest(mux, "POST", "/api/collections/users/auth-with-password", "", map[string]any{"identity": "public@example.com", "password": "password123"}).Code)
	require.Equal(t, 404, chenvRequest(mux, "GET", "/_/", "", nil).Code)
	require.Equal(t, 200, chenvRequest(mux, "GET", "/chenv-admin", "", nil).Code)
	require.Equal(t, 404, chenvRequest(mux, "GET", "/containers", "", nil).Code)
	require.Equal(t, 404, chenvRequest(mux, "POST", "/api/chenv/public/systems", "", nil).Code)
	// Disabling masking is an explicit admin action; public records remain a whitelist.
	res = chenvRequest(mux, "PUT", "/api/chenv/admin/config", adminToken, map[string]any{"name": "Test", "favicon": "/static/chenv-avatar.png", "theme": "dark", "hideAddresses": false})
	require.Equal(t, 200, res.Code, res.Body.String())
	res = chenvRequest(mux, "GET", "/api/chenv/public/systems/"+a.Id, "", nil)
	require.Contains(t, res.Body.String(), "198.51.100.42")
	require.NotContains(t, res.Body.String(), "never-export")
}

func TestChenvAdminLifecycle(t *testing.T) {
	h, mux, token := chenvTestHub(t)
	defer h.Cleanup()
	res := chenvRequest(mux, "GET", "/api/chenv/public/systems", "", nil)
	require.JSONEq(t, "[]", res.Body.String())
	res = chenvRequest(mux, "GET", "/api/chenv/admin/universal-token?enable=1&permanent=1", token, nil)
	require.Equal(t, 200, res.Code, res.Body.String())
	require.Contains(t, res.Body.String(), "\"permanent\":true")
	var universal map[string]any
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &universal))
	_, err := h.FindFirstRecordByFilter("chenv_universal_tokens", "user != ''")
	require.NoError(t, err)
	require.Equal(t, 403, chenvRequest(mux, "GET", "/api/chenv/admin/universal-token", "", nil).Code)
	require.Equal(t, 200, chenvRequest(mux, "GET", "/api/chenv/admin/universal-token?enable=0", token, nil).Code)
	input := map[string]any{"name": "First", "host": "127.0.0.1", "port": "45876", "paused": true, "displayName": "公开一号", "public": true, "sort": 2}
	res = chenvRequest(mux, "POST", "/api/chenv/admin/systems", token, input)
	require.Equal(t, 200, res.Code, res.Body.String())
	var created map[string]string
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &created))
	id := created["id"]
	res = chenvRequest(mux, "GET", "/api/chenv/public/systems/"+id, "", nil)
	require.Equal(t, 200, res.Code)
	require.Contains(t, res.Body.String(), "\"history\":[]")
	require.Contains(t, res.Body.String(), "公开一号")
	res = chenvRequest(mux, "GET", "/api/chenv/admin/systems/"+id+"/agent", token, nil)
	require.Equal(t, 200, res.Code, res.Body.String())
	require.Contains(t, res.Body.String(), "ssh-ed25519")
	require.Contains(t, res.Body.String(), "token")
	input["name"] = "Second"
	input["displayName"] = "公开二号"
	input["sort"] = 0
	res = chenvRequest(mux, "POST", "/api/chenv/admin/systems", token, input)
	require.Equal(t, 200, res.Code)
	var second map[string]string
	_ = json.Unmarshal(res.Body.Bytes(), &second)
	res = chenvRequest(mux, "GET", "/api/chenv/public/systems", "", nil)
	require.Less(t, strings.Index(res.Body.String(), "公开二号"), strings.Index(res.Body.String(), "公开一号"))
	probe := map[string]any{"name": "连通性", "system": id, "target": "127.0.0.1", "protocol": "icmp", "interval": 30, "enabled": false, "public": true}
	res = chenvRequest(mux, "POST", "/api/chenv/admin/monitors", token, probe)
	require.Equal(t, 200, res.Code, res.Body.String())
	var p map[string]string
	_ = json.Unmarshal(res.Body.Bytes(), &p)
	probe["target"] = "127.0.0.2"
	res = chenvRequest(mux, "PUT", "/api/chenv/admin/monitors/"+p["id"], token, probe)
	require.Equal(t, 200, res.Code, res.Body.String())
	var updated map[string]string
	_ = json.Unmarshal(res.Body.Bytes(), &updated)
	require.NotEqual(t, p["id"], updated["id"])
	_, err = h.FindRecordById("network_monitors", p["id"])
	require.Error(t, err)
	require.Equal(t, 204, chenvRequest(mux, "DELETE", "/api/chenv/admin/monitors/"+updated["id"], token, nil).Code)
	input["name"] = "First"
	input["displayName"] = "一号已隐藏"
	input["public"] = false
	require.Equal(t, 200, chenvRequest(mux, "PUT", "/api/chenv/admin/systems/"+id, token, input).Code)
	require.Equal(t, 404, chenvRequest(mux, "GET", "/api/chenv/public/systems/"+id, "", nil).Code)
	require.Equal(t, 204, chenvRequest(mux, "DELETE", "/api/chenv/admin/systems/"+id, token, nil).Code)
	res = chenvRequest(mux, "PUT", "/api/chenv/admin/config", token, map[string]any{"name": "monitor", "favicon": "javascript:alert(1)", "theme": "dark", "hideAddresses": true})
	require.Equal(t, 400, res.Code)
	res = chenvRequest(mux, "PUT", "/api/chenv/admin/config", token, map[string]any{"name": "monitor", "favicon": "https://chenv.cn/avatar.png", "theme": "dark", "hideAddresses": true})
	require.Equal(t, 200, res.Code, res.Body.String())
	res = chenvRequest(mux, "GET", "/api/chenv/public/config", "", nil)
	require.Contains(t, res.Body.String(), "monitor")
	res = chenvRequest(mux, "POST", "/api/chenv/admin/password", token, map[string]any{"current": "bad-password", "password": "new-password-12345"})
	require.Equal(t, 400, res.Code)
	res = chenvRequest(mux, "POST", "/api/chenv/admin/password", token, map[string]any{"current": "original-password-123", "password": "new-password-12345"})
	require.Equal(t, 200, res.Code, res.Body.String())
	require.Equal(t, 403, chenvRequest(mux, "GET", "/api/chenv/admin/systems", token, nil).Code)
	res = chenvRequest(mux, "POST", "/api/collections/_superusers/auth-with-password", "", map[string]any{"identity": "admin@example.com", "password": "new-password-12345"})
	require.Equal(t, 200, res.Code, res.Body.String())
}

func TestChenvAgentDetectedNamesAndDisks(t *testing.T) {
	h, mux, _ := chenvTestHub(t)
	defer h.Cleanup()
	system, err := bt.CreateRecord(h, "systems", map[string]any{"name": "Oracle", "host": "node-auto", "port": "45876", "status": "paused", "info": map[string]any{"h": "node-auto", "rdn": "vda", "efs": map[string]any{"sdb": 14}}})
	require.NoError(t, err)
	chenvPublish(t, h, system.Id, "Oracle", true)
	// Agent snapshots arrive after creation; creation and pause hooks clear info.
	_, err = h.DB().NewQuery("UPDATE systems SET info = {:info} WHERE id = {:id}").Bind(dbx.Params{
		"id": system.Id, "info": `{"h":"node-auto","rdn":"vda","efs":{"sdb":14}}`,
	}).Execute()
	require.NoError(t, err)
	_, err = bt.CreateRecord(h, "system_details", map[string]any{"system": system.Id, "hostname": "node-auto", "cpu": "Ampere", "cores": 4})
	require.NoError(t, err)
	_, err = bt.CreateRecord(h, "system_stats", map[string]any{"system": system.Id, "type": "1m", "stats": map[string]any{"cpu": 12, "efs": map[string]any{"sdb": map[string]any{"d": 40, "du": 10, "wb": 1024, "password": "never-export", "r": "never-export"}}}})
	require.NoError(t, err)
	response := chenvRequest(mux, "GET", "/api/chenv/public/systems/"+system.Id, "", nil)
	require.Equal(t, 200, response.Code, response.Body.String())
	for _, value := range []string{"node-auto", "vda", "sdb", "\"du\":10"} {
		require.Contains(t, response.Body.String(), value)
	}
	require.NotContains(t, response.Body.String(), "never-export")
	require.NotContains(t, response.Body.String(), "\"host\":")
	require.NotContains(t, response.Body.String(), "45876")
}
