package facade

import (
	"time"
)

type Session struct {
	Token  string
	User   string
	Expiry time.Time
}

type SessionProvider interface {
	CreateSession(username string) (Session, error)
	DeleteSession(username string) error
}
