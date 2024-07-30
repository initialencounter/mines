package smtp

import (
	"fmt"
	"net/smtp"
)

type MailConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	TLS      bool   `mapstructure:"tls"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type SMTP struct {
	host string
	port string
	auth smtp.Auth
	from string
}

func NewSMTP(config MailConfig) *SMTP {
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	return &SMTP{
		host: config.Host,
		port: config.Port,
		auth: auth,
		from: config.Username,
	}

}

type SendOptions struct {
	To      []string
	Subject string
	Body    string
}

func (s *SMTP) SendEmail(options SendOptions) (string, error) {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	msg := []byte("From: " + s.from + "\r\n" +
		"To: " + options.To[0] + "\r\n" +
		"Subject: " + options.Subject + "\r\n" +
		"\r\n" +
		options.Body)

	err := smtp.SendMail(addr, s.auth, s.from, options.To, msg)
	if err != nil {
		return "", err
	}
	return "Message Sent", nil
}
