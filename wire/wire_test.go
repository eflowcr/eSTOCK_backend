package wire_test

import (
	"testing"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/stretchr/testify/assert"
)

// TestEmailSenderForConfig_SMTPPriorityOrder verifies the three-tier selection:
//  1. RESEND_API_KEY set → ResendEmailSender
//  2. SMTP_HOST set (no Resend key) → SMTPEmailSender
//  3. Neither set → LoggerEmailSender
func TestEmailSenderForConfig_SMTPPriorityOrder(t *testing.T) {
	t.Parallel()

	t.Run("resend wins when api key is set", func(t *testing.T) {
		t.Parallel()
		cfg := configuration.Config{
			ResendAPIKey:  "re_test_key",
			SMTPHost:      "smtp-relay.brevo.com",
			SMTPPort:      587,
			SMTPUsername:  "user@brevo.com",
			SMTPPassword:  "secret",
			EmailFrom:     "noreply@eflowsuite.com",
			EmailFromName: "eSTOCK",
		}
		sender := wire.EmailSenderForConfig(cfg)
		_, ok := sender.(*tools.ResendEmailSender)
		assert.True(t, ok, "expected ResendEmailSender when RESEND_API_KEY is set")
	})

	t.Run("smtp fallback when only smtp host is set", func(t *testing.T) {
		t.Parallel()
		cfg := configuration.Config{
			ResendAPIKey:  "", // not set
			SMTPHost:      "smtp-relay.brevo.com",
			SMTPPort:      587,
			SMTPUsername:  "user@brevo.com",
			SMTPPassword:  "secret",
			EmailFrom:     "noreply@eflowsuite.com",
			EmailFromName: "eSTOCK",
		}
		sender := wire.EmailSenderForConfig(cfg)
		got, ok := sender.(*tools.SMTPEmailSender)
		assert.True(t, ok, "expected SMTPEmailSender when SMTP_HOST is set and RESEND_API_KEY is unset")
		if ok {
			assert.Equal(t, "smtp-relay.brevo.com", got.Host)
			assert.Equal(t, 587, got.Port)
			assert.Equal(t, "noreply@eflowsuite.com", got.FromAddr)
			assert.Equal(t, "eSTOCK", got.AppName)
		}
	})

	t.Run("logger fallback when neither is set", func(t *testing.T) {
		t.Parallel()
		cfg := configuration.Config{} // no Resend, no SMTP
		sender := wire.EmailSenderForConfig(cfg)
		_, ok := sender.(*tools.LoggerEmailSender)
		assert.True(t, ok, "expected LoggerEmailSender when neither RESEND_API_KEY nor SMTP_HOST is set")
	})
}
