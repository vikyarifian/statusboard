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

