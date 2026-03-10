package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
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
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO logs (message, group_id, timestamp) VALUES (?, ?, ?)`,
		message, groupID, time.Now().UTC().Format(time.RFC3339),
	)
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

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO unauthorized_logs (ip, reason, headers, timestamp) VALUES (?, ?, ?, ?)`,
		ip, reason, string(headersJSON), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert unauthorized log: %w", err)
	}

	return nil
}

// StartRetention runs a periodic purge, deleting log entries older than maxAge.
// It blocks until ctx is canceled, so call it in a goroutine.
func (s *LogStore) StartRetention(ctx context.Context, maxAge time.Duration) {
	if s == nil || s.db == nil {
		return
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once immediately on startup.
	s.purgeOlderThan(ctx, maxAge)

	for {
		select {
		case <-ticker.C:
			s.purgeOlderThan(ctx, maxAge)
		case <-ctx.Done():
			return
		}
	}
}

func (s *LogStore) purgeOlderThan(ctx context.Context, maxAge time.Duration) {
	cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339)

	if _, err := s.db.ExecContext(ctx, `DELETE FROM logs WHERE timestamp < ?`, cutoff); err != nil {
		log.Warn().Err(err).Msg("failed to purge old send logs")
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM unauthorized_logs WHERE timestamp < ?`, cutoff); err != nil {
		log.Warn().Err(err).Msg("failed to purge old unauthorized logs")
	}
}

func (s *LogStore) initSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS logs (
id        INTEGER PRIMARY KEY AUTOINCREMENT,
message   TEXT,
group_id  TEXT,
timestamp TEXT
)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp)`,
		`CREATE TABLE IF NOT EXISTS unauthorized_logs (
id        INTEGER PRIMARY KEY AUTOINCREMENT,
ip        TEXT,
reason    TEXT,
headers   TEXT,
timestamp TEXT
)`,
		`CREATE INDEX IF NOT EXISTS idx_unauthorized_logs_ip        ON unauthorized_logs(ip)`,
		`CREATE INDEX IF NOT EXISTS idx_unauthorized_logs_timestamp ON unauthorized_logs(timestamp)`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("init logs schema: %w", err)
		}
	}

	return nil
}
