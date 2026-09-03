package email

import (
	"testing"

	"vantalens/talentwriter/internal/models"
)

func TestUseMicrosoftGraph(t *testing.T) {
	cases := []struct {
		name     string
		settings models.CommentSettings
		want     bool
	}{
		{"explicit microsoft_graph", models.CommentSettings{MailProvider: "microsoft_graph"}, true},
		{"explicit graph alias", models.CommentSettings{MailProvider: "graph"}, true},
		{"provider case and space insensitive", models.CommentSettings{MailProvider: "  Microsoft_Graph "}, true},
		{"implicit via refresh token", models.CommentSettings{MicrosoftRefreshToken: "tok"}, true},
		{"smtp provider", models.CommentSettings{MailProvider: "smtp"}, false},
		{"empty provider without token", models.CommentSettings{}, false},
		{"unknown provider", models.CommentSettings{MailProvider: "sendgrid"}, false},
		// Explicit non-graph provider wins even when a refresh token exists.
		{"smtp provider overrides refresh token", models.CommentSettings{MailProvider: "smtp", MicrosoftRefreshToken: "tok"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := useMicrosoftGraph(tc.settings); got != tc.want {
				t.Fatalf("useMicrosoftGraph(%+v) = %v, want %v", tc.settings, got, tc.want)
			}
		})
	}
}

func TestGraphRecipients(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		if got := graphRecipients(nil); len(got) != 0 {
			t.Fatalf("graphRecipients(nil) = %v, want empty", got)
		}
	})

	t.Run("trims and skips blanks", func(t *testing.T) {
		got := graphRecipients([]string{"  a@example.com ", "", "   ", "b@example.com"})
		if len(got) != 2 {
			t.Fatalf("len(graphRecipients) = %d, want 2", len(got))
		}
		if addr := got[0]["emailAddress"]["address"]; addr != "a@example.com" {
			t.Fatalf("first recipient = %q, want trimmed address", addr)
		}
		if addr := got[1]["emailAddress"]["address"]; addr != "b@example.com" {
			t.Fatalf("second recipient = %q, want %q", addr, "b@example.com")
		}
	})
}

func TestSendPlainReturnsEarlyWithoutNetwork(t *testing.T) {
	t.Run("no recipients is a no-op", func(t *testing.T) {
		settings := models.CommentSettings{SMTPEnabled: true, SMTPHost: "smtp.invalid", SMTPPort: 25}
		if err := sendPlain(settings, nil, "s", "b"); err != nil {
			t.Fatalf("sendPlain with no recipients = %v, want nil", err)
		}
	})

	t.Run("disabled mail returns error", func(t *testing.T) {
		settings := models.CommentSettings{SMTPEnabled: false, SMTPHost: "smtp.invalid", SMTPPort: 25, SMTPFrom: "a@example.com"}
		err := sendPlain(settings, []string{"b@example.com"}, "s", "b")
		if err == nil {
			t.Fatal("sendPlain with disabled mail returned nil error")
		}
	})

	t.Run("missing from and host is a silent no-op", func(t *testing.T) {
		// SMTPEnabled but neither SMTPFrom/SMTPUser nor SMTPHost configured.
		settings := models.CommentSettings{SMTPEnabled: true, SMTPPort: 25}
		if err := sendPlain(settings, []string{"b@example.com"}, "s", "b"); err != nil {
			t.Fatalf("sendPlain without from/host = %v, want nil", err)
		}
	})
}

func TestSendNotificationEarlyReturns(t *testing.T) {
	comment := models.Comment{Author: "alice", Content: "hi"}

	t.Run("smtp disabled", func(t *testing.T) {
		settings := models.CommentSettings{SMTPEnabled: false, NotifyOnPending: true}
		if err := sendNotification(settings, comment, "post"); err != nil {
			t.Fatalf("sendNotification = %v, want nil when SMTP disabled", err)
		}
	})

	t.Run("notify on pending disabled", func(t *testing.T) {
		settings := models.CommentSettings{SMTPEnabled: true, NotifyOnPending: false}
		if err := sendNotification(settings, comment, "post"); err != nil {
			t.Fatalf("sendNotification = %v, want nil when NotifyOnPending disabled", err)
		}
	})

	t.Run("missing recipients", func(t *testing.T) {
		settings := models.CommentSettings{
			SMTPEnabled: true, NotifyOnPending: true,
			SMTPHost: "smtp.invalid", SMTPFrom: "a@example.com",
		}
		if err := sendNotification(settings, comment, "post"); err != nil {
			t.Fatalf("sendNotification = %v, want nil when SMTPTo empty", err)
		}
	})

	t.Run("missing host", func(t *testing.T) {
		settings := models.CommentSettings{
			SMTPEnabled: true, NotifyOnPending: true,
			SMTPFrom: "a@example.com", SMTPTo: []string{"b@example.com"},
		}
		if err := sendNotification(settings, comment, "post"); err != nil {
			t.Fatalf("sendNotification = %v, want nil when SMTPHost empty", err)
		}
	})

	t.Run("from falls back to smtp user check", func(t *testing.T) {
		// Neither SMTPFrom nor SMTPUser: no sender -> silent no-op.
		settings := models.CommentSettings{
			SMTPEnabled: true, NotifyOnPending: true,
			SMTPHost: "smtp.invalid", SMTPTo: []string{"b@example.com"},
		}
		if err := sendNotification(settings, comment, "post"); err != nil {
			t.Fatalf("sendNotification = %v, want nil when no sender configured", err)
		}
	})
}

func TestSendVerificationCodeRequiresEnabledMail(t *testing.T) {
	settings := models.CommentSettings{SMTPEnabled: false}
	if err := SendVerificationCode(settings, "user@example.com", "123456"); err == nil {
		t.Fatal("SendVerificationCode with disabled mail returned nil error")
	}
}

func TestQueueNotificationDropsWhenQueueFull(t *testing.T) {
	// Workers are not started, so the buffered queue never drains; filling it
	// beyond capacity must not block or panic.
	settings := models.CommentSettings{}
	for i := 0; i < queueSize+10; i++ {
		QueueNotification(settings, models.Comment{Author: "a"}, "post")
	}
	// Drain what we queued so other tests are unaffected.
	for drained := 0; drained < queueSize+10; drained++ {
		select {
		case <-queue:
		default:
			return
		}
	}
}
