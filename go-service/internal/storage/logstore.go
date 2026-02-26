package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type LogStore struct {
	db *sql.DB
}

func NewLogStore(ctx context.Context, dsn string) (*LogStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open logs db: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping logs db: %w", err)
	}

	store := &LogStore{db: db}
	if err := store.initSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *LogStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *LogStore) InsertSend(ctx context.Context, message, groupID string) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO logs (message, group_id, timestamp) VALUES (?, ?, ?)`, message, groupID, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert send log: %w", err)
	}
	return nil
}

func (s *LogStore) InsertUnauthorized(ctx context.Context, ip, reason string, headers map[string][]string) error {
	if s == nil || s.db == nil {
		return nil
	}

	headersJSON, err := json.Marshal(headers)
	if err != nil {
		return fmt.Errorf("marshal unauthorized headers: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO unauthorized_logs (ip, reason, headers, timestamp) VALUES (?, ?, ?, ?)`, ip, reason, string(headersJSON), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert unauthorized log: %w", err)
	}

	return nil
}

func (s *LogStore) initSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message TEXT,
			group_id TEXT,
			timestamp TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS unauthorized_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT,
			reason TEXT,
			headers TEXT,
			timestamp TEXT
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("init logs schema: %w", err)
		}
	}

	return nil
}
