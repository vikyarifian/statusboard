package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Device struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	Target         string        `json:"target"`
	CheckType      string        `json:"check_type"`
	LocationCabang string        `json:"location_cabang"`
	CurrentStatus  string        `json:"current_status"`
	LastLatencyMs  sql.NullInt64 `json:"last_latency_ms"`
	LastCheckedAt  time.Time     `json:"last_checked_at"`
}

