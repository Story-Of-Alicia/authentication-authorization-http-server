package providers

import (
	"authentication-server/internal"
	"authentication-server/internal/facade"
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

const TokenLength = 64

type PostgresSessionProvider struct {
	db  *sql.DB
	Ctx context.Context
	DSN string
}

func (p *PostgresSessionProvider) CreateSession(username string) (facade.Session, error) {
	token, err := internal.GenerateSessionToken(TokenLength)
	if err != nil {
		log.Fatal("failed to generate session token")
		return facade.Session{}, err
	}

	p.sessionCreate(username, token, time.Now().Add(10*time.Minute))

	return facade.Session{
		Token: token, User: username, Expiry: time.Now(),
	}, nil
}

func (p *PostgresSessionProvider) DeleteSession(username string) error {
	log.Println("Deleting session")
	return nil
}

func (p *PostgresSessionProvider) Init() error {
	db, err := sql.Open("postgres", p.DSN)
	if err != nil {
		return err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	context.AfterFunc(p.Ctx, func() {
		db.Close()
	})

	p.db = db

	err = p.initTables()
	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresSessionProvider) initTables() error {
	ctx, cancel := context.WithTimeout(p.Ctx, 5*time.Second)
	defer cancel()

	_, err := p.db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS sessions ("+
			"username VARCHAR(32) NOT NULL PRIMARY KEY,"+
			"token VARCHAR(64) NOT NULL,"+
			"expires_at TIMESTAMP);",
	)

	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresSessionProvider) sessionCreate(username string, token string, expiry time.Time) (string, error) {
	ctx, cancel := context.WithTimeout(p.Ctx, 5*time.Second)
	defer cancel()

	_, err := p.db.ExecContext(ctx,
		"INSERT INTO sessions (username, token, expires_at) VALUES ($1, $2, $3) ON CONFLICT (username) DO UPDATE SET token=$2, expires_at=$3",
		username, token, expiry)

	if err != nil {
		return "", err
	}

	return token, nil
}
