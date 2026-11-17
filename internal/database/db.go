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

func (db *DB) GetDeviceByID(id int) (*Device, error) {
	query := `SELECT id, name, target, check_type, location_cabang, current_status, last_latency_ms, last_checked_at FROM devices WHERE id = ?`
	var d Device
	err := db.QueryRow(query, id).Scan(&d.ID, &d.Name, &d.Target, &d.CheckType, &d.LocationCabang, &d.CurrentStatus, &d.LastLatencyMs, &d.LastCheckedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (db *DB) UpdateDeviceStatus(id int, status string, latencyMs *int64, checkedAt time.Time) error {
	var latencyVal interface{}
	if latencyMs != nil {
		latencyVal = *latencyMs
	}

	query := `UPDATE devices SET current_status = ?, last_latency_ms = ?, last_checked_at = ? WHERE id = ?`
	_, err := db.Exec(query, status, latencyVal, checkedAt, id)
	return err
}

func (db *DB) InsertLatencyRecord(deviceID int, latencyMs int64, recordedAt time.Time) error {
	query := `INSERT INTO latency_records (device_id, latency_ms, recorded_at) VALUES (?, ?, ?)`
	_, err := db.Exec(query, deviceID, latencyMs, recordedAt)
	return err
}

func (db *DB) CreateIncident(deviceID int, startedAt time.Time, note string) (int64, error) {
	query := `INSERT INTO incidents (device_id, started_at, note) VALUES (?, ?, ?)`
	res, err := db.Exec(query, deviceID, startedAt, note)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) ResolveActiveIncident(deviceID int, resolvedAt time.Time) error {
	query := `UPDATE incidents SET resolved_at = ? WHERE device_id = ? AND resolved_at IS NULL`
	_, err := db.Exec(query, resolvedAt, deviceID)
	return err
}

func (db *DB) GetIncidents(limit int) ([]Incident, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
	SELECT i.id, i.device_id, d.name, d.location_cabang, i.started_at, i.resolved_at, i.note
	FROM incidents i
	JOIN devices d ON i.device_id = d.id
	ORDER BY i.started_at DESC LIMIT ?
	`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var inc Incident
		if err := rows.Scan(&inc.ID, &inc.DeviceID, &inc.DeviceName, &inc.Cabang, &inc.StartedAt, &inc.ResolvedAt, &inc.Note); err != nil {
			return nil, err
		}
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

func (db *DB) SeedDevicesIfEmpty() error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if err != nil || count > 0 {
		return err
	}

	seeds := []struct {
		name, target, checkType, cabang string
	}{
		{"Cabang Jakarta Gateway", "127.0.0.1:8080", "http", "Jakarta HQ"},
		{"Printer Surat Jalan Bandung", "127.0.0.1:9100", "ping", "Bandung"},
		{"Server Absensi Surabaya", "127.0.0.1:8081", "http", "Surabaya"},
		{"Local Database PPh21", "127.0.0.1:5432", "ping", "Jakarta HQ"},
		{"Barcode Scanner Gateway Medan", "127.0.0.1:9200", "ping", "Medan"},
	}

	now := time.Now()
	for _, s := range seeds {
		_, err := db.Exec(
			"INSERT INTO devices (name, target, check_type, location_cabang, current_status, last_latency_ms, last_checked_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			s.name, s.target, s.checkType, s.cabang, "UNKNOWN", nil, now,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
