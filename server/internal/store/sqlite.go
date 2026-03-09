// Package store provides SQLite storage for simplified deployment.
// This mode eliminates the need for PostgreSQL, Redis, and MinIO.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// SQLiteStore provides a single-file storage backend for AI-Trace.
// Suitable for development, testing, and small deployments.
type SQLiteStore struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[string][]byte // Simple in-memory cache
}

// NewSQLiteStore creates a new SQLite-based store.
// dbPath can be ":memory:" for in-memory or a file path like "./ai-trace.db"
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL: %w", err)
	}

	store := &SQLiteStore{
		db:    db,
		cache: make(map[string][]byte),
	}

	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	schema := `
	-- Traces table
	CREATE TABLE IF NOT EXISTS traces (
		trace_id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL DEFAULT 'default',
		status TEXT NOT NULL DEFAULT 'active',
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Events table
	CREATE TABLE IF NOT EXISTS events (
		event_id TEXT PRIMARY KEY,
		trace_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		sequence_num INTEGER NOT NULL,
		payload_hash TEXT NOT NULL,
		payload TEXT,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (trace_id) REFERENCES traces(trace_id)
	);

	-- Certificates table
	CREATE TABLE IF NOT EXISTS certificates (
		cert_id TEXT PRIMARY KEY,
		trace_id TEXT NOT NULL,
		merkle_root TEXT NOT NULL,
		evidence_level TEXT NOT NULL DEFAULT 'internal',
		signature TEXT,
		event_count INTEGER NOT NULL,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (trace_id) REFERENCES traces(trace_id)
	);

	-- API Keys table
	CREATE TABLE IF NOT EXISTS api_keys (
		key_id TEXT PRIMARY KEY,
		key_hash TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT 'default',
		scopes TEXT,
		is_active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);
	CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
	CREATE INDEX IF NOT EXISTS idx_certs_trace_id ON certificates(trace_id);
	CREATE INDEX IF NOT EXISTS idx_traces_tenant_id ON traces(tenant_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Trace operations

type Trace struct {
	TraceID   string          `json:"trace_id"`
	TenantID  string          `json:"tenant_id"`
	Status    string          `json:"status"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (s *SQLiteStore) CreateTrace(ctx context.Context, trace *Trace) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO traces (trace_id, tenant_id, status, metadata) VALUES (?, ?, ?, ?)`,
		trace.TraceID, trace.TenantID, trace.Status, trace.Metadata,
	)
	return err
}

func (s *SQLiteStore) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	var trace Trace
	err := s.db.QueryRowContext(ctx,
		`SELECT trace_id, tenant_id, status, metadata, created_at, updated_at FROM traces WHERE trace_id = ?`,
		traceID,
	).Scan(&trace.TraceID, &trace.TenantID, &trace.Status, &trace.Metadata, &trace.CreatedAt, &trace.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &trace, err
}

// Event operations

type Event struct {
	EventID     string          `json:"event_id"`
	TraceID     string          `json:"trace_id"`
	EventType   string          `json:"event_type"`
	SequenceNum int             `json:"sequence_num"`
	PayloadHash string          `json:"payload_hash"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (s *SQLiteStore) CreateEvent(ctx context.Context, event *Event) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (event_id, trace_id, event_type, sequence_num, payload_hash, payload, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.EventID, event.TraceID, event.EventType, event.SequenceNum,
		event.PayloadHash, event.Payload, event.Metadata,
	)
	return err
}

func (s *SQLiteStore) GetEventsByTrace(ctx context.Context, traceID string) ([]*Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT event_id, trace_id, event_type, sequence_num, payload_hash, payload, metadata, created_at
		 FROM events WHERE trace_id = ? ORDER BY sequence_num`,
		traceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.EventID, &e.TraceID, &e.EventType, &e.SequenceNum,
			&e.PayloadHash, &e.Payload, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// Certificate operations

type Certificate struct {
	CertID        string          `json:"cert_id"`
	TraceID       string          `json:"trace_id"`
	MerkleRoot    string          `json:"merkle_root"`
	EvidenceLevel string          `json:"evidence_level"`
	Signature     string          `json:"signature,omitempty"`
	EventCount    int             `json:"event_count"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

func (s *SQLiteStore) CreateCertificate(ctx context.Context, cert *Certificate) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO certificates (cert_id, trace_id, merkle_root, evidence_level, signature, event_count, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		cert.CertID, cert.TraceID, cert.MerkleRoot, cert.EvidenceLevel,
		cert.Signature, cert.EventCount, cert.Metadata,
	)
	return err
}

func (s *SQLiteStore) GetCertificate(ctx context.Context, certID string) (*Certificate, error) {
	var cert Certificate
	err := s.db.QueryRowContext(ctx,
		`SELECT cert_id, trace_id, merkle_root, evidence_level, signature, event_count, metadata, created_at
		 FROM certificates WHERE cert_id = ?`,
		certID,
	).Scan(&cert.CertID, &cert.TraceID, &cert.MerkleRoot, &cert.EvidenceLevel,
		&cert.Signature, &cert.EventCount, &cert.Metadata, &cert.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &cert, err
}

func (s *SQLiteStore) GetCertificatesByTrace(ctx context.Context, traceID string) ([]*Certificate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT cert_id, trace_id, merkle_root, evidence_level, signature, event_count, metadata, created_at
		 FROM certificates WHERE trace_id = ? ORDER BY created_at DESC`,
		traceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*Certificate
	for rows.Next() {
		var c Certificate
		if err := rows.Scan(&c.CertID, &c.TraceID, &c.MerkleRoot, &c.EvidenceLevel,
			&c.Signature, &c.EventCount, &c.Metadata, &c.CreatedAt); err != nil {
			return nil, err
		}
		certs = append(certs, &c)
	}
	return certs, rows.Err()
}

// Cache operations (simple in-memory cache to replace Redis)

func (s *SQLiteStore) CacheSet(key string, value []byte, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = value
	// Note: TTL not implemented in simple version
}

func (s *SQLiteStore) CacheGet(key string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.cache[key]
	return v, ok
}

func (s *SQLiteStore) CacheDelete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, key)
}

// Stats

type Stats struct {
	TotalTraces       int64 `json:"total_traces"`
	TotalEvents       int64 `json:"total_events"`
	TotalCertificates int64 `json:"total_certificates"`
}

func (s *SQLiteStore) GetStats(ctx context.Context) (*Stats, error) {
	var stats Stats

	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM traces").Scan(&stats.TotalTraces)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&stats.TotalEvents)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM certificates").Scan(&stats.TotalCertificates)

	return &stats, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
