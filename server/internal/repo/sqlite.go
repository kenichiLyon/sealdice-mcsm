package repo

import (
	"database/sql"
	"fmt"
	"sealdice-mcsm/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRepo struct {
	db *sql.DB
}

func NewSQLiteRepo(dbPath string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &SQLiteRepo{db: db}
	if err := repo.init(); err != nil {
		db.Close()
		return nil, err
	}

	return repo, nil
}

func (r *SQLiteRepo) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS bindings (
		alias TEXT PRIMARY KEY,
		instance_id TEXT NOT NULL
	);
	`
	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create bindings table: %w", err)
	}
	return nil
}

func (r *SQLiteRepo) SaveBinding(alias, instanceID string) error {
	query := `INSERT OR REPLACE INTO bindings (alias, instance_id) VALUES (?, ?)`
	_, err := r.db.Exec(query, alias, instanceID)
	if err != nil {
		return fmt.Errorf("failed to save binding: %w", err)
	}
	return nil
}

func (r *SQLiteRepo) GetBinding(alias string) (*model.Binding, error) {
	query := `SELECT instance_id FROM bindings WHERE alias = ?`
	row := r.db.QueryRow(query, alias)

	var instanceID string
	err := row.Scan(&instanceID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get binding: %w", err)
	}

	return &model.Binding{
		Alias:      alias,
		InstanceID: instanceID,
	}, nil
}

func (r *SQLiteRepo) GetAllBindings() ([]model.Binding, error) {
	query := `SELECT alias, instance_id FROM bindings`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list bindings: %w", err)
	}
	defer rows.Close()

	var bindings []model.Binding
	for rows.Next() {
		var b model.Binding
		if err := rows.Scan(&b.Alias, &b.InstanceID); err != nil {
			return nil, fmt.Errorf("failed to scan binding: %w", err)
		}
		bindings = append(bindings, b)
	}
	return bindings, nil
}

func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}
