package email

import (
	"kama-chat-server/internal/config"
	"kama-chat-server/internal/service/redis"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/util/random"
	"net/smtp"
	"strconv"
	"time"
	"github.com/jordan-wright/email"
)

// ============================================================
// SendVerificationCode - 发送邮箱验证码
// ============================================================
func SendVerificationCode(toEmail string) (string, int) {
	key := "email_code_" + toEmail
	code, err := redis.GetKey(key)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}


	if code != "" {
		return "验证码已发送, 请检查邮箱或5分钟后重试", -2
	}

	codeInt, err := random.GetRandomInt(6)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}
	code = strconv.Itoa(codeInt)

	err = redis.SetKeyEx(key, code, 5*time.Minute)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}

	err = sendEmail(toEmail, code)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}

	return "验证码已发送至 " + toEmail, 0
}

// ============================================================
// VerifyCode - 验证验证码
// ============================================================
func VerifyCode(email string, inputCode string) (string, int) {
	key := "email_code_" + email

	storedCode, err := redis.GetKey(key)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}

	if storedCode == "" {
		return "验证码已过期，请重新获取", -2
	}

	if storedCode != inputCode {
		return "验证码错误", -2
	}
	// 验证成功， 删除验证码
	redis.DelKeyIfExists(key)
	return "验证成功", 0
}

// ============================================================
// sendEmail - 发送邮件（内部函数）
// ============================================================
func sendEmail(toEmail string, code string) error {
	conf := config.GetConfig()

	e := email.NewEmail()
    e.From = conf.EmailConfig.FromName + " <" + conf.EmailConfig.SmtpUsername + ">"
    e.To = []string{toEmail}
    e.Subject = "【KamaChat】邮箱验证码"
    e.HTML = []byte(buildEmailHTML(code))

    // 发送邮件
    auth := smtp.PlainAuth("", conf.EmailConfig.SmtpUsername, conf.EmailConfig.SmtpPassword, conf.EmailConfig.SmtpHost)
    return e.Send(
        conf.EmailConfig.SmtpHost+":"+strconv.Itoa(conf.EmailConfig.SmtpPort),
        auth,
    )
}

// ============================================================
// buildEmailHTML - 构建邮件HTML内容
// ============================================================
func buildEmailHTML(code string) string {
    return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; background-color: #f5f5f5; padding: 20px;">
    <div style="max-width: 500px; margin: 0 auto; background: white; border-radius: 8px; padding: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
        <h2 style="color: #333; margin-bottom: 20px;">邮箱验证码</h2>
        <p style="color: #666; font-size: 14px;">您好，您正在验证邮箱，验证码如下：</p>
        <div style="background: #f8f9fa; border-radius: 4px; padding: 20px; text-align: center; margin: 20px 0;">
            <span style="font-size: 32px; font-weight: bold; color: #007bff; letter-spacing: 8px;">` + code + `</span>
        </div>
        <p style="color: #999; font-size: 12px;">验证码有效期5分钟，请勿泄露给他人。</p>
        <p style="color: #999; font-size: 12px; margin-top: 30px;">此邮件由系统自动发送，请勿回复。</p>
    </div>
</body>
</html>
`
}






