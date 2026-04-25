package email

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"vantalens/talentwriter/internal/models"
)

const (
	maxRetries  = 3
	queueSize   = 100
	workerCount = 2
)

var (
	queue = make(chan models.EmailJob, queueSize)
	stats = struct {
		sync.Mutex
		sent    int64
		failed  int64
		retried int64
	}{}
)

func StartWorkers() {
	for i := 0; i < workerCount; i++ {
		go worker(i)
	}
	log.Printf("[EMAIL] Started %d email workers", workerCount)
}

func worker(id int) {
	for job := range queue {
		log.Printf("[EMAIL-WORKER-%d] Processing email for: %s", id, job.PostTitle)
		err := sendNotification(job.Settings, job.Comment, job.PostTitle)
		if err == nil {
			stats.Lock()
			stats.sent++
			stats.Unlock()
			log.Printf("[EMAIL-WORKER-%d] Email sent", id)
			continue
		}
		job.Retries++
		if job.Retries < maxRetries {
			waitTime := time.Duration(1<<uint(job.Retries)) * time.Second
			log.Printf("[EMAIL-WORKER-%d] Retry in %v", id, waitTime)
			time.Sleep(waitTime)
			stats.Lock()
			stats.retried++
			stats.Unlock()
			select {
			case queue <- job:
			default:
				stats.Lock()
				stats.failed++
				stats.Unlock()
			}
		} else {
			stats.Lock()
			stats.failed++
			stats.Unlock()
			log.Printf("[EMAIL-WORKER-%d] Failed after %d retries", id, maxRetries)
		}
	}
}

func QueueNotification(settings models.CommentSettings, comment models.Comment, postTitle string) {
	job := models.EmailJob{
		Settings:  settings,
		Comment:   comment,
		PostTitle: postTitle,
		CreatedAt: time.Now(),
	}
	select {
	case queue <- job:
		log.Printf("[EMAIL] Queued notification for: %s", postTitle)
	default:
		log.Printf("[EMAIL] Queue full, dropping notification")
	}
}

func SendVerificationCode(settings models.CommentSettings, to string, code string) error {
	subject := "Vantalens comment verification code"
	body := fmt.Sprintf("Your Vantalens comment verification code is: %s\n\nThis code expires in 10 minutes. If you did not request it, ignore this email.", code)
	return sendPlain(settings, []string{to}, subject, body)
}

func sendNotification(settings models.CommentSettings, comment models.Comment, postTitle string) error {
	if !settings.SMTPEnabled || !settings.NotifyOnPending {
		return nil
	}
	from := settings.SMTPFrom
	if from == "" {
		from = settings.SMTPUser
	}
	if from == "" || len(settings.SMTPTo) == 0 || settings.SMTPHost == "" {
		return nil
	}
	subject := fmt.Sprintf("New Comment - %s", postTitle)
	body := fmt.Sprintf("Post: %s\nAuthor: %s\nPhone: %s\nEmail: %s\nRisk: %s\nContent:\n%s", postTitle, comment.Author, comment.Phone, comment.Email, strings.Join(comment.RiskReasons, ", "), comment.Content)
	return sendPlain(settings, settings.SMTPTo, subject, body)
}

func sendPlain(settings models.CommentSettings, recipients []string, subject string, body string) error {
	if len(recipients) == 0 {
		return nil
	}
	if useMicrosoftGraph(settings) {
		return sendMicrosoftGraph(settings, recipients, subject, body)
	}
	if !settings.SMTPEnabled {
		return fmt.Errorf("mail is not enabled")
	}
	msg := bytes.NewBuffer(nil)
	from := settings.SMTPFrom
	if from == "" {
		from = settings.SMTPUser
	}
	if from == "" || settings.SMTPHost == "" {
		return nil
	}
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + strings.Join(recipients, ",") + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	msg.WriteString(body)
	addr := settings.SMTPHost + ":" + strconv.Itoa(settings.SMTPPort)
	auth := smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPass, settings.SMTPHost)
	if settings.SMTPPort == 465 {
		tlsConfig := &tls.Config{ServerName: settings.SMTPHost}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, settings.SMTPHost)
		if err != nil {
			return err
		}
		defer client.Close()
		if err := client.Auth(auth); err != nil {
			return err
		}
		if err := client.Mail(from); err != nil {
			return err
		}
		for _, to := range recipients {
			if err := client.Rcpt(to); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(msg.Bytes()); err != nil {
			_ = w.Close()
			return err
		}
		return w.Close()
	}
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.StartTLS(&tls.Config{ServerName: settings.SMTPHost}); err != nil {
		return err
	}
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, to := range recipients {
		if err := client.Rcpt(to); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg.Bytes()); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func useMicrosoftGraph(settings models.CommentSettings) bool {
	provider := strings.ToLower(strings.TrimSpace(settings.MailProvider))
	return provider == "microsoft_graph" || provider == "graph" || (provider == "" && strings.TrimSpace(settings.MicrosoftRefreshToken) != "")
}

func sendMicrosoftGraph(settings models.CommentSettings, recipients []string, subject string, body string) error {
	accessToken, err := microsoftAccessToken(settings)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"subject": subject,
			"body": map[string]string{
				"contentType": "Text",
				"content":     body,
			},
			"toRecipients": graphRecipients(recipients),
		},
		"saveToSentItems": true,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := "https://graph.microsoft.com/v1.0/me/sendMail"
	if sender := strings.TrimSpace(settings.MicrosoftSender); sender != "" {
		endpoint = "https://graph.microsoft.com/v1.0/users/" + url.PathEscape(sender) + "/sendMail"
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 25 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("microsoft graph sendMail failed: %s %s", resp.Status, strings.TrimSpace(string(respBody)))
}

func microsoftAccessToken(settings models.CommentSettings) (string, error) {
	tenant := strings.TrimSpace(settings.MicrosoftTenant)
	if tenant == "" {
		tenant = "common"
	}
	clientID := strings.TrimSpace(settings.MicrosoftClientID)
	refreshToken := strings.TrimSpace(settings.MicrosoftRefreshToken)
	if clientID == "" || refreshToken == "" {
		return "", fmt.Errorf("microsoft graph oauth is not configured")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	if secret := strings.TrimSpace(settings.MicrosoftClientSecret); secret != "" {
		form.Set("client_secret", secret)
	}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("scope", "https://graph.microsoft.com/Mail.Send offline_access")
	endpoint := "https://login.microsoftonline.com/" + url.PathEscape(tenant) + "/oauth2/v2.0/token"
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: 25 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("microsoft token refresh failed: %s %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("microsoft token refresh returned no access token")
	}
	return tokenResp.AccessToken, nil
}

func graphRecipients(recipients []string) []map[string]map[string]string {
	out := make([]map[string]map[string]string, 0, len(recipients))
	for _, recipient := range recipients {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			continue
		}
		out = append(out, map[string]map[string]string{
			"emailAddress": {"address": recipient},
		})
	}
	return out
}
