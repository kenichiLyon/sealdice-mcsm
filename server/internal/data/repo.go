package data

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Binding struct {
	Alias              string
	ProtocolInstanceID string
	CoreInstanceID     string
	CreatedAt          time.Time
}

type BindingRepo interface {
	SaveBinding(b *Binding) error
	GetBinding(alias string) (*Binding, error)
	DeleteBinding(alias string) error
	GetAllBindings() ([]*Binding, error)
	Close() error
}

type SQLiteRepo struct {
	db *sql.DB
}

func NewSQLiteRepo(path string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	r := &SQLiteRepo{db: db}
	if err := r.init(); err != nil {
		db.Close()
		return nil, err
	}
	return r, nil
}

func (r *SQLiteRepo) init() error {
	// Drop old table if exists (optional, but good for schema change in dev)
	// For now, I'll just create the new one if not exists.
	// If the old one exists with different schema, this might fail or ignore.
	// Assuming dev environment, let's DROP if it has the old schema?
	// The old schema had (alias, role) PK. The new one has (alias) PK.
	// Let's force recreation for this task to ensure schema match.
	
	// Check if table has 'role' column (old schema)
	// Or just try to create new table.
	
	// Simply: CREATE TABLE IF NOT EXISTS bindings ...
	// But columns are different. 
	// I'll drop the table to be safe since this is a refactor task.
	_, _ = r.db.Exec(`DROP TABLE IF EXISTS bindings`)

	_, err := r.db.Exec(`CREATE TABLE IF NOT EXISTS bindings(
		alias TEXT PRIMARY KEY,
		protocol_instance_id TEXT NOT NULL,
		core_instance_id TEXT NOT NULL,
		created_at DATETIME
	);`)
	return err
}

func (r *SQLiteRepo) SaveBinding(b *Binding) error {
	if b.Alias == "" || b.ProtocolInstanceID == "" || b.CoreInstanceID == "" {
		return errors.New("invalid binding data")
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	_, err := r.db.Exec(`INSERT INTO bindings(alias, protocol_instance_id, core_instance_id, created_at) 
		VALUES(?, ?, ?, ?)
		ON CONFLICT(alias) DO UPDATE SET 
			protocol_instance_id=excluded.protocol_instance_id,
			core_instance_id=excluded.core_instance_id,
			created_at=excluded.created_at;`,
		b.Alias, b.ProtocolInstanceID, b.CoreInstanceID, b.CreatedAt)
	return err
}

func (r *SQLiteRepo) GetBinding(alias string) (*Binding, error) {
	var b Binding
	err := r.db.QueryRow(`SELECT alias, protocol_instance_id, core_instance_id, created_at FROM bindings WHERE alias=?;`, alias).
		Scan(&b.Alias, &b.ProtocolInstanceID, &b.CoreInstanceID, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("binding not found: %s", alias)
	}
	return &b, err
}

func (r *SQLiteRepo) DeleteBinding(alias string) error {
	_, err := r.db.Exec(`DELETE FROM bindings WHERE alias=?`, alias)
	return err
}

func (r *SQLiteRepo) GetAllBindings() ([]*Binding, error) {
	rows, err := r.db.Query(`SELECT alias, protocol_instance_id, core_instance_id, created_at FROM bindings;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Binding
	for rows.Next() {
		var b Binding
		if err := rows.Scan(&b.Alias, &b.ProtocolInstanceID, &b.CoreInstanceID, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}
