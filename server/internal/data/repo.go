package data

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

type BindingRepo interface {
	SaveBinding(alias, role, instanceID string) error
	GetBinding(alias, role string) (string, error)
	GetAllBindings() (map[string]map[string]string, error)
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
	_, err := r.db.Exec(`CREATE TABLE IF NOT EXISTS bindings(
		alias TEXT NOT NULL,
		role TEXT NOT NULL,
		instance_id TEXT NOT NULL,
		PRIMARY KEY(alias, role)
	);`)
	return err
}

func (r *SQLiteRepo) SaveBinding(alias, role, instanceID string) error {
	if alias == "" || role == "" || instanceID == "" {
		return errors.New("invalid alias or instanceID")
	}
	_, err := r.db.Exec(`INSERT INTO bindings(alias, role, instance_id) VALUES(?, ?, ?)
		ON CONFLICT(alias, role) DO UPDATE SET instance_id=excluded.instance_id;`, alias, role, instanceID)
	return err
}

func (r *SQLiteRepo) GetBinding(alias, role string) (string, error) {
	var id string
	err := r.db.QueryRow(`SELECT instance_id FROM bindings WHERE alias=? AND role=?;`, alias, role).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("binding not found: %s/%s", alias, role)
	}
	return id, err
}

func (r *SQLiteRepo) GetAllBindings() (map[string]map[string]string, error) {
	rows, err := r.db.Query(`SELECT alias, role, instance_id FROM bindings;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]map[string]string)
	for rows.Next() {
		var a, rle, i string
		if err := rows.Scan(&a, &rle, &i); err != nil {
			return nil, err
		}
		if out[a] == nil {
			out[a] = make(map[string]string)
		}
		out[a][rle] = i
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}
