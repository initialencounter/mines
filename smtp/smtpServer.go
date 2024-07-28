package smtp

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type Smtp struct {
	SmtpServer string `mapstructure:"smtpServer"`
	Email      string `mapstructure:"email"`
	Password   string `mapstructure:"password"`
	Host       string `mapstructure:"host"`
}

func NewSmtp(SmtpServer string, Email string, Password string) *Smtp {
	var Host, _ = GetSMTPServer(Email)
	return &Smtp{
		SmtpServer,
		Email,
		Password,
		Host,
	}
}

func (s *Smtp) SendEmail(to string, subject string, body string) error {
	// 设置SMTP认证
	auth := smtp.PlainAuth("", s.Email, s.Password, s.Host)

	// 邮件消息
	message := []byte(subject + "\n" + body)

	// 发送邮件
	err := smtp.SendMail(s.SmtpServer, auth, s.Email, []string{to}, message)
	return err
}

// GetSMTPServer 从邮箱地址中提取 SMTP 服务器地址
func GetSMTPServer(email string) (string, error) {
	// 提取域名
	domain := strings.Split(email, "@")[1]

	// 查询 MX 记录
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return "", fmt.Errorf("failed to lookup MX records: %v", err)
	}

	if len(mxRecords) == 0 {
		return "", fmt.Errorf("no MX records found for domain: %s", domain)
	}

	// 返回第一个 MX 记录的地址
	smtpServer := mxRecords[0].Host
	// 去除结尾的点
	if strings.HasSuffix(smtpServer, ".") {
		smtpServer = strings.TrimSuffix(smtpServer, ".")
	}

	return smtpServer, nil
}
