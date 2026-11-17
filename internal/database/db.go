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

type Incident struct {
	ID         int            `json:"id"`
	DeviceID   int            `json:"device_id"`
	DeviceName string         `json:"device_name"`
	Cabang     string         `json:"cabang"`
	StartedAt  time.Time      `json:"started_at"`
	ResolvedAt sql.NullTime   `json:"resolved_at"`
	Note       sql.NullString `json:"note"`
}

type LatencyRecord struct {
	ID         int       `json:"id"`
	DeviceID   int       `json:"device_id"`
	LatencyMs  int64     `json:"latency_ms"`
	RecordedAt time.Time `json:"recorded_at"`
}

type DB struct {
	*sql.DB
}

// sqlite3 limitation: PRAGMA foreign_keys = ON must be explicitly appended to DSN string or executed per connection because driver disables FK enforcement by default
func Open(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d := &DB{DB: db}
	if err := d.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return d, nil
}

func (db *DB) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		target TEXT NOT NULL,
		check_type TEXT NOT NULL,
		location_cabang TEXT NOT NULL,
		current_status TEXT NOT NULL,
		last_latency_ms INTEGER,
		last_checked_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS incidents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL REFERENCES devices(id),
		started_at DATETIME NOT NULL,
		resolved_at DATETIME,
		note TEXT
	);

	CREATE TABLE IF NOT EXISTS latency_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL REFERENCES devices(id),
		latency_ms INTEGER NOT NULL,
		recorded_at DATETIME NOT NULL
	);
	`
	_, err := db.Exec(schema)
	return err
}

func (db *DB) GetAllDevices() ([]Device, error) {
	query := `SELECT id, name, target, check_type, location_cabang, current_status, last_latency_ms, last_checked_at FROM devices ORDER BY id ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Name, &d.Target, &d.CheckType, &d.LocationCabang, &d.CurrentStatus, &d.LastLatencyMs, &d.LastCheckedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

