package server

import "github.com/emersion/go-smtp"

type Backend struct{}

func (bkd *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}
