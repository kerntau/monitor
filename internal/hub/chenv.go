package hub

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/henrygd/beszel/internal/hub/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func chenvEnabled() bool { v, _ := utils.GetEnv("CHENV_MODE"); return v != "false" }

type chenvConfig struct {
	Name          string `json:"name"`
	Favicon       string `json:"favicon"`
	Theme         string `json:"theme"`
	HideAddresses bool   `json:"hideAddresses"`
}

func (h *Hub) chenvConfig() (chenvConfig, error) {
	c := chenvConfig{Name: "chenv", Favicon: "/static/chenv-avatar.png", Theme: "system", HideAddresses: true}
	r, err := h.FindFirstRecordByFilter("chenv_config", "")
	if err != nil {
		return c, err
	}
	err = r.UnmarshalJSONField("value", &c)
	return c, err
}

// A dedicated public projection is used instead of exporting PocketBase records.
func chenvFields(r *core.Record, fields ...string) map[string]any {
	out := map[string]any{}
	for _, f := range fields {
		out[f] = r.Get(f)
	}
	return out
}
func chenvJSONFields(r *core.Record, field string, keys ...string) map[string]any {
	var data map[string]any
	_ = r.UnmarshalJSONField(field, &data)
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := data[k]; ok {
			out[k] = v
		}
	}
	return out
}

// Numeric chart metrics cannot carry arbitrary nested configuration or strings.
func chenvMetricValue(value any) any {
	switch v := value.(type) {
	case float64, float32, int, int64, uint64:
		return value
	case []any:
		for _, item := range v {
			if chenvMetricValue(item) == nil {
				return nil
			}
		}
		return v
	default:
		return nil
	}
}

var chenvIPText = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b|\[[0-9a-fA-F:]+\]|\b[0-9a-fA-F]{0,4}:[0-9a-fA-F:]*:[0-9a-fA-F:]*\b`)

func chenvRedact(v any, secrets []string) any {
	switch x := v.(type) {
	case string:
		for _, s := range secrets {
			if len(s) > 2 {
				x = strings.ReplaceAll(x, s, "[隐藏]")
			}
		}
		return chenvIPText.ReplaceAllStringFunc(x, func(candidate string) string {
			if net.ParseIP(strings.Trim(candidate, "[]")) != nil {
				return "[隐藏]"
			}
			return candidate
		})
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			key := chenvRedact(k, secrets).(string)
			out[key] = chenvRedact(val, secrets)
		}
		return out
	case []any:
		for i, val := range x {
			x[i] = chenvRedact(val, secrets)
		}
		return x
	default:
		return v
	}
}

func (h *Hub) chenvDisplay(collection, field, id string) (*core.Record, error) {
	return h.FindFirstRecordByFilter(collection, field+" = {:id}", dbx.Params{"id": id})
}
func (h *Hub) chenvPublicSystem(id string) (*core.Record, *core.Record, error) {
	d, err := h.chenvDisplay("chenv_systems", "system", id)
	if err != nil || !d.GetBool("public") {
		return nil, nil, fmt.Errorf("not published")
	}
	r, err := h.FindRecordById("systems", id)
	return r, d, err
}

func (h *Hub) chenvSystemView(r, d *core.Record, c chenvConfig) map[string]any {
	out := chenvFields(r, "id", "status", "updated")
	out["name"] = d.GetString("name")
	out["sort"] = d.GetInt("sort")
	if out["name"] == "" {
		out["name"] = r.GetString("name")
	}
	out["info"] = chenvJSONFields(r, "info", "cpu", "mp", "dp", "la", "u", "t", "h", "k", "c", "m", "os", "v", "b", "bb", "rdn")
	if extra, ok := chenvJSONFields(r, "info", "efs")["efs"].(map[string]any); ok {
		filtered := map[string]any{}
		for name, value := range extra {
			if value = chenvMetricValue(value); value != nil {
				filtered[name] = value
			}
		}
		out["info"].(map[string]any)["efs"] = filtered
	}
	details, err := h.FindFirstRecordByFilter("system_details", "system = {:id}", dbx.Params{"id": r.Id})
	if err == nil {
		out["hardware"] = chenvFields(details, "hostname", "kernel", "cores", "threads", "cpu", "os", "os_name", "arch", "memory")
	}
	if !c.HideAddresses {
		out["host"] = r.GetString("host")
		out["port"] = r.GetString("port")
	}
	if c.HideAddresses {
		// Agent-reported hostnames describe the machine, independently of its connection address.
		hardware := out["hardware"]
		hostname := r.GetString("info.h")
		if info, ok := out["info"].(map[string]any); ok {
			if value, ok := info["h"].(string); ok {
				hostname = value
			}
		}
		out = chenvRedact(out, []string{r.GetString("host")}).(map[string]any)
		if hardware != nil {
			out["hardware"] = chenvRedact(hardware, nil)
		}
		if hostname != "" {
			out["info"].(map[string]any)["h"] = chenvRedact(hostname, nil)
		}
	}
	return out
}

func (h *Hub) chenvPublicSystems(e *core.RequestEvent) error {
	c, err := h.chenvConfig()
	if err != nil {
		return err
	}
	ds, err := h.FindRecordsByFilter("chenv_systems", "public = true", "sort,id", 0, 0)
	if err != nil {
		return err
	}
	out := []map[string]any{}
	for _, d := range ds {
		r, err := h.FindRecordById("systems", d.GetString("system"))
		if err != nil {
			return err
		}
		out = append(out, h.chenvSystemView(r, d, c))
	}
	return e.JSON(200, out)
}

func chenvRange(e *core.RequestEvent) (string, time.Time, error) {
	ranges := map[string]struct {
		kind     string
		duration time.Duration
	}{"1h": {"1m", time.Hour}, "6h": {"1m", 6 * time.Hour}, "12h": {"10m", 12 * time.Hour}, "24h": {"20m", 24 * time.Hour}, "7d": {"120m", 7 * 24 * time.Hour}, "1w": {"120m", 7 * 24 * time.Hour}, "30d": {"480m", 30 * 24 * time.Hour}}
	v := e.Request.URL.Query().Get("range")
	if v == "" {
		v = "1h"
	}
	r, ok := ranges[v]
	if !ok {
		return "", time.Time{}, fmt.Errorf("invalid range")
	}
	return r.kind, time.Now().Add(-r.duration), nil
}

func (h *Hub) chenvPublicDetail(e *core.RequestEvent) error {
	r, d, err := h.chenvPublicSystem(e.Request.PathValue("id"))
	if err != nil {
		return e.NotFoundError("服务器未公开", nil)
	}
	c, err := h.chenvConfig()
	if err != nil {
		return err
	}
	kind, start, err := chenvRange(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	rows, err := h.FindRecordsByFilter("system_stats", "system = {:id} && type = {:kind} && created >= {:start}", "-created", 1800, 0, dbx.Params{"id": r.Id, "kind": kind, "start": start.UTC().Format("2006-01-02 15:04:05.000Z")})
	if err != nil {
		return err
	}
	history := []map[string]any{}
	slices.Reverse(rows)
	for _, row := range rows {
		stats := chenvJSONFields(row, "stats", "cpu", "cpum", "cpub", "cpus", "mp", "m", "mu", "mb", "mm", "mz", "s", "su", "dp", "d", "du", "la", "b", "bm", "ns", "nr", "nsm", "nrm", "dio", "diom", "dios", "diosm", "diot", "dr", "dw", "drm", "dwm")
		for key, value := range stats {
			if value = chenvMetricValue(value); value != nil {
				stats[key] = value
			} else {
				delete(stats, key)
			}
		}
		extra := chenvJSONFields(row, "stats", "efs")
		if filesystems, ok := extra["efs"].(map[string]any); ok {
			clean := map[string]any{}
			for name, raw := range filesystems {
				if values, ok := raw.(map[string]any); ok {
					metrics := map[string]any{}
					for _, key := range []string{"d", "du", "r", "w", "rm", "wm", "rb", "wb", "rbm", "wbm", "tr", "tw", "dios", "diosm"} {
						if value, ok := values[key]; ok {
							if value = chenvMetricValue(value); value != nil {
								metrics[key] = value
							}
						}
					}
					clean[name] = metrics
				}
			}
			stats["efs"] = clean
		}
		if c.HideAddresses {
			stats = chenvRedact(stats, []string{r.GetString("host")}).(map[string]any)
		}
		history = append(history, map[string]any{"created": row.Get("created"), "stats": stats})
	}
	out := h.chenvSystemView(r, d, c)
	out["history"] = history
	monitors, err := h.chenvMonitorViews(r, c, kind, start)
	if err != nil {
		return err
	}
	out["monitors"] = monitors
	return e.JSON(200, out)
}

func (h *Hub) chenvMonitorViews(system *core.Record, c chenvConfig, kind string, start time.Time) ([]map[string]any, error) {
	rows, err := h.FindRecordsByFilter("network_monitors", "system = {:id}", "created", 0, 0, dbx.Params{"id": system.Id})
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, r := range rows {
		d, err := h.chenvDisplay("chenv_monitors", "monitor", r.Id)
		if err != nil || !d.GetBool("public") {
			continue
		}
		v := chenvFields(r, "id", "system", "protocol", "interval", "enabled", "res", "resMin1h", "resMax1h", "resAvg1h", "loss1h", "updated")
		name := d.GetString("name")
		if c.HideAddresses && (name == r.GetString("target") || name == r.GetString("server")) {
			name = ""
		}
		v["name"] = name
		v["certInfo"] = chenvJSONFields(r, "certInfo", "expires", "issuer")
		if cfg, err := h.FindFirstRecordByFilter("chenv_config", ""); err == nil {
			hash := hmac.New(sha256.New, []byte(cfg.Id))
			identity, _ := json.Marshal([]any{r.GetString("protocol"), r.GetString("target"), r.GetInt("port"), r.GetString("server")})
			hash.Write(identity)
			v["comparisonKey"] = hex.EncodeToString(hash.Sum(nil))
		}
		if !c.HideAddresses {
			v["target"] = r.GetString("target")
			v["port"] = r.GetInt("port")
			v["server"] = r.GetString("server")
		}
		rows, err := h.FindRecordsByFilter("network_monitor_stats", "monitor = {:id} && type = {:kind} && created >= {:start}", "-created", 1800, 0, dbx.Params{"id": r.Id, "kind": kind, "start": start.UnixMilli()})
		if err != nil {
			return nil, err
		}
		history := []map[string]any{}
		slices.Reverse(rows)
		for _, row := range rows {
			sample := chenvFields(row, "created")
			total, success := row.GetFloat("total_count"), row.GetFloat("success_count")
			var avg, loss float64
			if success > 0 {
				avg = row.GetFloat("res_sum") / success / 1000
			}
			if total > 0 {
				loss = (total - success) / total * 100
			}
			sample["latency"] = avg
			sample["res_avg"] = avg * 1000
			sample["res_min"] = row.GetFloat("res_min")
			sample["res_max"] = row.GetFloat("res_max")
			if success == 0 {
				sample["res_avg"] = nil
				sample["res_min"] = nil
				sample["res_max"] = nil
			}
			sample["loss"] = loss
			history = append(history, sample)
		}
		v["history"] = history
		if c.HideAddresses {
			v = chenvRedact(v, []string{system.GetString("host"), r.GetString("target"), r.GetString("server")}).(map[string]any)
		}
		out = append(out, v)
	}
	return out, nil
}

func (h *Hub) chenvAdminSystems(e *core.RequestEvent) error {
	rows, err := h.FindAllRecords("systems")
	if err != nil {
		return err
	}
	out := []map[string]any{}
	for _, r := range rows {
		v := chenvFields(r, "id", "name", "host", "port", "status", "info")
		d, err := h.chenvDisplay("chenv_systems", "system", r.Id)
		if err == nil {
			v["displayName"] = d.GetString("name")
			v["public"] = d.GetBool("public")
			v["sort"] = d.GetInt("sort")
		}
		out = append(out, v)
	}
	return e.JSON(200, out)
}

type chenvSystemInput struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        string `json:"port"`
	Paused      bool   `json:"paused"`
	DisplayName string `json:"displayName"`
	Public      bool   `json:"public"`
	Sort        int    `json:"sort"`
	Token       string `json:"token"`
}

func (h *Hub) chenvSaveSystem(e *core.RequestEvent) error {
	var in chenvSystemInput
	if err := e.BindBody(&in); err != nil {
		return e.BadRequestError("输入无效", nil)
	}
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Host) == "" || len(in.Name) > 120 || len(in.Host) > 512 {
		return e.BadRequestError("名称和连接地址必填", nil)
	}
	if in.Token != "" && !regexp.MustCompile(`^[A-Za-z0-9-]{20,64}$`).MatchString(in.Token) {
		return e.BadRequestError("Agent 令牌无效", nil)
	}
	id := e.Request.PathValue("id")
	var saved *core.Record
	err := h.RunInTransaction(func(app core.App) error {
		var r *core.Record
		var err error
		if id == "" {
			c, err := app.FindCollectionByNameOrId("systems")
			if err != nil {
				return err
			}
			r = core.NewRecord(c)
		} else {
			r, err = app.FindRecordById("systems", id)
			if err != nil {
				return err
			}
		}
		changed := r.GetString("host") != in.Host || r.GetString("port") != in.Port
		r.Set("name", strings.TrimSpace(in.Name))
		r.Set("host", strings.TrimSpace(in.Host))
		r.Set("port", in.Port)
		if in.Paused {
			r.Set("status", "paused")
		} else if id == "" || changed || r.GetString("status") == "paused" {
			r.Set("status", "pending")
		}
		if err = app.Save(r); err != nil {
			return err
		}
		if id == "" {
			fc, err := app.FindCollectionByNameOrId("fingerprints")
			if err != nil {
				return err
			}
			fp := core.NewRecord(fc)
			fp.Set("system", r.Id)
			if in.Token == "" {
				in.Token = security.RandomString(40)
			}
			fp.Set("token", in.Token)
			if err = app.Save(fp); err != nil {
				return err
			}
		}
		d, err := app.FindFirstRecordByFilter("chenv_systems", "system = {:id}", dbx.Params{"id": r.Id})
		if err != nil {
			c, err := app.FindCollectionByNameOrId("chenv_systems")
			if err != nil {
				return err
			}
			d = core.NewRecord(c)
			d.Set("system", r.Id)
		}
		d.Set("name", in.DisplayName)
		d.Set("public", in.Public)
		d.Set("sort", in.Sort)
		saved = r
		return app.Save(d)
	})
	if err != nil {
		return e.BadRequestError("保存服务器失败", err)
	}
	return e.JSON(200, map[string]string{"id": saved.Id})
}

type chenvMonitorInput struct {
	System   string `json:"system"`
	Target   string `json:"target"`
	Protocol string `json:"protocol"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Interval int    `json:"interval"`
	Enabled  bool   `json:"enabled"`
	Name     string `json:"name"`
	Public   bool   `json:"public"`
}

func (h *Hub) chenvAdminMonitors(e *core.RequestEvent) error {
	rows, err := h.FindAllRecords("network_monitors")
	if err != nil {
		return err
	}
	out := []map[string]any{}
	for _, r := range rows {
		v := chenvFields(r, "id", "system", "target", "protocol", "server", "port", "interval", "enabled", "res", "resMin1h", "resMax1h", "resAvg1h", "loss1h", "certInfo", "updated")
		if d, err := h.chenvDisplay("chenv_monitors", "monitor", r.Id); err == nil {
			v["name"] = d.GetString("name")
			v["public"] = d.GetBool("public")
		}
		out = append(out, v)
	}
	return e.JSON(200, out)
}
func (h *Hub) chenvSaveMonitor(e *core.RequestEvent) error {
	var in chenvMonitorInput
	if err := e.BindBody(&in); err != nil {
		return e.BadRequestError("输入无效", nil)
	}
	if !slices.Contains([]string{"icmp", "tcp", "http", "dns"}, in.Protocol) || in.Interval < 1 || in.Interval > 3600 || in.Port < 0 || in.Port > 65535 || in.Target == "" {
		return e.BadRequestError("探测参数无效，间隔需为 1 至 3600 秒", nil)
	}
	if in.Protocol == "tcp" && in.Port == 0 {
		return e.BadRequestError("TCP 端口必填", nil)
	}
	if in.Protocol != "tcp" {
		in.Port = 0
	}
	if in.Protocol != "dns" {
		in.Server = ""
	}
	id := e.Request.PathValue("id")
	var saved *core.Record
	err := h.RunInTransaction(func(app core.App) error {
		if _, err := app.FindRecordById("systems", in.System); err != nil {
			return err
		}
		c, err := app.FindCollectionByNameOrId("network_monitors")
		if err != nil {
			return err
		}
		r := core.NewRecord(c)
		r.Set("system", in.System)
		r.Set("target", in.Target)
		r.Set("protocol", in.Protocol)
		r.Set("server", in.Server)
		r.Set("port", in.Port)
		r.Set("interval", in.Interval)
		r.Set("enabled", in.Enabled)
		newID := generateMonitorID(in.System, *monitorConfigFromRecord(r))
		if id != "" {
			old, err := app.FindRecordById("network_monitors", id)
			if err != nil {
				return err
			}
			if id == newID {
				r = old
				r.Set("interval", in.Interval)
				r.Set("enabled", in.Enabled)
			} else {
				if _, err = app.FindRecordById("network_monitors", newID); err == nil {
					return fmt.Errorf("探测已存在")
				}
				if err = app.Delete(old); err != nil {
					return err
				}
			}
		}
		if err = app.Save(r); err != nil {
			return err
		}
		d, err := app.FindFirstRecordByFilter("chenv_monitors", "monitor = {:id}", dbx.Params{"id": r.Id})
		if err != nil {
			dc, err := app.FindCollectionByNameOrId("chenv_monitors")
			if err != nil {
				return err
			}
			d = core.NewRecord(dc)
			d.Set("monitor", r.Id)
		}
		d.Set("name", in.Name)
		d.Set("public", in.Public)
		saved = r
		return app.Save(d)
	})
	if err != nil {
		return e.BadRequestError("保存探测失败", err)
	}
	if saved.GetBool("enabled") {
		if err = h.upsertNetworkMonitor(saved, true); err != nil {
			h.Logger().Warn("Monitor will sync on reconnect", "monitor", saved.Id)
		}
	} else {
		_ = h.deleteNetworkMonitor(saved)
	}
	return e.JSON(200, map[string]string{"id": saved.Id})
}

func (h *Hub) chenvSaveConfig(e *core.RequestEvent) error {
	var c chenvConfig
	if err := e.BindBody(&c); err != nil {
		return e.BadRequestError("输入无效", nil)
	}
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 80 || !slices.Contains([]string{"light", "dark", "system"}, c.Theme) {
		return e.BadRequestError("站点名称或主题无效", nil)
	}
	u, err := url.Parse(c.Favicon)
	if err != nil || (u.Scheme != "https" && !(u.Scheme == "" && strings.HasPrefix(c.Favicon, "/") && !strings.HasPrefix(c.Favicon, "//"))) {
		return e.BadRequestError("图标必须是 HTTPS 地址或站内绝对路径", nil)
	}
	r, err := h.FindFirstRecordByFilter("chenv_config", "")
	if err != nil {
		return err
	}
	r.Set("value", c)
	if err = h.Save(r); err != nil {
		return err
	}
	return e.JSON(200, c)
}

func (h *Hub) chenvAgent(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if _, err := h.FindRecordById("systems", id); err != nil {
		return e.NotFoundError("服务器不存在", nil)
	}
	r, err := h.FindFirstRecordByFilter("fingerprints", "system = {:id}", dbx.Params{"id": id})
	if err != nil {
		c, err := h.FindCollectionByNameOrId("fingerprints")
		if err != nil {
			return err
		}
		r = core.NewRecord(c)
		r.Set("system", id)
		r.Set("token", security.RandomString(40))
		if err = h.Save(r); err != nil {
			return err
		}
	}
	return e.JSON(200, map[string]string{"key": h.pubKey, "token": r.GetString("token")})
}
func (h *Hub) chenvPassword(e *core.RequestEvent) error {
	var in struct {
		Current  string `json:"current"`
		Password string `json:"password"`
	}
	if err := e.BindBody(&in); err != nil || len(in.Password) < 12 || !e.Auth.ValidatePassword(in.Current) {
		return e.BadRequestError("当前密码不正确或新密码少于 12 位", nil)
	}
	e.Auth.SetPassword(in.Password)
	e.Auth.RefreshTokenKey()
	if err := h.Save(e.Auth); err != nil {
		return err
	}
	return e.JSON(200, map[string]bool{"ok": true})
}

func (h *Hub) chenvInitialize(app core.App) error {
	c, err := app.FindCollectionByNameOrId("systems")
	if err != nil {
		return err
	}
	c.Fields.GetByName("users").(*core.RelationField).Required = false
	if err = app.Save(c); err != nil {
		return err
	}
	settings := app.Settings()
	settings.Batch.Enabled = false
	return app.Save(settings)
}

func (h *Hub) registerChenv(se *core.ServeEvent) {
	// This runs after PB authentication but before collection handlers. No AUTO_LOGIN.
	se.Router.BindFunc(func(e *core.RequestEvent) error {
		path := e.Request.URL.Path
		if path == "/_" || strings.HasPrefix(path, "/_/") {
			return e.NotFoundError("Not found", nil)
		}
		if strings.HasPrefix(path, "/api/chenv/public/") || path == "/api/beszel/agent-connect" || path == "/api/health" {
			return e.Next()
		}
		// Only password login/refresh of superusers remain public. PB validates credentials.
		if path == "/api/collections/_superusers/auth-with-password" || path == "/api/collections/_superusers/auth-refresh" {
			return e.Next()
		}
		if strings.HasPrefix(path, "/api/") && (e.Auth == nil || !e.Auth.IsSuperuser()) {
			return e.ForbiddenError("仅管理员可访问", nil)
		}
		if strings.HasPrefix(path, "/api/beszel/") {
			return e.NotFoundError("Not found", nil)
		}
		return e.Next()
	})
	pub := se.Router.Group("/api/chenv/public")
	pub.BindFunc(func(e *core.RequestEvent) error {
		e.Response.Header().Set("Cache-Control", "no-store")
		return e.Next()
	})
	pub.GET("/config", func(e *core.RequestEvent) error {
		c, err := h.chenvConfig()
		if err != nil {
			return err
		}
		return e.JSON(200, c)
	})
	pub.GET("/monitors", h.chenvPublicMonitors)
	pub.GET("/monitor-stats", h.chenvPublicMonitorStats)
	pub.GET("/systems", h.chenvPublicSystems)
	pub.GET("/systems/{id}", h.chenvPublicDetail)
	se.Router.GET("/api/beszel/agent-connect", h.handleAgentConnect)
	admin := se.Router.Group("/api/chenv/admin")
	admin.Bind(apis.RequireAuth(core.CollectionNameSuperusers))
	admin.BindFunc(func(e *core.RequestEvent) error {
		e.Response.Header().Set("Cache-Control", "no-store")
		return e.Next()
	})
	admin.GET("/config", func(e *core.RequestEvent) error {
		c, err := h.chenvConfig()
		if err != nil {
			return err
		}
		return e.JSON(200, c)
	})
	admin.PUT("/config", h.chenvSaveConfig)
	admin.GET("/systems", h.chenvAdminSystems)
	admin.GET("/universal-token", h.getUniversalToken)
	admin.GET("/agent-info", func(e *core.RequestEvent) error {
		return e.JSON(200, map[string]string{"key": h.pubKey, "token": security.RandomString(40)})
	})
	admin.POST("/systems", h.chenvSaveSystem)
	admin.PUT("/systems/{id}", h.chenvSaveSystem)
	admin.GET("/systems/{id}/agent", h.chenvAgent)
	admin.GET("/monitors", h.chenvAdminMonitors)
	admin.POST("/monitors", h.chenvSaveMonitor)
	admin.PUT("/monitors/{id}", h.chenvSaveMonitor)
	for _, item := range []struct{ path, collection string }{{"/systems/{id}", "systems"}, {"/monitors/{id}", "network_monitors"}} {
		admin.DELETE(item.path, func(e *core.RequestEvent) error {
			r, err := h.FindRecordById(item.collection, e.Request.PathValue("id"))
			if err != nil {
				return e.NotFoundError("记录不存在", nil)
			}
			if err = h.Delete(r); err != nil {
				return err
			}
			return e.NoContent(http.StatusNoContent)
		})
	}
	admin.POST("/password", h.chenvPassword)
}

// Only published probes belonging to published systems are visible, including comparison requests.
func (h *Hub) chenvPublicMonitors(e *core.RequestEvent) error {
	c, err := h.chenvConfig()
	if err != nil {
		return err
	}
	kind, start, err := chenvRange(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	ds, err := h.FindRecordsByFilter("chenv_systems", "public=true", "sort,id", 0, 0)
	if err != nil {
		return err
	}
	out := []map[string]any{}
	for _, d := range ds {
		r, err := h.FindRecordById("systems", d.GetString("system"))
		if err != nil {
			continue
		}
		items, err := h.chenvMonitorViews(r, c, kind, start)
		if err != nil {
			return err
		}
		for _, item := range items {
			delete(item, "history")
			out = append(out, item)
		}
	}
	return e.JSON(200, out)
}
func (h *Hub) chenvPublicMonitorStats(e *core.RequestEvent) error {
	ids := strings.Split(e.Request.URL.Query().Get("ids"), ",")
	if len(ids) == 0 || len(ids) > 24 {
		return e.BadRequestError("最多比较 24 个探测", nil)
	}
	kind, start, err := chenvRange(e)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	out := []map[string]any{}
	for _, id := range ids {
		d, err := h.chenvDisplay("chenv_monitors", "monitor", id)
		if err != nil || !d.GetBool("public") {
			return e.NotFoundError("探测未公开", nil)
		}
		r, err := h.FindRecordById("network_monitors", id)
		if err != nil {
			return e.NotFoundError("探测不存在", nil)
		}
		if _, _, err = h.chenvPublicSystem(r.GetString("system")); err != nil {
			return e.NotFoundError("服务器未公开", nil)
		}
	}
	for _, id := range ids {
		rows, err := h.FindRecordsByFilter("network_monitor_stats", "monitor={:id} && type={:kind} && created>={:start}", "-created", 1800, 0, dbx.Params{"id": id, "kind": kind, "start": start.UnixMilli()})
		if err != nil {
			return err
		}
		slices.Reverse(rows)
		for _, row := range rows {
			out = append(out, chenvFields(row, "monitor", "created", "res_min", "res_max", "res_sum", "success_count", "total_count"))
		}
	}
	return e.JSON(200, out)
}
