package providers

import (
	"authentication-server/internal/facade"
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type PostgresSessionProvider struct {
	db     *sql.DB
	ctx    context.Context
	cancel context.CancelFunc

	DSN string
}

func (p *PostgresSessionProvider) CreateSession(username string) (facade.Session, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PostgresSessionProvider) DeleteSession(username string) error {
	//TODO implement me
	panic("implement me")
}

func (p *PostgresSessionProvider) Init() error {
	db, err := sql.Open("mysql", p.DSN)
	if err != nil {
		return err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	p.ctx, p.cancel = context.WithCancel(context.Background())
	return nil
}

func (p *PostgresSessionProvider) initTables() error {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	_, err := p.db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS `sessions` ("+
			"username VARCHAR(32) NOT NULL PRIMARY KEY,"+
			"token VARCHAR(64) NOT NULL,"+
			"expires_at TIMESTAMP);",
	)

	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresSessionProvider) sessionCreate(username string, token string) (string, error) {
	ctx, cancle := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancle()

	expiriation := time.Now().Add(time.Hour)

	_, err := p.db.ExecContext(ctx,
		"INSERT sessions (username, token, expires_at) VALUES (?, ?, ?)",
		username, token, expiriation)

	if err != nil {
		return "", err
	}

	return token, nil
}

func (p *PostgresSessionProvider) sessionExists(username string) (bool, error) {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	var exists bool

	err := p.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM facade WHERE username = ?)",
		username).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (p *PostgresSessionProvider) sessionUpdate(username string, token string) error {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	expiration := time.Now().Add(time.Hour)

	_, err := p.db.ExecContext(ctx,
		"UPDATE sessions SET token = ?, expires_at = ? WHERE username = ?", token, expiration, username)

	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresSessionProvider) sessionDelete(username string) {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	_, err := p.db.ExecContext(ctx, "DELETE FROM sessions WHERE username = ?", username)

	if err != nil {
		return
	}
}
