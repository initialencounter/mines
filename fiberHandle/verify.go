package fiberHandle

import (
	"fmt"
	emailverifier "github.com/AfterShip/email-verifier"
	"github.com/gofiber/fiber/v2"
	"main/database"
	"main/smtp"
	"main/utils"
	"regexp"
)

var (
	emailVerifier = emailverifier.NewVerifier()
)

type ForgotPasswordRequest struct {
	User  string `json:"user"`
	Email string `json:"email"`
}

func VerifyCode(handler *database.DBHandler, c *fiber.Ctx, config smtp.MailConfig, codeCache *utils.CodeCache) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}

	userName := req.User
	email := req.Email

	if res, _ := handler.NameExists(userName); !res {
		return fiber.NewError(fiber.StatusUnauthorized, "User not found")
	}

	getEmail, err := handler.GetEmail(userName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get email")
	}
	if getEmail != email {
		return c.Status(fiber.StatusBadRequest).SendString("Email does not match")
	}

	code, err := utils.GenerateCode()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to generate code")
	}

	codeCache.Set(userName, code.Code, code.CreationTime)
	var conn = smtp.NewSMTP(config)
	var sendOptions = smtp.SendOptions{
		To:      []string{email},
		Subject: "[MineSweeper] Reset Password",
		Body:    fmt.Sprintf("您好，\n\n您已要求重置与此电子邮件地址 (%s) 关联的 Minesweeper 帐户的密码。\n\n密码重置代码：\n\n%s\n\n此密码更改代码自发送此电子邮件起 2 小时后失效。\n\n如果您没有发起此请求，请忽略此电子邮件。\n\n回复此电子邮件不会有人查看或答复。\n\n谢谢，\n\nMinesweeper Server\n\n\n\n\n***这是自动通知。回复此电子邮件不会有人查看或答复。", email, code.Code),
	}
	email, err = conn.SendEmail(sendOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to send email")
	}
	return c.Status(fiber.StatusOK).SendString("Email sent")
}

func VerifyEmail(email string) (bool, string) {
	ret, err := emailVerifier.Verify(email)
	if err != nil {
		return false, "verify email address failed, error is: " + err.Error()
	}
	if !ret.Syntax.Valid {
		return false, "email address syntax is invalid"
	}
	return true, ""
}

func VerifyUserName(username string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return !re.MatchString(username)
}
