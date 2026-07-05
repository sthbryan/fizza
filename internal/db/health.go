package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type unixStat_t = unix.Statfs_t

var unixStatfs = unix.Statfs

type MigrationInfo struct {
	Version  int64  `json:"version"`
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
	Applied  bool   `json:"applied"`
	AppliedAt string `json:"applied_at,omitempty"`
}

type DoctorReport struct {
	OK             bool            `json:"ok"`
	Checks         []DoctorCheck   `json:"checks"`
	Integrity      string          `json:"integrity"`
	PendingMigs    int             `json:"pending_migrations"`
	SchemaVersion  int64           `json:"schema_version"`
	DBPath         string          `json:"db_path"`
	DBSize         int64           `json:"db_size"`
	FreeDiskBytes  uint64          `json:"free_disk_bytes,omitempty"`
	Warnings       []string        `json:"warnings,omitempty"`
}

type DoctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

func MigrationStatus(ctx context.Context, conn *sql.DB) (pending []MigrationInfo, applied []MigrationInfo, err error) {
	migs, err := LoadMigrations()
	if err != nil {
		return nil, nil, err
	}
	hashes := map[int64]string{}
	for _, m := range migs {
		sum := sha256.Sum256([]byte(m.Body))
		hashes[m.Version] = hex.EncodeToString(sum[:])
	}
	rows, err := conn.QueryContext(ctx, `SELECT version, applied_at, COALESCE(checksum,'') FROM schema_migrations`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	appliedAt := map[int64]string{}
	stored := map[int64]string{}
	for rows.Next() {
		var v int64
		var ts, cs string
		if err := rows.Scan(&v, &ts, &cs); err != nil {
			return nil, nil, err
		}
		appliedAt[v] = ts
		stored[v] = cs
	}
	for _, m := range migs {
		info := MigrationInfo{
			Version:  m.Version,
			Name:     m.Name,
			Checksum: hashes[m.Version],
		}
		if ts, ok := appliedAt[m.Version]; ok {
			info.Applied = true
			info.AppliedAt = ts
			applied = append(applied, info)
		} else {
			pending = append(pending, info)
		}
	}
	return pending, applied, nil
}

func Doctor(ctx context.Context, conn *sql.DB) (*DoctorReport, error) {
	report := &DoctorReport{OK: true}

	var integrity string
	if err := conn.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		report.OK = false
		report.Checks = append(report.Checks, DoctorCheck{Name: "integrity_check", OK: false, Detail: err.Error()})
	} else {
		ok := integrity == "ok"
		report.OK = report.OK && ok
		report.Integrity = integrity
		report.Checks = append(report.Checks, DoctorCheck{Name: "integrity_check", OK: ok})
	}

	pending, applied, err := MigrationStatus(ctx, conn)
	if err != nil {
		return nil, err
	}
	report.PendingMigs = len(pending)
	if len(applied) > 0 {
		report.SchemaVersion = applied[len(applied)-1].Version
	}
	report.Checks = append(report.Checks, DoctorCheck{
		Name:   "migrations",
		OK:     len(pending) == 0,
		Detail: fmt.Sprintf("%d applied, %d pending", len(applied), len(pending)),
	})
	if len(pending) > 0 {
		report.OK = false
	}

	var quickCheck string
	if err := conn.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&quickCheck); err == nil {
		report.Checks = append(report.Checks, DoctorCheck{Name: "quick_check", OK: quickCheck == "ok"})
	}

	row := conn.QueryRowContext(ctx, `SELECT path FROM pragma_database_list WHERE seq = 0`)
	var dbPath string
	if err := row.Scan(&dbPath); err == nil {
		report.DBPath = dbPath
		if info, err := os.Stat(dbPath); err == nil {
			report.DBSize = info.Size()
			if dir := filepath.Dir(dbPath); dir != "" {
				if free, err := diskFree(dir); err == nil {
					report.FreeDiskBytes = free
				}
			}
		}
	}

	report.Warnings = append(report.Warnings, checkWarnings(report)...)
	return report, nil
}

func checkWarnings(r *DoctorReport) []string {
	var w []string
	if r.DBSize > 100*1024*1024 {
		w = append(w, fmt.Sprintf("database is large (%d bytes); consider vacuum or archiving", r.DBSize))
	}
	if r.PendingMigs > 0 {
		w = append(w, fmt.Sprintf("%d pending migration(s) will be applied on next open", r.PendingMigs))
	}
	return w
}

func diskFree(path string) (uint64, error) {
	var stat unixStat_t
	if err := unixStatfs(path, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}