package handler

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"statusboard/internal/checker"
	"statusboard/internal/database"
)

type Handler struct {
	db        *database.DB
	templates *template.Template
}

type DashboardData struct {
	Devices      []database.Device
	TotalCount   int
	OnlineCount  int
	OfflineCount int
}

func NewHandler(db *database.DB) (*Handler, error) {
	h := &Handler{
		db: db,
	}

	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return "-"
			}
			return t.Format("02 Jan 2006 15:04:05")
		},
		"formatNullTime": func(nt sql.NullTime) string {
			if !nt.Valid {
				return "Active"
			}
			return nt.Time.Format("02 Jan 2006 15:04:05")
		},
		"formatLatency": func(l sql.NullInt64) string {
			if !l.Valid {
				return "N/A"
			}
			return fmt.Sprintf("%d ms", l.Int64)
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).Parse(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	h.templates = tmpl
	return h, nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.HandleDashboard)
	mux.HandleFunc("GET /grid", h.HandleGrid)
	mux.HandleFunc("POST /devices/{id}/check", h.HandleManualCheck)
	mux.HandleFunc("GET /incidents", h.HandleIncidents)
}

func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	devices, err := h.db.GetAllDevices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := buildDashboardData(devices)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleGrid(w http.ResponseWriter, r *http.Request) {
	devices, err := h.db.GetAllDevices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := buildDashboardData(devices)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "grid", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// htmx limitation: hx-target cannot dynamically target parent table row when replacing components inline without exact ID matching, so we render single row templates
func (h *Handler) HandleManualCheck(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	dev, err := h.db.GetDeviceByID(id)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	res := checker.PerformCheck(dev.Target, dev.CheckType, 2*time.Second)
	now := time.Now()

	var latencyPtr *int64
	if res.Status != "OFFLINE" {
		latencyPtr = &res.LatencyMs
	}

	if err := h.db.UpdateDeviceStatus(dev.ID, res.Status, latencyPtr, now); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.Status != "OFFLINE" {
		_ = h.db.InsertLatencyRecord(dev.ID, res.LatencyMs, now)
	}

	if dev.CurrentStatus != "OFFLINE" && res.Status == "OFFLINE" {
		note := fmt.Sprintf("Manual check detected failure for %s (%s)", dev.Name, dev.LocationCabang)
		_, _ = h.db.CreateIncident(dev.ID, now, note)
	} else if dev.CurrentStatus == "OFFLINE" && res.Status != "OFFLINE" {
		_ = h.db.ResolveActiveIncident(dev.ID, now)
	}

	updatedDev, err := h.db.GetDeviceByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "device_row", updatedDev); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) HandleIncidents(w http.ResponseWriter, r *http.Request) {
	incidents, err := h.db.GetIncidents(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "incidents", incidents); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func buildDashboardData(devices []database.Device) DashboardData {
	data := DashboardData{
		Devices:    devices,
		TotalCount: len(devices),
	}
	for _, d := range devices {
		if d.CurrentStatus == "ONLINE" {
			data.OnlineCount++
		} else if d.CurrentStatus == "OFFLINE" {
			data.OfflineCount++
		}
	}
	return data
}

const htmlContent = `
{{define "index"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>IT Infrastructure Statusboard</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; background: #f4f6f8; margin: 0; padding: 20px; color: #333; }
        .container { max-width: 1100px; margin: 0 auto; }
        h1 { margin-top: 0; font-size: 24px; color: #1e293b; }
        nav { margin-bottom: 20px; display: flex; gap: 10px; }
        nav button { background: #0f172a; color: #fff; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; }
        nav button:hover { background: #334155; }
        .card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); margin-bottom: 20px; }
        .stats { display: flex; gap: 20px; margin-bottom: 15px; }
        .stat-box { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; padding: 12px 20px; flex: 1; }
        .stat-val { font-size: 22px; font-weight: bold; margin-top: 4px; }
        table { width: 100%; border-collapse: collapse; text-align: left; }
        th, td { padding: 12px; border-bottom: 1px solid #e2e8f0; }
        th { background: #f8fafc; font-size: 13px; text-transform: uppercase; letter-spacing: 0.5px; color: #64748b; }
        .badge { font-weight: bold; font-size: 11px; padding: 4px 8px; border-radius: 12px; text-transform: uppercase; }
        .badge-ONLINE { background: #dcfce7; color: #15803d; }
        .badge-OFFLINE { background: #fee2e2; color: #b91c1c; }
        .badge-DEGRADED { background: #fef3c7; color: #b45309; }
        .badge-UNKNOWN { background: #f1f5f9; color: #475569; }
        .btn-check { background: #0284c7; color: white; border: none; padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 12px; }
        .btn-check:hover { background: #0369a1; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Operations IT Statusboard</h1>
        <nav>
            <button hx-get="/grid" hx-target="#main-content">Grid Status</button>
            <button hx-get="/incidents" hx-target="#main-content">Incident Log</button>
        </nav>
        <div id="main-content">
            {{template "grid" .}}
        </div>
    </div>
</body>
</html>
{{end}}

{{define "grid"}}
<div hx-get="/grid" hx-trigger="every 10s" hx-swap="outerHTML" class="card">
    <div class="stats">
        <div class="stat-box">Total Monitored<div class="stat-val">{{.TotalCount}}</div></div>
        <div class="stat-box">Online<div class="stat-val" style="color:#15803d">{{.OnlineCount}}</div></div>
        <div class="stat-box">Offline<div class="stat-val" style="color:#b91c1c">{{.OfflineCount}}</div></div>
    </div>
    <table>
        <thead>
            <tr>
                <th>Cabang / Location</th>
                <th>Device / Service</th>
                <th>Target</th>
                <th>Status</th>
                <th>Latency</th>
                <th>Last Checked</th>
                <th>Action</th>
            </tr>
        </thead>
        <tbody>
            {{range .Devices}}
                {{template "device_row" .}}
            {{end}}
        </tbody>
    </table>
</div>
{{end}}

{{define "device_row"}}
<tr id="device-row-{{.ID}}">
    <td><strong>{{.LocationCabang}}</strong></td>
    <td>{{.Name}}</td>
    <td><code>{{.Target}}</code> ({{.CheckType}})</td>
    <td><span class="badge badge-{{.CurrentStatus}}">{{.CurrentStatus}}</span></td>
    <td>{{formatLatency .LastLatencyMs}}</td>
    <td>{{formatTime .LastCheckedAt}}</td>
    <td>
        <button class="btn-check" hx-post="/devices/{{.ID}}/check" hx-target="#device-row-{{.ID}}" hx-swap="outerHTML">
            Re-check
        </button>
    </td>
</tr>
{{end}}

{{define "incidents"}}
<div class="card">
    <h2>Incident Log</h2>
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>Cabang</th>
                <th>Device Name</th>
                <th>Started At</th>
                <th>Resolved At</th>
                <th>Note</th>
            </tr>
        </thead>
        <tbody>
            {{range .}}
            <tr>
                <td>#{{.ID}}</td>
                <td><strong>{{.Cabang}}</strong></td>
                <td>{{.DeviceName}}</td>
                <td>{{formatTime .StartedAt}}</td>
                <td>{{formatNullTime .ResolvedAt}}</td>
                <td>{{if .Note.Valid}}{{.Note.String}}{{else}}-{{end}}</td>
            </tr>
            {{else}}
            <tr>
                <td colspan="6" style="text-align:center;">No incidents logged.</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}
`
