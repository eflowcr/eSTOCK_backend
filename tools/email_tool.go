package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/http"
	"net/smtp"
	"net/textproto"
	"strings"
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
	htmlBody := renderResetEmailHTML(userName, resetLink, r.AppName)
	text := fmt.Sprintf("Hola %s,\n\nRestablece tu contraseña de %s: %s\n\nEl enlace expira en 1 hora.", userName, r.AppName, resetLink)
	return r.Send(ctx, toEmail, fmt.Sprintf("Restablece tu contraseña de %s", r.AppName), htmlBody, text)
}

// ─────────────────────────────────────────────────────────────────────────────
// GatewayEmailSender — routes transactional emails via VPS Manager API (S-EM2)
// ─────────────────────────────────────────────────────────────────────────────

type GatewayEmailSender struct {
	BaseURL  string
	APIKey   string
	FromAddr string
	AppName  string
	client   *http.Client
}

type gatewayEmailRequest struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html,omitempty"`
	BodyText string `json:"body_text,omitempty"`
}

// NewGatewayEmailSender constructs a sender that posts to {baseURL}/emails/send.
// Trailing slashes on baseURL are normalized so callers can pass either
// "https://host/api/v1" or "https://host/api/v1/" without producing a double slash
// in the request URL (HR-W3-B7 M5).
func NewGatewayEmailSender(baseURL, apiKey, fromAddr, appName string) *GatewayEmailSender {
	return &GatewayEmailSender{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		APIKey:   apiKey,
		FromAddr: fromAddr,
		AppName:  appName,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *GatewayEmailSender) Send(ctx context.Context, to, subject, htmlBody, textBody string) error {
	payload := gatewayEmailRequest{
		To:       to,
		Subject:  subject,
		BodyHTML: htmlBody,
		BodyText: textBody,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gateway: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.BaseURL+"/emails/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gateway: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("gateway: http: %w", err)
	}
	defer resp.Body.Close()
	// 201 = sent immediately, 202 = accepted/queued for retry — both success for caller
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
		// Drain to allow connection reuse (keep-alive).
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	// Sanitize: do NOT echo upstream body in returned error — may leak internal
	// details (stack traces, DB messages) to caller logs (HR-W3-B7 M6).
	io.Copy(io.Discard, resp.Body)
	return fmt.Errorf("gateway: HTTP %d", resp.StatusCode)
}

func (g *GatewayEmailSender) SendPasswordReset(toEmail, userName, resetLink string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	htmlBody := renderResetEmailHTML(userName, resetLink, g.AppName)
	text := fmt.Sprintf("Hola %s,\n\nRestablece tu contraseña de %s: %s\n\nEl enlace expira en 1 hora.",
		userName, g.AppName, resetLink)
	return g.Send(ctx, toEmail, fmt.Sprintf("Restablece tu contraseña de %s", g.AppName), htmlBody, text)
}

// ─────────────────────────────────────────────────────────────────────────────
// SMTPEmailSender — generic SMTP/STARTTLS sender (Brevo, Mailgun, Postmark…)
// ─────────────────────────────────────────────────────────────────────────────

// SMTPEmailSender sends transactional emails via any standard SMTP relay with
// STARTTLS + PLAIN auth (compatible with Brevo smtp-relay.brevo.com:587,
// Mailgun, Postmark SMTP, etc.).
type SMTPEmailSender struct {
	Host     string // e.g. "smtp-relay.brevo.com"
	Port     int    // e.g. 587
	Username string
	Password string
	FromAddr string // e.g. "noreply@eflowsuite.com"
	AppName  string // e.g. "eSTOCK"
}

// Send delivers a multipart/alternative email (HTML + plain text) via STARTTLS.
// It honours ctx cancellation: if the context is already done before dialing,
// Send returns ctx.Err() immediately without touching the network.
func (s *SMTPEmailSender) Send(ctx context.Context, to, subject, htmlBody, textBody string) error {
	// Honour context cancellation before any network I/O.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	from := fmt.Sprintf("%s <%s>", s.AppName, s.FromAddr)
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	raw, err := buildMIMEMessage(from, to, subject, htmlBody, textBody)
	if err != nil {
		return fmt.Errorf("smtp: build MIME: %w", err)
	}

	if err := smtp.SendMail(addr, auth, s.FromAddr, []string{to}, raw); err != nil {
		return fmt.Errorf("smtp: send: %w", err)
	}
	return nil
}

// SendPasswordReset sends the standard password-reset email using the SMTP transport.
func (s *SMTPEmailSender) SendPasswordReset(toEmail, userName, resetLink string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	htmlBody := renderResetEmailHTML(userName, resetLink, s.AppName)
	text := fmt.Sprintf("Hola %s,\n\nRestablece tu contraseña de %s: %s\n\nEl enlace expira en 1 hora.", userName, s.AppName, resetLink)
	return s.Send(ctx, toEmail, fmt.Sprintf("Restablece tu contraseña de %s", s.AppName), htmlBody, text)
}

// buildMIMEMessage constructs a multipart/alternative MIME message with plain
// text and HTML parts using quoted-printable encoding for the HTML part.
func buildMIMEMessage(from, to, subject, htmlBody, textBody string) ([]byte, error) {
	var buf bytes.Buffer

	// RFC 5322 headers.
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	mw := multipart.NewWriter(&buf)
	buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + mw.Boundary() + "\"\r\n")
	buf.WriteString("\r\n")

	// Plain-text part.
	th := make(textproto.MIMEHeader)
	th.Set("Content-Type", "text/plain; charset=UTF-8")
	th.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, err := mw.CreatePart(th)
	if err != nil {
		return nil, err
	}
	qpw := quotedprintable.NewWriter(pw)
	if _, err := strings.NewReader(textBody).WriteTo(qpw); err != nil {
		return nil, err
	}
	if err := qpw.Close(); err != nil {
		return nil, err
	}

	// HTML part.
	hh := make(textproto.MIMEHeader)
	hh.Set("Content-Type", "text/html; charset=UTF-8")
	hh.Set("Content-Transfer-Encoding", "quoted-printable")
	hw, err := mw.CreatePart(hh)
	if err != nil {
		return nil, err
	}
	qph := quotedprintable.NewWriter(hw)
	if _, err := strings.NewReader(htmlBody).WriteTo(qph); err != nil {
		return nil, err
	}
	if err := qph.Close(); err != nil {
		return nil, err
	}

	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Email templates
// ─────────────────────────────────────────────────────────────────────────────

// RenderNotificationEmail returns (htmlBody, textBody) for a given event type.
// Falls back to a generic template for unknown event types.
func RenderNotificationEmail(eventType, title, body string) (htmlBody, text string) {
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

// RenderTrialEmail returns (subject, htmlBody, textBody) for trial lifecycle emails.
// templateType is one of: "trial_reminder_7d", "trial_reminder_11d", "trial_reminder_13d", "trial_expired".
// tenantName and daysLeft are HTML-escaped before embedding.
func RenderTrialEmail(templateType, tenantName string, daysLeft int) (subject, htmlBody, textBody string) {
	switch templateType {
	case "trial_reminder_13d", "trial_reminder_11d", "trial_reminder_7d":
		subject = fmt.Sprintf("Tu prueba de eSTOCK vence en %d días", daysLeft)
		htmlBody = renderTrialReminderHTML(tenantName, daysLeft)
		textBody = fmt.Sprintf(
			"Hola %s,\n\nTu período de prueba gratuita de eSTOCK vence en %d día(s).\n\n"+
				"Para continuar usando eSTOCK sin interrupciones, activa tu suscripción en:\nhttps://app.estock.app/billing\n\n"+
				"Si tienes alguna duda, escríbenos a soporte@eprac.com.\n\neSTOCK — Sistema de gestión de inventario",
			tenantName, daysLeft,
		)
	case "trial_expired":
		subject = "Tu prueba de eSTOCK ha expirado"
		htmlBody = renderTrialExpiredHTML(tenantName)
		textBody = fmt.Sprintf(
			"Hola %s,\n\nTu período de prueba gratuita de eSTOCK ha finalizado y tu cuenta ha sido suspendida.\n\n"+
				"Para reactivar tu cuenta y recuperar acceso completo, visita:\nhttps://app.estock.app/billing\n\n"+
				"Si necesitas ayuda, contáctanos en soporte@eprac.com.\n\neSTOCK — Sistema de gestión de inventario",
			tenantName,
		)
	default:
		subject = "Aviso sobre tu cuenta eSTOCK"
		htmlBody = renderGenericHTML("Aviso sobre tu cuenta eSTOCK", fmt.Sprintf("Hola %s, hay un aviso sobre tu cuenta.", html.EscapeString(tenantName)))
		textBody = fmt.Sprintf("Hola %s,\n\nHay un aviso sobre tu cuenta de eSTOCK.", tenantName)
	}
	return
}

// renderTrialReminderHTML returns a branded HTML email body for a trial reminder.
// tenantName is HTML-escaped before use.
func renderTrialReminderHTML(tenantName string, daysLeft int) string {
	safeName := html.EscapeString(tenantName)
	urgencyColor := "#F59E0B" // amber — default
	urgencyLabel := "Recordatorio de prueba"
	urgencyBg := "#FEF3C7"
	urgencyText := "#92400E"

	if daysLeft <= 7 {
		urgencyColor = "#EF4444" // red — urgent
		urgencyLabel = "Accion requerida — prueba por vencer"
		urgencyBg = "#FEE2E2"
		urgencyText = "#991B1B"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <div style="background:%s;border-left:4px solid %s;padding:12px 16px;border-radius:4px;margin-bottom:24px;">
      <strong style="color:%s;">%s</strong>
    </div>
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:22px;">Tu prueba vence en %d día(s)</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">
      Hola <strong>%s</strong>,<br><br>
      Tu período de prueba gratuita de <strong>eSTOCK</strong> vence en <strong>%d día(s)</strong>.
      Para continuar usando eSTOCK sin interrupciones, activa tu suscripción ahora.
    </p>
    <a href="https://app.estock.app/billing"
       style="display:inline-block;background:#203173;color:#e8d833;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;">
      Activar suscripción
    </a>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">
      Si tienes alguna pregunta, escríbenos a <a href="mailto:soporte@eprac.com" style="color:#203173;">soporte@eprac.com</a>.<br>
      eSTOCK — Sistema de gestión de inventario
    </p>
  </div>
</body></html>`,
		urgencyBg, urgencyColor, urgencyText, urgencyLabel,
		daysLeft, safeName, daysLeft,
	)
}

// renderTrialExpiredHTML returns a branded HTML email body for trial expiration.
// tenantName is HTML-escaped before use.
func renderTrialExpiredHTML(tenantName string) string {
	safeName := html.EscapeString(tenantName)
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <div style="background:#FEE2E2;border-left:4px solid #EF4444;padding:12px 16px;border-radius:4px;margin-bottom:24px;">
      <strong style="color:#991B1B;">Cuenta suspendida</strong>
    </div>
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:22px;">Tu prueba de eSTOCK ha expirado</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">
      Hola <strong>%s</strong>,<br><br>
      Tu período de prueba gratuita de <strong>eSTOCK</strong> ha finalizado y tu cuenta ha sido <strong>suspendida</strong>.
      Para recuperar acceso completo y reactivar tu cuenta, actualiza tu plan de facturación.
    </p>
    <a href="https://app.estock.app/billing"
       style="display:inline-block;background:#203173;color:#e8d833;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;">
      Reactivar cuenta
    </a>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">
      ¿Necesitas ayuda? Contáctanos en <a href="mailto:soporte@eprac.com" style="color:#203173;">soporte@eprac.com</a>.<br>
      eSTOCK — Sistema de gestión de inventario
    </p>
  </div>
</body></html>`, safeName)
}

func renderResetEmailHTML(userName, resetLink, appName string) string {
	safeUserName := html.EscapeString(userName)
	safeAppName := html.EscapeString(appName)
	// resetLink is system-generated (not user-controlled), but escape attribute value for safety.
	safeResetLink := html.EscapeString(resetLink)
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
</body></html>`, safeUserName, safeAppName, safeResetLink)
}

func renderTaskAssignedHTML(title, body string) string {
	return renderGenericHTML(title, body)
}

func renderLotExpiringHTML(title, body string) string {
	safeTitle := html.EscapeString(title)
	safeBody := html.EscapeString(body)
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
</body></html>`, safeTitle, safeBody)
}

func renderLowStockHTML(title, body string) string {
	safeTitle := html.EscapeString(title)
	safeBody := html.EscapeString(body)
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
</body></html>`, safeTitle, safeBody)
}

func renderUserWelcomeHTML(title, body string) string {
	safeTitle := html.EscapeString(title)
	safeBody := html.EscapeString(body)
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:24px;">¡Bienvenido a eSTOCK!</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">%s</p>
    <pre style="background:#F8FAFC;border:1px solid #E2E8F0;border-radius:8px;padding:16px;font-size:14px;color:#334155;">%s</pre>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">eSTOCK — Sistema de gestión de inventario</p>
  </div>
</body></html>`, safeTitle, safeBody)
}

func renderGenericHTML(title, body string) string {
	safeTitle := html.EscapeString(title)
	safeBody := html.EscapeString(body)
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:22px;">%s</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">%s</p>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">eSTOCK — Sistema de gestión de inventario</p>
  </div>
</body></html>`, safeTitle, safeBody)
}
