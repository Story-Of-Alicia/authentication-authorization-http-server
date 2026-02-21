package providers

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	db *sql.DB

	ctx    context.Context
	Cancel context.CancelFunc

	addr string
}

func CreateDb(dsn string) (DB, error) {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return DB{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return DB{
		db:     db,
		addr:   dsn,
		ctx:    ctx,
		Cancel: cancel,
	}, nil
}

func (d *DB) initTables() error {
	ctx, cancel := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancel()

	_, err := d.db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS `sessions` ("+
			"id SERIAL PRIMARY KEY,"+
			"token VARCHAR(64) NOT NULL,"+
			"username VARCHAR(32) NOT NULL,"+
			"expires_at TIMESTAMP);",
	)

	if err != nil {
		return err
	}

	_, err = d.db.ExecContext(ctx,
		"CREATE INDEX session_index ON sessions"+
			"(token, username);")

	if err != nil {
		return err
	}

	_, err = d.db.ExecContext(ctx,
		"CREATE UNIQUE INDEX unique_session_index ON sessions"+
			"(username);")

	if err != nil {
		return err
	}

	return nil
}

func (d *DB) DropTables() error {
	ctx, cancel := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancel()

	_, err := d.db.ExecContext(ctx,
		"DROP TABLE sessions;")

	if err != nil {
		return err
	}

	return nil
}

func (d *DB) CreateSession(username string, token string) (string, error) {
	ctx, cancle := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancle()

	expiriation := time.Now().Add(time.Hour)

	_, err := d.db.ExecContext(ctx,
		"INSERT sessions (username, token, expires_at) VALUES (?, ?, ?)",
		username, token, expiriation)

	if err != nil {
		return "", err
	}

	return token, nil
}

func (d *DB) IsSessionExists(username string) (bool, error) {
	ctx, cancle := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancle()

	var exists bool

	err := d.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM facade WHERE username = ?)",
		username).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (d *DB) UpdateSession(username string, token string) error {
	ctx, cancle := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancle()

	expiration := time.Now().Add(time.Hour)

	_, err := d.db.ExecContext(ctx,
		"UPDATE sessions SET token = ?, expires_at = ? WHERE username = ?", token, expiration, username)

	if err != nil {
		return err
	}

	return nil
}

func (d *DB) DeleteSession(username string) {
	ctx, cancle := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancle()

	_, err := d.db.ExecContext(ctx, "DELETE FROM sessions WHERE username = ?", username)

	if err != nil {
		return
	}
}

func (d *DB) CloseConn() error {
	return d.db.Close()
}
