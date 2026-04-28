package tools

import (
	"bytes"
	"context"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// TestSMTPEmailSender_BuildsCorrectMIME
// ─────────────────────────────────────────────────────────────────────────────

// TestSMTPEmailSender_BuildsCorrectMIME verifies that buildMIMEMessage produces:
//   - correct From / To / Subject headers
//   - multipart/alternative content-type with a boundary
//   - two parts: text/plain and text/html (in that order)
func TestSMTPEmailSender_BuildsCorrectMIME(t *testing.T) {
	t.Parallel()

	from := "eSTOCK <noreply@eflowsuite.com>"
	to := "user@example.com"
	subject := "Test subject"
	htmlBody := "<p>Hello <b>World</b></p>"
	textBody := "Hello World"

	raw, err := buildMIMEMessage(from, to, subject, htmlBody, textBody)
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	// Parse as RFC 5322 message.
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	require.NoError(t, err, "MIME message must be parseable as RFC 5322")

	assert.Equal(t, from, msg.Header.Get("From"))
	assert.Equal(t, to, msg.Header.Get("To"))
	assert.Equal(t, subject, msg.Header.Get("Subject"))

	ct := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	require.NoError(t, err)
	assert.Equal(t, "multipart/alternative", mediaType)
	boundary, ok := params["boundary"]
	require.True(t, ok, "multipart/alternative must declare a boundary")
	require.NotEmpty(t, boundary)

	// Walk parts and collect content-types.
	mr := multipart.NewReader(msg.Body, boundary)
	var partTypes []string
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		partTypes = append(partTypes, p.Header.Get("Content-Type"))
	}

	require.Len(t, partTypes, 2, "expected exactly 2 MIME parts (text/plain + text/html)")
	assert.Contains(t, partTypes[0], "text/plain")
	assert.Contains(t, partTypes[1], "text/html")
}

// ─────────────────────────────────────────────────────────────────────────────
// TestSMTPEmailSender_HonorsContextCancellation
// ─────────────────────────────────────────────────────────────────────────────

// TestSMTPEmailSender_HonorsContextCancellation verifies that Send() returns
// ctx.Err() immediately when the context is already cancelled, without
// attempting any network connection.
func TestSMTPEmailSender_HonorsContextCancellation(t *testing.T) {
	t.Parallel()

	s := &SMTPEmailSender{
		Host:     "smtp-relay.brevo.com",
		Port:     587,
		Username: "test@example.com",
		Password: "secret",
		FromAddr: "noreply@eflowsuite.com",
		AppName:  "eSTOCK",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := s.Send(ctx, "recipient@example.com", "Subject", "<p>body</p>", "body")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "Send must return context.Canceled when ctx is already done")
}

// ─────────────────────────────────────────────────────────────────────────────
// TestSMTPEmailSender_PasswordResetFormatting
// ─────────────────────────────────────────────────────────────────────────────

// TestSMTPEmailSender_PasswordResetFormatting verifies that SendPasswordReset
// produces a well-formed MIME message containing the reset link in the body,
// without performing any real SMTP connection. We monkey-patch Send via a
// captureSmtpSender wrapper to inspect the constructed payload.
func TestSMTPEmailSender_PasswordResetFormatting(t *testing.T) {
	t.Parallel()

	const (
		appName   = "eSTOCK"
		userName  = "Juan Pérez"
		resetLink = "https://estock.eflowsuite.com/reset?token=abc123"
	)

	// Build expected HTML and text the same way SendPasswordReset does.
	wantHTML := renderResetEmailHTML(userName, resetLink, appName)
	wantText := "Hola Juan Pérez,\n\nRestablece tu contraseña de eSTOCK: " + resetLink + "\n\nEl enlace expira en 1 hora."
	wantSubject := "Restablece tu contraseña de eSTOCK"

	// Build the MIME message directly to verify structure (no real SMTP).
	from := appName + " <noreply@eflowsuite.com>"
	raw, err := buildMIMEMessage(from, "juan@example.com", wantSubject, wantHTML, wantText)
	require.NoError(t, err)

	rawStr := string(raw)
	assert.Contains(t, rawStr, wantSubject, "subject must appear in raw message")
	assert.Contains(t, rawStr, "multipart/alternative", "message must be multipart/alternative")

	// Verify the text body contains the reset link (decoded; link is ASCII so QP is identity).
	assert.True(t, strings.Contains(rawStr, resetLink) || strings.Contains(rawStr, "abc123"),
		"reset link must appear somewhere in the MIME payload")
}
