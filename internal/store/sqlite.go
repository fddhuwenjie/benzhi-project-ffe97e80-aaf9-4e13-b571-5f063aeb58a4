package store

import (
	"context"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

type SQLiteRepository struct{ db *sql.DB }

func Open(path string) (*SQLiteRepository, error) {
	dsn := path
	if path != ":memory:" {
		dsn = "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	db.SetMaxOpenConns(8)
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	}
	repo := &SQLiteRepository{db: db}
	if err := repo.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepository) Close() error { return r.db.Close() }

func (r *SQLiteRepository) Begin(ctx context.Context) (Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqliteTx{tx: tx}, nil
}

func (r *SQLiteRepository) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS release_cases (id TEXT PRIMARY KEY, language_name TEXT NOT NULL, collection_batch TEXT NOT NULL, owner TEXT NOT NULL, release_level TEXT NOT NULL, status TEXT NOT NULL, revision INTEGER NOT NULL, created_at TEXT NOT NULL, sealed_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS consent_grants (id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES release_cases(id), participant_code TEXT NOT NULL, evidence_ref TEXT NOT NULL, permitted_uses TEXT NOT NULL, region_limits TEXT NOT NULL, valid_until TEXT, withdrawn_at TEXT, UNIQUE(case_id, participant_code))`,
		`CREATE TABLE IF NOT EXISTS recording_assets (id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES release_cases(id), stable_key TEXT NOT NULL, summary TEXT NOT NULL, duration_ms INTEGER NOT NULL, captured_on TEXT NOT NULL, participant_codes TEXT NOT NULL, content_sha256 TEXT NOT NULL, UNIQUE(case_id, stable_key))`,
		`CREATE TABLE IF NOT EXISTS sensitivity_findings (id TEXT PRIMARY KEY, asset_id TEXT NOT NULL REFERENCES recording_assets(id), start_ms INTEGER NOT NULL, end_ms INTEGER NOT NULL, category TEXT NOT NULL, severity TEXT NOT NULL, disposition TEXT NOT NULL, treatment_note TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS steward_reviews (id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES release_cases(id), round INTEGER NOT NULL, reviewer TEXT NOT NULL, decision TEXT NOT NULL, reason_codes TEXT NOT NULL, comment TEXT NOT NULL, decided_at TEXT NOT NULL, finding_count INTEGER NOT NULL DEFAULT 0, UNIQUE(case_id, round))`,
		`CREATE TABLE IF NOT EXISTS release_manifests (id TEXT PRIMARY KEY, case_id TEXT NOT NULL UNIQUE REFERENCES release_cases(id), case_revision INTEGER NOT NULL, asset_entries TEXT NOT NULL, constraint_summary TEXT NOT NULL, canonical_json TEXT NOT NULL, sha256 TEXT NOT NULL, sealed_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS audit_events (sequence INTEGER PRIMARY KEY AUTOINCREMENT, case_id TEXT NOT NULL REFERENCES release_cases(id), actor TEXT NOT NULL, request_id TEXT NOT NULL, before_revision INTEGER NOT NULL, after_revision INTEGER NOT NULL, event_type TEXT NOT NULL, occurred_at TEXT NOT NULL, details TEXT NOT NULL, UNIQUE(case_id, after_revision))`,
		`CREATE TABLE IF NOT EXISTS idempotency_records (request_id TEXT PRIMARY KEY, case_id TEXT NOT NULL, operation TEXT NOT NULL, payload_hash TEXT NOT NULL DEFAULT '', status_code INTEGER NOT NULL, body BLOB NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS idx_cases_status ON release_cases(status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_events_case ON audit_events(case_id, sequence)`,
	}
	for _, statement := range statements {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("迁移 SQLite: %w", err)
		}
	}
	// 兼容已存在的数据库文件。
	var column int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('steward_reviews') WHERE name='finding_count'`).Scan(&column); err == nil && column == 0 {
		if _, err := r.db.ExecContext(ctx, `ALTER TABLE steward_reviews ADD COLUMN finding_count INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('idempotency_records') WHERE name='payload_hash'`).Scan(&column); err == nil && column == 0 {
		if _, err := r.db.ExecContext(ctx, `ALTER TABLE idempotency_records ADD COLUMN payload_hash TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	return nil
}
