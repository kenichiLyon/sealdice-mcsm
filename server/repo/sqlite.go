package repo

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

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
	_, err := r.db.Exec(`CREATE TABLE IF NOT EXISTS bindings(
		alias TEXT PRIMARY KEY,
		instance_id TEXT NOT NULL UNIQUE
	);`)
	return err
}

func (r *SQLiteRepo) SaveBinding(alias, instanceID string) error {
	if alias == "" || instanceID == "" {
		return errors.New("invalid alias or instanceID")
	}
	_, err := r.db.Exec(`INSERT INTO bindings(alias, instance_id) VALUES(?, ?)
		ON CONFLICT(alias) DO UPDATE SET instance_id=excluded.instance_id;`, alias, instanceID)
	return err
}

func (r *SQLiteRepo) GetBinding(alias string) (string, error) {
	var id string
	err := r.db.QueryRow(`SELECT instance_id FROM bindings WHERE alias=?;`, alias).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("alias not found: %s", alias)
	}
	return id, err
}

func (r *SQLiteRepo) GetAllBindings() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT alias, instance_id FROM bindings;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var a, i string
		if err := rows.Scan(&a, &i); err != nil {
			return nil, err
		}
		out[a] = i
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}

