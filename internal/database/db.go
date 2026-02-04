package database

import (
	"context"
	"database/sql"
	"soaauth/internal/config"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	db *sql.DB

	ctx    context.Context
	cancel context.CancelFunc

	addr string
}

func CreateDb() (DB, error) {
	addr := config.GetConfigInstance().DbDSN

	db, err := sql.Open("mysql", addr)

	if err != nil {
		return DB{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return DB{
		db:     db,
		addr:   addr,
		ctx:    ctx,
		cancel: cancel,
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

func (d *DB) CreateSession() error {
	return nil
}
