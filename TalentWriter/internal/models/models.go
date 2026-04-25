package models

import "time"

type Post struct {
	Title       string `json:"title"`
	Lang        string `json:"lang"`
	Path        string `json:"path"`
	Date        string `json:"date"`
	Status      string `json:"status"`
	StatusColor string `json:"status_color"`
	Pinned      bool   `json:"pinned"`
}

type ArticleRecord struct {
	Post
	Content   string `json:"content,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Comment struct {
	ID           string   `json:"id"`
	Author       string   `json:"author"`
	Phone        string   `json:"phone,omitempty"`
	Email        string   `json:"email"`
	Content      string   `json:"content"`
	Timestamp    string   `json:"timestamp"`
	Approved     bool     `json:"approved"`
	PostPath     string   `json:"post_path"`
	IPAddress    string   `json:"ip_address"`
	UserAgent    string   `json:"user_agent"`
	ParentID     string   `json:"parent_id,omitempty"`
	Images       []string `json:"images,omitempty"`
	Fingerprint  string   `json:"fingerprint,omitempty"`
	CaptchaScore int      `json:"captcha_score,omitempty"`
	VPNSuspected bool     `json:"vpn_suspected,omitempty"`
	RiskReasons  []string `json:"risk_reasons,omitempty"`
}

type CommentSettings struct {
	MailProvider          string   `json:"mail_provider"`
	SMTPEnabled           bool     `json:"smtp_enabled"`
	SMTPHost              string   `json:"smtp_host"`
	SMTPPort              int      `json:"smtp_port"`
	SMTPUser              string   `json:"smtp_user"`
	SMTPPass              string   `json:"smtp_pass"`
	SMTPFrom              string   `json:"smtp_from"`
	SMTPTo                []string `json:"smtp_to"`
	MicrosoftTenant       string   `json:"microsoft_tenant"`
	MicrosoftClientID     string   `json:"microsoft_client_id"`
	MicrosoftClientSecret string   `json:"microsoft_client_secret"`
	MicrosoftRefreshToken string   `json:"microsoft_refresh_token"`
	MicrosoftSender       string   `json:"microsoft_sender"`
	NotifyOnPending       bool     `json:"notify_on_pending"`
	BlacklistIPs          []string `json:"blacklist_ips"`
	BlacklistWords        []string `json:"blacklist_keywords"`
}

type CommentsFile struct {
	Comments []Comment `json:"comments"`
}

type Frontmatter struct {
	Title      string
	Draft      bool
	Date       string
	Categories []string
	Pinned     bool
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Content string      `json:"content,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type CommentWithPost struct {
	Comment
	PostTitle string `json:"post_title"`
}

type EmailJob struct {
	Settings  CommentSettings
	Comment   Comment
	PostTitle string
	Retries   int
	CreatedAt time.Time
}

type JWTClaims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Jti string `json:"jti"`
	Typ string `json:"typ"`
}

type VisitorIP struct {
	IP         string `json:"ip"`
	VisitCount int    `json:"visit_count"`
	FirstSeen  string `json:"first_seen"`
	LastSeen   string `json:"last_seen"`
	Region     string `json:"region,omitempty"`
	Device     string `json:"device,omitempty"`
}

type PageStatistics struct {
	Path     string `json:"path"`
	Title    string `json:"title"`
	Views    int    `json:"views"`
	UV       int    `json:"uv,omitempty"`
	LastSeen string `json:"last_seen,omitempty"`
}

type WebRTCReport struct {
	Supported  bool     `json:"supported"`
	Candidates []string `json:"candidates,omitempty"`
	PublicIPs  []string `json:"public_ips,omitempty"`
	LocalIPs   []string `json:"local_ips,omitempty"`
}

type AnalyticsCollectRequest struct {
	SessionID string        `json:"session_id"`
	Path      string        `json:"path"`
	Title     string        `json:"title"`
	Referrer  string        `json:"referrer"`
	DNSHost   string        `json:"dns_host"`
	Language  string        `json:"language"`
	Timezone  string        `json:"timezone"`
	Screen    string        `json:"screen"`
	WebRTC    *WebRTCReport `json:"webrtc,omitempty"`
	PageView  bool          `json:"page_view"`
}

type VisitRecord struct {
	ID        int64         `json:"id"`
	SessionID string        `json:"session_id"`
	Path      string        `json:"path"`
	Title     string        `json:"title"`
	Referrer  string        `json:"referrer"`
	IP        string        `json:"ip"`
	Device    string        `json:"device"`
	Browser   string        `json:"browser"`
	OS        string        `json:"os"`
	Region    string        `json:"region"`
	Country   string        `json:"country"`
	City      string        `json:"city"`
	DNSHost   string        `json:"dns_host"`
	Language  string        `json:"language"`
	Timezone  string        `json:"timezone"`
	Screen    string        `json:"screen"`
	UserAgent string        `json:"user_agent"`
	WebRTC    *WebRTCReport `json:"webrtc,omitempty"`
	CreatedAt string        `json:"created_at"`
	PageView  bool          `json:"page_view"`
}

type SiteStatistics struct {
	TotalPages      int              `json:"total_pages"`
	TotalViews      int              `json:"total_views"`
	TotalComments   int              `json:"total_comments"`
	PendingComments int              `json:"pending_comments"`
	UniqueIPs       int              `json:"unique_ips"`
	UniqueSessions  int              `json:"unique_sessions"`
	Pages           []PageStatistics `json:"pages,omitempty"`
	Visitors        []VisitorIP      `json:"visitors,omitempty"`
	RecentVisits    []VisitRecord    `json:"recent_visits,omitempty"`
}
