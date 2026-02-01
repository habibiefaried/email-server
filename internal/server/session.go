package server

import (
	"io"
	"log"

	"github.com/emersion/go-smtp"
)

type Session struct{}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	log.Printf("Mail from: %s\n", from)
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	log.Printf("Rcpt to: %s\n", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	log.Printf("Email body: %s\n", string(body))
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}
