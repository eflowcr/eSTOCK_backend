package tools

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// EmailSender is the contract for sending transactional emails.
// In S1, only LoggerEmailSender is active. ResendEmailSender is a stub for production.
type EmailSender interface {
	SendPasswordReset(toEmail, userName, resetLink string) error
}

// LoggerEmailSender logs emails to stdout (dev/test). No real email is sent.
type LoggerEmailSender struct{}

func (l *LoggerEmailSender) SendPasswordReset(toEmail, userName, resetLink string) error {
	log.Info().
		Str("to", toEmail).
		Str("user", userName).
		Str("reset_link", resetLink).
		Msg("[DEV EMAIL] password reset link — copia este link al navegador para testear")
	return nil
}

// ResendEmailSender uses the Resend API (production). Activate in S1 prod / S2.
// Requires: go get github.com/resend/resend-go/v2
type ResendEmailSender struct {
	APIKey   string
	FromAddr string // e.g. "noreply@eprac.com"
	AppName  string // e.g. "eSTOCK"
}

func (r *ResendEmailSender) SendPasswordReset(toEmail, userName, resetLink string) error {
	// TODO (S1 prod / S2): descomentar cuando se agregue resend-go al go.mod
	//
	// client := resend.NewClient(r.APIKey)
	// _, err := client.Emails.Send(&resend.SendEmailRequest{
	//     From:    fmt.Sprintf("%s <%s>", r.AppName, r.FromAddr),
	//     To:      []string{toEmail},
	//     Subject: fmt.Sprintf("Restablece tu contraseña de %s", r.AppName),
	//     Html:    renderResetEmailHTML(userName, resetLink, r.AppName),
	// })
	// return err
	return fmt.Errorf("resend not configured — falling back to logger")
}

// renderResetEmailHTML generates the password reset email HTML with ePRAC brand tokens (navy/gold).
func renderResetEmailHTML(userName, resetLink, appName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:24px;">
      Restablece tu contraseña
    </h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">
      Hola %s,<br><br>
      Recibimos una solicitud para restablecer la contraseña de tu cuenta de %s.
      Haz clic en el botón para crear una nueva contraseña. El enlace expira en <strong>1 hora</strong>.
    </p>
    <a href="%s" style="display:inline-block;background:#203173;color:#e8d833;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;">
      Restablecer contraseña
    </a>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">
      Si no solicitaste este cambio, puedes ignorar este correo — tu contraseña seguirá siendo la misma.
    </p>
  </div>
</body></html>`, userName, appName, resetLink)
}
