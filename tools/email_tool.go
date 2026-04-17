package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// EmailSender is the contract for sending transactional emails.
type EmailSender interface {
	SendPasswordReset(toEmail, userName, resetLink string) error
	Send(ctx context.Context, to, subject, htmlBody, textBody string) error
}

// ─────────────────────────────────────────────────────────────────────────────
// LoggerEmailSender — dev/test (no real email sent)
// ─────────────────────────────────────────────────────────────────────────────

type LoggerEmailSender struct{}

func (l *LoggerEmailSender) SendPasswordReset(toEmail, userName, resetLink string) error {
	log.Info().
		Str("to", toEmail).
		Str("user", userName).
		Str("reset_link", resetLink).
		Msg("[DEV EMAIL] password reset link — copia este link al navegador para testear")
	return nil
}

func (l *LoggerEmailSender) Send(_ context.Context, to, subject, _, textBody string) error {
	log.Info().
		Str("to", to).
		Str("subject", subject).
		Str("body", textBody).
		Msg("[DEV EMAIL] send")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ResendEmailSender — production (Resend API)
// ─────────────────────────────────────────────────────────────────────────────

type ResendEmailSender struct {
	APIKey   string
	FromAddr string // e.g. "noreply@estock.app"
	AppName  string // e.g. "eSTOCK"
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text"`
}

// RateLimitError is returned when Resend responds with HTTP 429.
type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string { return e.Message }

func (r *ResendEmailSender) Send(ctx context.Context, to, subject, htmlBody, textBody string) error {
	from := fmt.Sprintf("%s <%s>", r.AppName, r.FromAddr)
	payload := resendPayload{
		From:    from,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
		Text:    textBody,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("resend: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("resend: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{Message: "resend: rate limit exceeded (429)"}
	}
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend: HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (r *ResendEmailSender) SendPasswordReset(toEmail, userName, resetLink string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	html := renderResetEmailHTML(userName, resetLink, r.AppName)
	text := fmt.Sprintf("Hola %s,\n\nRestablece tu contraseña de %s: %s\n\nEl enlace expira en 1 hora.", userName, r.AppName, resetLink)
	return r.Send(ctx, toEmail, fmt.Sprintf("Restablece tu contraseña de %s", r.AppName), html, text)
}

// ─────────────────────────────────────────────────────────────────────────────
// Email templates
// ─────────────────────────────────────────────────────────────────────────────

// RenderNotificationEmail returns (htmlBody, textBody) for a given event type.
// Falls back to a generic template for unknown event types.
func RenderNotificationEmail(eventType, title, body string) (html, text string) {
	switch eventType {
	case "task_assigned":
		return renderTaskAssignedHTML(title, body), fmt.Sprintf("%s\n\n%s", title, body)
	case "task_completed":
		return renderGenericHTML(title, body), fmt.Sprintf("%s\n\n%s", title, body)
	case "lot_expiring_7d":
		return renderLotExpiringHTML(title, body), fmt.Sprintf("%s\n\n%s", title, body)
	case "lot_expiring_1d":
		return renderLotExpiringHTML(title, body), fmt.Sprintf("%s\n\n%s", title, body)
	case "low_stock":
		return renderLowStockHTML(title, body), fmt.Sprintf("%s\n\n%s", title, body)
	case "user_welcome":
		return renderUserWelcomeHTML(title, body), fmt.Sprintf("%s\n\n%s", title, body)
	default:
		return renderGenericHTML(title, body), fmt.Sprintf("%s\n\n%s", title, body)
	}
}

func renderResetEmailHTML(userName, resetLink, appName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:24px;">Restablece tu contraseña</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">
      Hola %s,<br><br>Recibimos una solicitud para restablecer la contraseña de tu cuenta de %s.
      Haz clic en el botón para crear una nueva contraseña. El enlace expira en <strong>1 hora</strong>.
    </p>
    <a href="%s" style="display:inline-block;background:#203173;color:#e8d833;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;">Restablecer contraseña</a>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">Si no solicitaste este cambio, puedes ignorar este correo.</p>
  </div>
</body></html>`, userName, appName, resetLink)
}

func renderTaskAssignedHTML(title, body string) string {
	return renderGenericHTML(title, body)
}

func renderLotExpiringHTML(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <div style="background:#FEF3C7;border-left:4px solid #F59E0B;padding:12px 16px;border-radius:4px;margin-bottom:24px;">
      <strong style="color:#92400E;">⚠ Alerta de vencimiento</strong>
    </div>
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:22px;">%s</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">%s</p>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">eSTOCK — Sistema de gestión de inventario</p>
  </div>
</body></html>`, title, body)
}

func renderLowStockHTML(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <div style="background:#FEE2E2;border-left:4px solid #EF4444;padding:12px 16px;border-radius:4px;margin-bottom:24px;">
      <strong style="color:#991B1B;">🔴 Stock bajo</strong>
    </div>
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:22px;">%s</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">%s</p>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">eSTOCK — Sistema de gestión de inventario</p>
  </div>
</body></html>`, title, body)
}

func renderUserWelcomeHTML(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:24px;">¡Bienvenido a eSTOCK!</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">%s</p>
    <pre style="background:#F8FAFC;border:1px solid #E2E8F0;border-radius:8px;padding:16px;font-size:14px;color:#334155;">%s</pre>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">eSTOCK — Sistema de gestión de inventario</p>
  </div>
</body></html>`, title, body)
}

func renderGenericHTML(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:22px;">%s</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">%s</p>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">eSTOCK — Sistema de gestión de inventario</p>
  </div>
</body></html>`, title, body)
}
