// ==================== WangScape Writer - Go学习项目 ====================
// 这是一个完整的生产级Go应用，适合初学者学习
//
// 学习路线：
// 1. 包（package）& 导入（import）：第1-50行
// 2. 数据结构（struct）& 类型定义（type）：第60-150行
// 3. 函数定义与参数：handleLogin, verifyJWT等
// 4. HTTP处理：http.HandlerFunc, w http.ResponseWriter, r *http.Request
// 5. 错误处理：Go的 err != nil 模式
// 6. 并发：sync.Mutex, goroutine (go func)
// 7. 文件操作：os.Open, os.WriteFile等
// 8. JSON处理：json.Marshal, json.Unmarshal
// 9. 密码学：crypto包的使用
// 10. 中间件模式：withCORS, withAuth, limitRequestBody
//
// 关键Go概念在本文件中的应用：
// - nil在Go中表示零值（地址为nil的指针）
// - error是interface而不是异常
// - defer用于确保资源清理
// - goroutine轻量级绿色线程
// - interface{} 是空接口，可接受任何类型
// - 大写首字母表示exported（public），小写表示unexported（private）

package main

// ==================== 标准库导入详解 ====================
// Go的标准库提供了丰富的功能，大多数应用只需要标准库
// 这里按功能分组显示（实际代码中通常不这样注释）:

import (
    // ===== 数据处理 =====
    "bufio"           // 带缓冲的I/O（读取Hugo输出）
    "bytes"           // 字节缓冲区操作
    "encoding/base64" // Base64编码/解码
    "encoding/csv"    // CSV文件处理
    "encoding/hex"    // 十六进制编码
    "encoding/json"   // JSON序列化/反序列化（最常用）
    "strconv"         // 字符串与其他类型转换
    "strings"         // 字符串操作（Split, Contains等）
    
    // ===== 密码学和安全（crypto包是Go安全的核心）=====
    "crypto/aes"      // AES加密算法
    "crypto/cipher"   // 加密密码模式（CBC, GCM等）
    "crypto/hmac"     // HMAC认证
    "crypto/rand"     // 密码学安全随机数
    "crypto/sha256"   // SHA256哈希算法
    "crypto/subtle"   // 恒定时间比较（防时序攻击）
    "crypto/tls"      // TLS/SSL支持
    
    // ===== 网络和HTTP（Web应用的核心）=====
    "context"         // 上下文（超时、取消、deadline）
    "net"             // 底层网络操作
    "net/http"        // HTTP客户端和服务器
    "net/mail"        // 邮箱地址解析
    "net/smtp"        // SMTP协议实现（发送邮件）
    "net/url"         // URL解析和编码
    
    // ===== 文件和I/O =====
    "compress/gzip"   // gzip压缩/解压
    "fmt"             // 格式化输出（Printf等）
    "html"            // HTML特殊字符转义（防XSS）
    "io"              // I/O接口
    "log"             // 日志记录
    "os"              // 操作系统接口（文件、环境变量等）
    "os/exec"         // 执行外部命令
    "path/filepath"   // 文件路径操作
    
    // ===== 程序和工具 =====
    "regexp"          // 正则表达式
    "runtime"         // Go运行时信息
    "sort"            // 排序算法
    "sync"            // 并发原语（Mutex, WaitGroup等）
    "time"            // 时间处理
)

const (
	PORT     = 8080
	htmlPort = 1313
)

var hugoPath string
var hugoURL string   // 存储Hugo server的URL，供Live Preview使用

const (
    maxCommentNameLen   = 50
    maxCommentEmailLen  = 100
    maxCommentContentLen = 2000
    maxCommentImages    = 5
    maxImageSize        = 5 << 20
)

var (
    adminToken = "" // 从环境变量或配置读取
    rateLimiter = struct {
        sync.Mutex
        records map[string][]time.Time
    }{records: make(map[string][]time.Time)}
)

// ==================== 邮件重试队列（P1优化） ====================
// 用于异步、可靠的邮件发送，支持重试机制

type EmailJob struct {
	Settings CommentSettings
	Comment  Comment
	PostTitle string
	Retries  int
	CreatedAt time.Time
}

const (
	maxEmailRetries = 3
	emailQueueSize  = 100
	emailWorkerCount = 2
)

var (
	emailQueue = make(chan EmailJob, emailQueueSize)
	emailStats = struct {
		sync.Mutex
		sent     int64
		failed   int64
		retried  int64
	}{}
)

// 全局翻译缓存（P2优化）
var translationCache = struct {
	sync.RWMutex
	items map[string]string // key: md5(text+lang) -> translated
}{items: make(map[string]string)}

// 全局文章列表缓存（P2优化）
var (
	postsCache []Post
	postsCacheExpiry time.Time
	postsCacheMutex sync.RWMutex
)

// ==================== 数据结构定义 ====================
// Go中的struct是简单、高效的类型聚合方式
// struct定义了包含多个字段的数据类型，可混合不同类型的字段

// Post 代表博客文章的元数据
// struct后面的 `json:"fieldname"` 被称为struct tag，用于JSON序列化/反序列化
// 使用tag可以自动映射JSON字段，"omitempty"表示字段为空时不序列化
type Post struct {
	Title       string `json:"title"`          // 文章标题
	Lang        string `json:"lang"`          // 语言代码（如zh-cn, en）
	Path        string `json:"path"`          // 文件路径
	Date        string `json:"date"`          // 发布日期
	Status      string `json:"status"`        // 发布状态（draft/published）
	StatusColor string `json:"status_color"` // 状态颜色（用于前端UI）
	Pinned      bool   `json:"pinned"`        // 是否置顶
}

// Frontmatter 代表Markdown文件的YAML前置元数据
// Hugo使用这种格式存储文章元数据（位于文件开头的---|---之间）
type Frontmatter struct {
	Title      string   // 文章标题
	Draft      bool     // 是否为草稿
	Date       string   // 文章发布日期
	Categories []string // 文章分类列表（切片 slice 类型，动态数组）
	Pinned     bool     // 是否为置顶文章
}

// APIResponse 是标准化的API响应结构
// 所有API端点都应返回这种格式，便于前端统一处理
// interface{} 是Go的通用类型，可存储任何类型的值（类似Python的Any）
type APIResponse struct {
	Success bool        `json:"success"`           // 操作是否成功
	Message string      `json:"message,omitempty"` // 返回消息或错误提示
	Content string      `json:"content,omitempty"` // 返回的文本内容
	Data    interface{} `json:"data,omitempty"`    // 返回的数据（任意类型）
}

// Comment 代表博客评论数据
// 这是一个完整的评论对象，包含发布者信息、内容和审核状态
type Comment struct {
	ID           string   `json:"id"`             // 唯一ID（由时间戳和随机数生成）
	Author       string   `json:"author"`         // 评论者名字
	Email        string   `json:"email"`          // 评论者邮箱（用于后续联系）
	Content      string   `json:"content"`        // 评论正文
	Timestamp    string   `json:"timestamp"`      // 评论发布时间戳
	Approved     bool     `json:"approved"`       // 是否已被版主批准
	PostPath     string   `json:"post_path"`      // 所属文章的路径
	IPAddress    string   `json:"ip_address"`     // 评论者IP地址（用于审核和防滥用）
	UserAgent    string   `json:"user_agent"`     // 浏览器信息（调试和审核用）
	ParentID     string   `json:"parent_id,omitempty"` // 父评论ID（支持嵌套回复）
	Images       []string `json:"images,omitempty"`    // 评论附带的图片URL列表
	IssueNumber  int      `json:"issue_number,omitempty"` // GitHub Issue编号（用于从GitHub读取评论）
}

// CommentSettings 代表评论审核和邮件通知的配置
// 这个结构体存储在config/comment_settings.json中，管理员可以动态修改配置
type CommentSettings struct {
    SMTPEnabled     bool     `json:"smtp_enabled"`      // 是否启用邮件通知功能
    SMTPHost        string   `json:"smtp_host"`         // SMTP服务器地址（如mail.google.com）
    SMTPPort        int      `json:"smtp_port"`         // SMTP端口（587用STARTTLS，465用SMTPS隐式加密）
    SMTPUser        string   `json:"smtp_user"`         // SMTP登录用户名
    SMTPPass        string   `json:"smtp_pass"`         // SMTP密码（可加密存储）
    SMTPFrom        string   `json:"smtp_from"`         // 邮件发件人（不设置则用SMTPUser）
    SMTPTo          []string `json:"smtp_to"`           // 接收通知的邮箱地址列表
    NotifyOnPending bool     `json:"notify_on_pending"` // 新评论需审核时是否通知
    BlacklistIPs    []string `json:"blacklist_ips"`     // IP黑名单（拦截这些IP的评论）
    BlacklistWords  []string `json:"blacklist_keywords"` // 关键词黑名单（包含这些词自动拒绝）
}

// CommentsFile 代表评论数据文件的结构
// 用于在comments.json文件中存储所有评论
type CommentsFile struct {
	Comments []Comment `json:"comments"` // 评论列表（切片）
}


// getCommentSettingsPath 获取评论配置文件的完整路径
// Hugo的所有配置文件都存放在config目录下
func getCommentSettingsPath() string {
    return filepath.Join(hugoPath, "config", "comment_settings.json")
}

// loadCommentSettings 从config/comment_settings.json加载评论设置
// 如果文件不存在，返回默认配置；如果解析失败，也返回默认配置（容错处理）
func loadCommentSettings() CommentSettings {
    path := getCommentSettingsPath()
    settings := CommentSettings{
        SMTPEnabled:     false,
        SMTPPort:        587,
        SMTPTo:          []string{"w2343419@gmail.com"},
        NotifyOnPending: true,
        BlacklistIPs:    []string{},
        BlacklistWords:  []string{},
    }

    if _, err := os.Stat(path); os.IsNotExist(err) {
        return settings
    }

    content, err := os.ReadFile(path)
    if err != nil {
        return settings
    }

    if err := json.Unmarshal(content, &settings); err != nil {
        return settings
    }

    return settings
}

// saveCommentSettings 将评论设置保存到JSON文件
// MarshalIndent用于格式化JSON输出，便于手动编辑配置文件
func saveCommentSettings(settings CommentSettings) error {
    path := getCommentSettingsPath()
    data, err := json.MarshalIndent(settings, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}

// isCommentBlacklisted 检查评论是否被黑名单拦截
// 支持IP黑名单和关键词黑名单两种过滤方式
// 所有检查都转换为小写以支持不区分大小写的匹配
func isCommentBlacklisted(settings CommentSettings, ip, author, email, content string) bool {
    ip = strings.TrimSpace(strings.ToLower(ip))
    text := strings.ToLower(strings.Join([]string{author, email, content}, " "))

    for _, b := range settings.BlacklistIPs {
        if strings.TrimSpace(strings.ToLower(b)) != "" && ip != "" && strings.Contains(ip, strings.TrimSpace(strings.ToLower(b))) {
            return true
        }
    }

    for _, w := range settings.BlacklistWords {
        keyword := strings.TrimSpace(strings.ToLower(w))
        if keyword != "" && strings.Contains(text, keyword) {
            return true
        }
    }

    return false
}

// ==================== 邮件队列处理（P1优化） ====================

// startEmailWorkers 启动邮件发送工作线程（带重试机制）
// 在 init() 中调用，处理异步邮件发送
func startEmailWorkers() {
	for i := 0; i < emailWorkerCount; i++ {
		go emailWorker(i)
	}
	log.Printf("[INFO] Started %d email worker threads", emailWorkerCount)
}

// emailWorker 邮件工作线程（带指数退避重试）
func emailWorker(id int) {
	for job := range emailQueue {
		log.Printf("[EMAIL-WORKER-%d] Processing email job for %s (attempt %d/%d)", 
			id, job.PostTitle, job.Retries+1, maxEmailRetries)
		
		// 尝试发送邮件
		err := sendCommentNotification(job.Settings, job.Comment, job.PostTitle)
		
		if err == nil {
			// 成功
			emailStats.Lock()
			emailStats.sent++
			emailStats.Unlock()
			log.Printf("[EMAIL-WORKER-%d] ✅ Email sent successfully", id)
			continue
		}
		
		// 失败，检查是否需要重试
		job.Retries++
		if job.Retries < maxEmailRetries {
			// 指数退避：1秒 -> 2秒 -> 4秒
			waitTime := time.Duration(1<<uint(job.Retries)) * time.Second
			log.Printf("[EMAIL-WORKER-%d] ⚠️ Send failed: %v, retrying in %v", id, err, waitTime)
			
			time.Sleep(waitTime)
			
			emailStats.Lock()
			emailStats.retried++
			emailStats.Unlock()
			
			// 重新入队
			select {
			case emailQueue <- job:
					// 成功入队
			case <-time.After(time.Second):
				// 队列满，记录为失败
				emailStats.Lock()
				emailStats.failed++
				emailStats.Unlock()
				log.Printf("[EMAIL-WORKER-%d] ❌ Email queue full, discarding job", id)
			}
		} else {
			// 已达到最大重试次数
			emailStats.Lock()
			emailStats.failed++
			emailStats.Unlock()
			log.Printf("[EMAIL-WORKER-%d] ❌ Email failed after %d retries: %v", id, maxEmailRetries, err)
		}
	}
}

// queueEmailNotification 将邮件投递到队列（不阻塞）
// 替代之前的 go sendCommentNotification() 方式
func queueEmailNotification(settings CommentSettings, comment Comment, postTitle string) {
	job := EmailJob{
		Settings:  settings,
		Comment:   comment,
		PostTitle: postTitle,
		Retries:   0,
		CreatedAt: time.Now(),
	}
	
	select {
	case emailQueue <- job:
		log.Printf("[QUEUE] Email job enqueued: %s (queue size: %d/%d)", postTitle, len(emailQueue), emailQueueSize)
	case <-time.After(time.Millisecond):
		// 队列满，立即返回但记录警告
		log.Printf("[WARN] Email queue full, dropping notification for: %s", postTitle)
	}
}

// sendCommentNotification 发送新评论待审核的邮件通知给管理员
// 使用SMTP协议，支持TLS加密（端口587）和SMTPS隐式加密（端口465）
func sendCommentNotification(settings CommentSettings, comment Comment, postTitle string) error {
    if !settings.SMTPEnabled {
        log.Printf("[DEBUG] SMTP未启用，跳过邮件发送")
        return nil
    }
    if !settings.NotifyOnPending {
        log.Printf("[DEBUG] 通知功能未启用，跳过邮件发送")
        return nil
    }
    
    log.Printf("[DEBUG] 准备发送邮件: 主机=%s 端口=%d 用户=%s 收件人=%v", settings.SMTPHost, settings.SMTPPort, settings.SMTPUser, settings.SMTPTo)

    from := settings.SMTPFrom
    if from == "" {
        from = settings.SMTPUser
    }
    if from == "" || len(settings.SMTPTo) == 0 || settings.SMTPHost == "" || settings.SMTPPort == 0 {
        log.Printf("[WARN] SMTP配置不完整: from=%s to=%v host=%s port=%d", from, settings.SMTPTo, settings.SMTPHost, settings.SMTPPort)
        return nil
    }

    subject := fmt.Sprintf("新评论待审核 - %s", postTitle)
    body := fmt.Sprintf(
        "文章: %s\n作者: %s\n邮箱: %s\n时间: %s\nIP: %s\nUA: %s\n\n内容:\n%s\n",
        postTitle,
        escapeHTML(comment.Author),      // 安全转义
        escapeHTML(comment.Email),       // 安全转义
        comment.Timestamp,
        comment.IPAddress,
        escapeHTML(comment.UserAgent),   // 安全转义
        escapeHTML(comment.Content),     // 安全转义
    )

    msg := bytes.NewBuffer(nil)
    msg.WriteString("From: " + from + "\r\n")
    msg.WriteString("To: " + strings.Join(settings.SMTPTo, ",") + "\r\n")
    msg.WriteString("Subject: " + subject + "\r\n")
    msg.WriteString("MIME-Version: 1.0\r\n")
    msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
    msg.WriteString("\r\n")
    msg.WriteString(body)

    // 使用新的密码获取函数（支持加密密码和环境变量，提升安全性）
    password, err := getSMTPPassword(settings)
    if err != nil {
        log.Printf("[ERROR] 获取SMTP密码失败: %v", err)
        return err
    }

    addr := settings.SMTPHost + ":" + strconv.Itoa(settings.SMTPPort)
    
    // 检查是否使用安全端口
    var tlsConfig *tls.Config
    if settings.SMTPPort == 465 {
        // SMTPS (隐式TLS)
        tlsConfig = &tls.Config{
            ServerName:         settings.SMTPHost,
            InsecureSkipVerify: false, // 生产环境必须验证证书
        }
    }
    
    auth := smtp.PlainAuth("", settings.SMTPUser, password, settings.SMTPHost)
    
    // 使用SendMail（添加TLS支持）
    if settings.SMTPPort == 465 {
        // SMTPS连接
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
        
        for _, to := range settings.SMTPTo {
            if err := client.Rcpt(to); err != nil {
                return err
            }
        }
        
        w, err := client.Data()
        if err != nil {
            return err
        }
        _, err = w.Write(msg.Bytes())
        if err != nil {
            return err
        }
        return w.Close()
    } else {
        // 标准SMTP + STARTTLS (端口587)
        log.Printf("[DEBUG] 连接到SMTP服务器: %s", addr)
        client, err := smtp.Dial(addr)
        if err != nil {
            log.Printf("[ERROR] SMTP连接失败: %v", err)
            return fmt.Errorf("SMTP连接失败: %w", err)
        }
        defer client.Close()
        
        // 升级到TLS
        log.Printf("[DEBUG] 启动TLS加密...")
        if err := client.StartTLS(&tls.Config{ServerName: settings.SMTPHost}); err != nil {
            log.Printf("[ERROR] TLS启动失败: %v", err)
            return fmt.Errorf("TLS启动失败: %w", err)
        }
        
        // 认证
        log.Printf("[DEBUG] 进行SMTP认证...")
        if err := client.Auth(auth); err != nil {
            log.Printf("[ERROR] SMTP认证失败: %v", err)
            return fmt.Errorf("SMTP认证失败: %w", err)
        }
        
        // 设置发件人
        log.Printf("[DEBUG] 设置发件人: %s", from)
        if err := client.Mail(from); err != nil {
            log.Printf("[ERROR] 设置发件人失败: %v", err)
            return fmt.Errorf("设置发件人失败: %w", err)
        }
        
        for _, to := range settings.SMTPTo {
            log.Printf("[DEBUG] 添加收件人: %s", to)
            if err := client.Rcpt(to); err != nil {
                log.Printf("[ERROR] 添加收件人失败: %v", err)
                return fmt.Errorf("添加收件人失败: %w", err)
            }
        }
        
        log.Printf("[DEBUG] 准备发送邮件数据...")
        w, err := client.Data()
        if err != nil {
            log.Printf("[ERROR] 启动数据传输失败: %v", err)
            return fmt.Errorf("启动数据传输失败: %w", err)
        }
        _, err = w.Write(msg.Bytes())
        if err != nil {
            log.Printf("[ERROR] 写入邮件内容失败: %v", err)
            return fmt.Errorf("写入邮件内容失败: %w", err)
        }
        
        if err := w.Close(); err != nil {
            log.Printf("[ERROR] 完成邮件发送失败: %v", err)
            return fmt.Errorf("完成邮件发送失败: %w", err)
        }
        
        log.Printf("[INFO] ✅ 邮件发送成功！收件人: %v", settings.SMTPTo)
        return nil
    }
}

type CommentWithPost struct {
    Comment
    PostTitle string `json:"post_title"`
}

// collectAllComments 遍历所有文章并收集其评论
// 返回包含文章标题的评论对象列表
func collectAllComments() ([]CommentWithPost, error) {
    var results []CommentWithPost
    contentRoot := filepath.Join(hugoPath, "content")
    if _, err := os.Stat(contentRoot); err != nil {
        return results, nil
    }

    err := filepath.Walk(contentRoot, func(path string, info os.FileInfo, err error) error {
        if err != nil || !info.IsDir() {
            return nil
        }
        commentsPath := filepath.Join(path, "comments.json")
        if _, err := os.Stat(commentsPath); err == nil {
            indexPath := filepath.Join(path, "index.md")
            comments, err := getComments(indexPath)
            if err != nil {
                return nil
            }
            content, err := os.ReadFile(indexPath)
            if err != nil {
                return nil
            }
            fm := parseFrontmatter(string(content))
            for _, c := range comments {
                relPath, _ := filepath.Rel(hugoPath, indexPath)
                c.PostPath = relPath
                results = append(results, CommentWithPost{Comment: c, PostTitle: fm.Title})
            }
        }
        return nil
    })

    if err != nil {
        return results, err
    }

    return results, nil
}

// ==================== 安全工具函数 ====================
// 这些函数演示了Web应用的重要安全概念

// escapeHTML 安全地转义HTML特殊字符，防止XSS（跨站脚本）攻击
// XSS攻击：攻击者在评论中注入 <img src=x onerror=\"alert('xss')\">
// 如果直接显示在页面上，浏览器会执行这段JavaScript代码
// escapeHTML将<变成&lt;，>变成&gt;，使其不被解析为代码标签
// 这是保护应用和用户最重要的防护措施之一
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// validateEmail 验证邮箱地址格式
// 使用Go标准库的RFC 5322解析器，比正则表达式更准确
// 正则表达式很难覆盖邮箱的所有合法格式，而标准库已经处理了许多边界情况
func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil // 成功解析表示格式有效
}

// validatePath 严格验证路径，防止目录遍历
func validatePath(relPath, basePath string) (string, error) {
	// 规范化路径（多次Clean确保安全）
	cleaned := filepath.Clean(relPath)
	cleaned = filepath.Clean(cleaned)
	
	// 检查绝对路径
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute paths not allowed")
	}
	
	// 检查目录遍历
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/../") {
		return "", fmt.Errorf("directory traversal not allowed")
	}
	
	// Windows特定检查
	if strings.ContainsAny(cleaned, ":") {
		return "", fmt.Errorf("invalid characters in path")
	}
	
	// 构建完整路径
	fullPath := filepath.Join(basePath, cleaned)
	fullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid path")
	}
	
	// 验证路径在基目录内
	basePath, _ = filepath.Abs(basePath)
	fullPathLower := strings.ToLower(fullPath)
	basePathLower := strings.ToLower(basePath)
	
	if !strings.HasPrefix(fullPathLower, basePathLower) {
		return "", fmt.Errorf("path outside base directory")
	}
	
	return fullPath, nil
}

// ==================== 密码加密管理 ====================

// getSMTPEncryptionKey 从环境变量获取加密密钥
func getSMTPEncryptionKey() ([]byte, error) {
	keyHex := os.Getenv("SMTP_ENCRYPTION_KEY")
	if keyHex == "" {
		// 如果没有设置密钥，返回错误
		return nil, fmt.Errorf("SMTP_ENCRYPTION_KEY not set in environment")
	}
	
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_ENCRYPTION_KEY format: %v", err)
	}
	
	// 验证密钥长度（应为32字节用于AES-256）
	if len(key) != 32 {
		return nil, fmt.Errorf("SMTP_ENCRYPTION_KEY must be 64 hex characters (32 bytes for AES-256)")
	}
	
	return key, nil
}

// encryptPassword 使用AES-256-GCM加密SMTP密码
func encryptPassword(plainPassword string) (string, error) {
	key, err := getSMTPEncryptionKey()
	if err != nil {
		return "", err
	}
	
	// 创建cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}
	
	// 创建GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}
	
	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}
	
	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plainPassword), nil)
	
	// 返回base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptPassword 解密SMTP密码
func decryptPassword(encryptedPassword string) (string, error) {
	key, err := getSMTPEncryptionKey()
	if err != nil {
		return "", err
	}
	
	// 解码base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", fmt.Errorf("failed to decode password: %v", err)
	}
	
	// 创建cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}
	
	// 创建GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}
	
	// 提取nonce（前nonceSize字节）
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %v", err)
	}
	
	return string(plaintext), nil
}

// getSMTPPassword 从配置或环境变量安全地获取SMTP密码（强制加密）
// ⚠️ 安全性改进：不允许使用明文密码，必须加密或使用环境变量
func getSMTPPassword(settings CommentSettings) (string, error) {
	// 方式1：优先从环境变量读取（推荐用于生产环境）
	envPassword := os.Getenv("SMTP_PASSWORD")
	if envPassword != "" {
		return envPassword, nil
	}
	
	// 方式2：从配置文件解密（必须已加密）
	if settings.SMTPPass != "" {
		// 尝试解密
		decrypted, err := decryptPassword(settings.SMTPPass)
		if err == nil {
			return decrypted, nil
		}
		
		// 检查是否看起来像base64（已加密格式）
		if isBase64(settings.SMTPPass) {
			// 尝试解密应该成功，如果失败说明密钥错误
			log.Printf("[ERROR] SMTP密码解密失败，可能是加密密钥错误: %v", err)
			return "", fmt.Errorf("SMTP password decryption failed - check SMTP_ENCRYPTION_KEY environment variable")
		}
		
		// 如果不是base64格式，说明是明文（不安全！）
		log.Printf("[ERROR] SMTP password appears to be in plaintext - this is insecure!")
		log.Printf("[ERROR] Please encrypt the password using: ")
		log.Printf("[ERROR]   1. Set SMTP_ENCRYPTION_KEY environment variable")
		log.Printf("[ERROR]   2. Use the encryption function to encrypt your password")
		log.Printf("[ERROR]   3. Update config/comment_settings.json with encrypted value")
		return "", fmt.Errorf("SMTP password must be encrypted - plaintext passwords are not allowed")
	}
	
	return "", fmt.Errorf("SMTP password not configured - set SMTP_PASSWORD env var or encrypt password in config")
}

// isBase64 检查字符串是否是有效的base64格式
func isBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// ==================== JWT身份认证系统 ====================

var jwtSecret []byte

// initJWTSecret 初始化JWT密钥。
// 当环境变量缺失时，回退为进程内临时随机密钥，避免本地双击启动直接退出。
func initJWTSecret() {
	secretEnv := os.Getenv("JWT_SECRET")
	if secretEnv == "" {
        temp := make([]byte, 32)
        if _, err := rand.Read(temp); err != nil {
            log.Fatalf("[FATAL] failed to generate temporary JWT secret: %v", err)
        }
        jwtSecret = temp
        log.Printf("[WARN] JWT_SECRET not set; using a temporary in-memory secret for this run")
        return
	}
	
	// 验证密钥长度（至少64个hex字符 = 32字节）
	if len(secretEnv) < 64 {
        log.Printf("[WARN] JWT_SECRET is shorter than recommended 64 hex chars; using it as raw bytes")
        jwtSecret = []byte(secretEnv)
        return
	}
	
	// 尝试从hex字符串解码（推荐方式）
	if decoded, err := hex.DecodeString(secretEnv); err == nil && len(decoded) >= 32 {
		jwtSecret = decoded
		log.Printf("[INFO] JWT secret loaded from environment (length: %d bytes)", len(jwtSecret))
	} else {
		// 备选：直接使用字符串作为密钥
		jwtSecret = []byte(secretEnv)
		log.Printf("[WARN] JWT secret loaded as raw string, recommend using hex format for better cryptography")
	}
}

type jwtClaims struct {
    Sub string `json:"sub"`
    Iat int64  `json:"iat"`
    Exp int64  `json:"exp"`
    Jti string `json:"jti"` // JWT ID for refresh token rotation
    Typ string `json:"typ"` // token type: "access" or "refresh"
}

// 刷新令牌存储 (内存存储，生产环境建议使用Redis)
var refreshTokenStore = make(map[string]int64) // jti -> expiry time
var refreshTokenMutex sync.RWMutex

func base64URLEncode(data []byte) string {
    return base64.RawURLEncoding.EncodeToString(data)
}

func base64URLDecode(s string) ([]byte, error) {
    return base64.RawURLEncoding.DecodeString(s)
}

func getJWTExpiry() time.Duration {
    if hoursStr := os.Getenv("JWT_TTL_HOURS"); hoursStr != "" {
        if hours, err := strconv.Atoi(hoursStr); err == nil && hours > 0 {
            return time.Duration(hours) * time.Hour
        }
    }
    return 8 * time.Hour
}

func signJWT(headerPayload string) string {
    h := hmac.New(sha256.New, jwtSecret)
    h.Write([]byte(headerPayload))
    return base64URLEncode(h.Sum(nil))
}

// ==================== JWT令牌生成 ====================
// createJWT 创建JSON Web Token用于用户身份认证
// 支持两种令牌类型：
//   1. access: 用于API请求认证，短期有效（默认8小时，可通过JWT_TTL_HOURS环境变量配置）
//   2. refresh: 用于刷新access令牌，长期有效（30天）
// 令牌采用HMAC-SHA256签名，存有唯一ID(jti)支持令牌轮转机制
func createJWT(username string, tokenType string) (string, error) {
    if len(jwtSecret) == 0 {
        return "", fmt.Errorf("JWT secret not initialized")
    }
	
    // JWT头部：指定算法和令牌类型
    header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
    // 生成唯一的令牌ID，用于令牌轮转和撤销机制
    jti := fmt.Sprintf("%s-%d-%s", username, time.Now().UnixNano(), generateRandomString(8))
    
    // 计算令牌过期时间
    var expiry time.Duration
    if tokenType == "refresh" {
        expiry = 30 * 24 * time.Hour // 刷新令牌有效期：30天（用于长期离线场景）
    } else {
        expiry = getJWTExpiry() // 访问令牌有效期：从环境变量JWT_TTL_HOURS读取，默认8小时
    }
    
    // JWT负载内容：标准声明
    claims := jwtClaims{
        Sub: username,                            // Subject: 令牌主体（用户名）
        Iat: time.Now().Unix(),                   // Issued At: 令牌颁发时间
        Exp: time.Now().Add(expiry).Unix(),       // Expiration: 令牌过期时间
        Jti: jti,                                 // JWT ID: 令牌唯一标识（用于轮转）
        Typ: tokenType,                           // Type: 令牌类型（access或refresh）
    }
    claimsJSON, err := json.Marshal(claims)
    if err != nil {
        return "", err
    }
	
    // JWT签名过程：header.payload.signature
    payload := base64URLEncode(claimsJSON)
    unsigned := header + "." + payload
    signature := signJWT(unsigned)  // 使用HMAC-SHA256签名
    token := unsigned + "." + signature
    
    // 将刷新令牌存储到内存中，用于后续验证时检查令牌是否被撤销
    if tokenType == "refresh" {
        refreshTokenMutex.Lock()
        refreshTokenStore[jti] = time.Now().Add(expiry).Unix()
        refreshTokenMutex.Unlock()
    }
    
    return token, nil
}

func generateRandomString(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, length)
    for i := range b {
        randByte := make([]byte, 1)
        if _, err := rand.Read(randByte); err != nil {
            // 如果密码随机数生成失败，则回退到顺序方法
            b[i] = charset[i%len(charset)]
            continue
        }
        b[i] = charset[int(randByte[0])%len(charset)]
    }
    return string(b)
}

// ==================== JWT令牌验证 ====================
// verifyJWT 验证JWT令牌的有效性和完整性
// 检查项目：
//   1. 令牌格式（必须有3个'.'分隔的部分）
//   2. 签名有效性（使用恒定时间比较防止时序攻击）
//   3. 负载内容的有效性（JSON格式）
//   4. 令牌是否过期
//   5. 令牌签发时间是否合理
//   6. 刷新令牌是否被撤销
func verifyJWT(token string) (*jwtClaims, error) {
    // 分解JWT为三部分：header.payload.signature
    parts := strings.Split(token, ".")
    if len(parts) != 3 {
        return nil, fmt.Errorf("invalid token format")
    }
    // 验证签名：重新计算header.payload的签名，与原签名比较
    unsigned := parts[0] + "." + parts[1]
    expectedSig := signJWT(unsigned)
    // 使用恒定时间比较防止时序攻击（即使签名错误也不会通过耗时分析比对长度）
    if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(parts[2])) != 1 {
        return nil, fmt.Errorf("invalid token signature")
    }
	
    payloadBytes, err := base64URLDecode(parts[1])
    if err != nil {
        return nil, fmt.Errorf("invalid token payload")
    }
    var claims jwtClaims
    if err := json.Unmarshal(payloadBytes, &claims); err != nil {
        return nil, fmt.Errorf("invalid token claims")
    }
	
    // 检查令牌有效期
    now := time.Now().Unix()
    if claims.Exp <= now {
        return nil, fmt.Errorf("token expired")  // 令牌已过期
    }
    if claims.Iat > now+60 {
        return nil, fmt.Errorf("token issued in the future")  // 防止时钟偏差（允许60秒误差）
    }
    
    // 刷新令牌的额外验证：检查是否在撤销列表中
    if claims.Typ == "refresh" {
        refreshTokenMutex.RLock()
        expiry, exists := refreshTokenStore[claims.Jti]
        refreshTokenMutex.RUnlock()
        
        if !exists || expiry < now {
            return nil, fmt.Errorf("refresh token revoked or expired")
        }
    }
    
    return &claims, nil
}

func verifyAdminCredentials(username, password string) bool {
    adminUser := os.Getenv("ADMIN_USERNAME")
    if adminUser == "" {
        adminUser = "admin"
    }
    if username != adminUser {
        return false
    }
	
    passwordEnv := os.Getenv("ADMIN_PASSWORD")
    passwordHash := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_HASH")))

    if passwordEnv == "" && passwordHash == "" {
        log.Printf("[ERROR] Admin password not configured")
        return false
    }
	
    if passwordHash != "" {
        sum := sha256.Sum256([]byte(password))
        calc := hex.EncodeToString(sum[:])
        result := subtle.ConstantTimeCompare([]byte(calc), []byte(passwordHash)) == 1
        return result
    }
	
    result := subtle.ConstantTimeCompare([]byte(password), []byte(passwordEnv)) == 1
    return result
}

func extractBearerToken(r *http.Request) string {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return ""
    }
    parts := strings.SplitN(authHeader, " ", 2)
    if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
        return ""
    }
    return strings.TrimSpace(parts[1])
}

func requireAuth(w http.ResponseWriter, r *http.Request) bool {
    // 如果未配置任何认证方式，仅允许本地访问
    adminTokenEnv := os.Getenv("ADMIN_TOKEN")
    adminPass := os.Getenv("ADMIN_PASSWORD")
    adminHash := os.Getenv("ADMIN_PASSWORD_HASH")
    if adminTokenEnv == "" && adminPass == "" && adminHash == "" {
        if !isLocalRequest(r) {
            respondJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "认证未配置"})
            return false
        }
        return true
    }
	
    // 兼容旧的X-Admin-Token
    if adminTokenEnv != "" {
        if r.Header.Get("X-Admin-Token") == adminTokenEnv {
            return true
        }
    }
	
    // JWT验证 (仅接受access令牌)
    token := extractBearerToken(r)
    if token != "" {
        claims, err := verifyJWT(token)
        if err == nil {
            // 检查令牌类型
            if claims.Typ == "access" || claims.Typ == "" {
                // 空的Typ表示旧版本的access令牌
                return true
            }
        }
    }
	
    respondJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "未授权"})
    return false
}

// ==================== 中间件：认证保护 ====================
// withAuth 是一个中间件工厂函数，用于保护API端点
// 它验证请求是否通过了认证，未认证的请求将被拒绝
// 支持以下认证方式：
//   1. X-Admin-Token 头（旧版本，兼容性）
//   2. Bearer Token (JWT)，支持access令牌类型
//   3. 本地未配置认证时，仅允许127.0.0.1访问
func withAuth(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !requireAuth(w, r) {
            return
        }
        handler(w, r)
    }
}

// ==================== Go并发安全和资源管理 ====================
// sync.Mutex（互斥锁）是Go的基本同步原语
// 保护对共享资源（日志文件）的访问，防止数据竞争
var auditLogMu sync.Mutex

// writeAuditLog 安全地将审计日志写入文件
// 演示Go的关键特性：
//   1. sync.Mutex实现并发控制
//   2. defer确保资源清理（RAII模式）
//   3. 早期return简化错误处理流程
func writeAuditLog(action string, r *http.Request, details map[string]interface{}) {
    // ======= 获取互斥锁 =======
    // Lock()阻塞直到获得锁，防止其他goroutine同时写入
    auditLogMu.Lock()
    // defer延迟执行，保证函数返回前Unlock()一定被调用
    // 这防止了死锁（代码量多时容易遗漏unlock）
    defer auditLogMu.Unlock()

    // ======= 组装日志数据 =======
    // map[string]interface{} 是Go的通用字典类型
    // interface{} 可存储任意类型（灵活但需类型断言）
    entry := map[string]interface{}{
        "ts":     time.Now().Format(time.RFC3339),
        "action": action,
        "ip":     getRealClientIP(r),
        "ua":     r.UserAgent(),
    }
    // for-range遍历map，顺序随机（Go的设计特性）
    for k, v := range details {
        entry[k] = v
    }

    // ======= JSON序列化 =======
    data, err := json.Marshal(entry)
    if err != nil {
        log.Printf("[WARN] Failed to marshal audit log: %v", err)
        return
    }

    // ======= 文件操作 =======
    logPath := filepath.Join(hugoPath, "config", "audit.log")
    file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
    if err != nil {
        log.Printf("[WARN] Failed to open audit log: %v", err)
        return
    }
    defer file.Close()

    // ======= 写入日志 =======
    _, _ = file.Write(append(data, '\n'))
}

// 定期轮转审计日志 (每天午夜或文件超过100MB时)
func rotateAuditLogPeriodically() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        logPath := filepath.Join(hugoPath, "config", "audit.log")
        info, err := os.Stat(logPath)
        if err != nil {
            continue
        }

        // 检查文件大小是否超过100MB
        if info.Size() > 100*1024*1024 {
            rotateAuditLog(logPath)
        }
    }
}

// 执行日志轮转
func rotateAuditLog(logPath string) {
    auditLogMu.Lock()
    defer auditLogMu.Unlock()

    timestamp := time.Now().Format("2006-01-02-15-04-05")
    newName := logPath + "." + timestamp

    // 重命名当前日志文件
    if err := os.Rename(logPath, newName); err != nil {
        log.Printf("[AUDIT] Failed to rotate audit log: %v", err)
        return
    }

    // 压缩旧日志文件 (可选)
    go compressAuditLog(newName)

    // 清理超过30天的日志
    go cleanupOldAuditLogs(filepath.Dir(logPath))
}

// 压缩日志文件
func compressAuditLog(filePath string) {
    gzipPath := filePath + ".gz"
    inputFile, err := os.Open(filePath)
    if err != nil {
        return
    }
    defer inputFile.Close()

    outputFile, err := os.Create(gzipPath)
    if err != nil {
        return
    }
    defer outputFile.Close()

    writer := gzip.NewWriter(outputFile)
    defer writer.Close()

    if _, err := io.Copy(writer, inputFile); err != nil {
        return
    }

    // 删除原始文件
    os.Remove(filePath)
}

// 清理超过30天的日志
func cleanupOldAuditLogs(logDir string) {
    entries, err := os.ReadDir(logDir)
    if err != nil {
        return
    }

    cutoffTime := time.Now().AddDate(0, 0, -30)

    for _, entry := range entries {
        if !entry.IsDir() && strings.HasPrefix(entry.Name(), "audit.log.") {
            filePath := filepath.Join(logDir, entry.Name())
            info, err := entry.Info()
            if err != nil {
                continue
            }

            if info.ModTime().Before(cutoffTime) {
                os.Remove(filePath)
            }
        }
    }
}

// ==================== IP欺骗防护 ====================

// getRealClientIP 获取真实客户端IP，防止IP欺骗
// ==================== IP地址获取 ====================
// getRealClientIP 安全地获取客户端的真实IP地址
// 支持代理环境下的IP获取，但不容易被伪造
// 优先级：
//   1. 在代理下：检查X-Forwarded-For的最后IP(直接接入代理IP) -> X-Real-IP
//   2. 不在代理下：直接使用连接时的RemoteAddr
func getRealClientIP(r *http.Request) string {
	// 优先检查可信代理的X-Forwarded-For头（仅在生产环境使用代理时）
	// 在开发环境，直接使用RemoteAddr
	isProxied := os.Getenv("BEHIND_PROXY") == "true"
	
	if isProxied {
		// 检查X-Forwarded-For（可信代理设置）
		forwarded := r.Header.Get("X-Forwarded-For")
		if forwarded != "" {
			// 取最后一个IP（直接连接的代理IP）
			ips := strings.Split(forwarded, ",")
			if len(ips) > 0 {
				ip := strings.TrimSpace(ips[len(ips)-1])
				if isValidIP(ip) {
					return ip
				}
			}
		}
		
		// 检查X-Real-IP（查好阿本欺阻子优先為縮統欺巿象）
		realIP := r.Header.Get("X-Real-IP")
		if realIP != "" && isValidIP(realIP) {
			return realIP
		}
	}
	
	// 使用直接连接时的IP（不作异IP欺骗）
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	
	return ip
}

// isValidIP 验证IP地址格式
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ==================== 初始化函数 ====================
// init 是程序启动时自动执行的字段初始化函数，执行各项必要的前置准备
func init() {
	var err error
	// 获取当前Hugo项目的根目录（通常是程序执行目录）
	hugoPath, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	
	// 重要：从环境变量JWT_SECRET或文件中读取JWT密钥
	// 应用会根据密钥签名和验证JWT令牌以确保应用重启后token不会失效
	initJWTSecret()
    // 从环境变量读取管理员令牌（旧版本字段）
    adminToken = os.Getenv("ADMIN_TOKEN")
}

// translateText 使用多源翻译API翻译文本
// translateText 使用多源翻译API翻译文本（带缓存P2优化）
func translateText(text, sourceLang, targetLang string) string {
	// 直接调用带缓存的版本
	return translateTextWithCache(text, sourceLang, targetLang)
}

// translateTextWithCache 带缓存的翻译函数（P2优化）
func translateTextWithCache(text, sourceLang, targetLang string) string {
	input := strings.TrimSpace(text)
	if input == "" || sourceLang == "" || targetLang == "" || sourceLang == targetLang {
		return text
	}

	// 计算缓存键
	cacheKey := generateTranslationCacheKey(input, sourceLang, targetLang)

	// 检查缓存
	translationCache.RLock()
	if cached, ok := translationCache.items[cacheKey]; ok {
		translationCache.RUnlock()
		log.Printf("[CACHE-HIT] Translation cache hit: %s -> %s (key: %s)", sourceLang, targetLang, cacheKey[:8]+"...")
		return cached
	}
	translationCache.RUnlock()

	// 执行翻译
	var translated string
	var err error

	if translated, err = translateWithMyMemory(input, sourceLang, targetLang); err == nil {
		log.Printf("[TRANSLATE-MYMEMORY] Successfully translated %d chars from %s to %s", len(input), sourceLang, targetLang)
	} else {
		log.Printf("[WARN] MyMemory翻译失败: %v", err)
		// 尝试备用方案
		if translated, err = translateWithGoogle(input, sourceLang, targetLang); err == nil {
			log.Printf("[TRANSLATE-GOOGLE] Google备用翻译成功")
		} else {
			log.Printf("[WARN] Google备用翻译失败: %v", err)
			return text
		}
	}

	// 保存到缓存
	translationCache.Lock()
	translationCache.items[cacheKey] = translated
	translationCache.Unlock()
	
	log.Printf("[CACHE-SAVE] Translation cached (cache size: %d)", len(translationCache.items))

	return translated
}

// generateTranslationCacheKey 生成翻译缓存键
func generateTranslationCacheKey(text, sourceLang, targetLang string) string {
	h := sha256.Sum256([]byte(text + "|" + sourceLang + "|" + targetLang))
	return hex.EncodeToString(h[:])
}

func translateWithMyMemory(text, sourceLang, targetLang string) (string, error) {
    client := &http.Client{Timeout: 12 * time.Second}
    escapedText := url.QueryEscape(text)
    apiURL := fmt.Sprintf("https://api.mymemory.translated.net/get?q=%s&langpair=%s|%s", escapedText, sourceLang, targetLang)

    req, err := http.NewRequest(http.MethodGet, apiURL, nil)
    if err != nil {
        return "", err
    }
    req.Header.Set("User-Agent", "WangScape-Writer/1.0")

    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
        return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
    }

    var result struct {
        ResponseData struct {
            TranslatedText string `json:"translatedText"`
        } `json:"responseData"`
        ResponseStatus  int    `json:"responseStatus"`
        ResponseDetails string `json:"responseDetails"`
        QuotaFinished   bool   `json:"quotaFinished"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }

    translated := strings.TrimSpace(html.UnescapeString(result.ResponseData.TranslatedText))
    if result.ResponseStatus != 200 || translated == "" {
        return "", fmt.Errorf("responseStatus=%d details=%s", result.ResponseStatus, result.ResponseDetails)
    }

    if result.QuotaFinished {
        return "", fmt.Errorf("quota finished")
    }

    return translated, nil
}

func translateWithGoogle(text, sourceLang, targetLang string) (string, error) {
    client := &http.Client{Timeout: 12 * time.Second}
    escapedText := url.QueryEscape(text)
    apiURL := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=%s&tl=%s&dt=t&q=%s", sourceLang, targetLang, escapedText)

    req, err := http.NewRequest(http.MethodGet, apiURL, nil)
    if err != nil {
        return "", err
    }
    req.Header.Set("User-Agent", "WangScape-Writer/1.0")

    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
        return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
    }

    var data interface{}
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return "", err
    }

    arr, ok := data.([]interface{})
    if !ok || len(arr) == 0 {
        return "", fmt.Errorf("invalid response format")
    }

    segments, ok := arr[0].([]interface{})
    if !ok || len(segments) == 0 {
        return "", fmt.Errorf("missing translation segments")
    }

    var builder strings.Builder
    for _, seg := range segments {
        piece, ok := seg.([]interface{})
        if !ok || len(piece) == 0 {
            continue
        }
        translatedPart, ok := piece[0].(string)
        if !ok || translatedPart == "" {
            continue
        }
        builder.WriteString(translatedPart)
    }

    translated := strings.TrimSpace(html.UnescapeString(builder.String()))
    if translated == "" {
        return "", fmt.Errorf("empty translated text")
    }

    return translated, nil
}

// ==================== 文件上传安全检查 ====================

// validateFileUpload 验证上传文件的安全性
func validateFileUpload(filename string, fileSize int64, contentType string, allowedMimeTypes map[string]bool, maxSize int64) error {
	// 1. 检查文件大小
	if fileSize <= 0 {
		return fmt.Errorf("invalid file size")
	}
	if fileSize > maxSize {
		return fmt.Errorf("file size exceeds limit: %d > %d", fileSize, maxSize)
	}
	
	// 2. 检查MIME类型
	if !allowedMimeTypes[contentType] {
		return fmt.Errorf("unsupported file type: %s", contentType)
	}
	
	// 3. 检查文件名
	if filename == "" {
		return fmt.Errorf("empty filename")
	}
	
	// 移除路径信息，只保留文件名
	filename = filepath.Base(filename)
	
	// 检查是否包含目录遍历字符
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("invalid filename: path traversal detected")
	}
	
	// 4. 检查特殊字符
	validChars := regexp.MustCompile(`^[a-zA-Z0-9._\-]+$`)
	if !validChars.MatchString(filename) {
		return fmt.Errorf("filename contains invalid characters")
	}
	
	// 5. 检查双重扩展名（防止服务器配置漏洞）
	parts := strings.Split(filename, ".")
	if len(parts) > 2 {
		return fmt.Errorf("multiple extensions not allowed")
	}
	
	return nil
}

// detectImageMIME 更严格地检测图片MIME类型（检查文件头）
func detectImageMIME(data []byte) (string, error) {
	if len(data) < 12 {
		return "", fmt.Errorf("file too small")
	}
	
	// PNG: 89 50 4E 47
	if bytes.Equal(data[0:4], []byte{0x89, 0x50, 0x4E, 0x47}) {
		return "image/png", nil
	}
	
    // JPEG: FF D8 FF 文件笆迹
	if bytes.Equal(data[0:3], []byte{0xFF, 0xD8, 0xFF}) {
		return "image/jpeg", nil
	}
	
	// GIF: 47 49 46 38 (GIF8)
	if bytes.Equal(data[0:4], []byte{0x47, 0x49, 0x46, 0x38}) {
		return "image/gif", nil
	}
	
	// WebP: RIFF ... WEBP 文件笆迹
	if len(data) >= 12 && bytes.Equal(data[0:4], []byte{0x52, 0x49, 0x46, 0x46}) &&
		bytes.Equal(data[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
		return "image/webp", nil
	}
	
	return "", fmt.Errorf("unsupported image format")
}

// getContent 读取文件内容
// 应用: 自动会调用此函数读取Markdown文格物资料
func getContent(relPath string) (string, error) {
	// 验证路径安全性
	fullPath, err := validatePath(relPath, hugoPath)
	if err != nil {
		return "", fmt.Errorf("path validation failed: %v", err)
	}

	// 检查文件扩展名
	if !strings.HasSuffix(strings.ToLower(fullPath), ".md") {
		return "", fmt.Errorf("invalid file type")
	}

	if _, err := os.Stat(fullPath); err != nil {
		return "", fmt.Errorf("file not found")
	}

	content, err := os.ReadFile(fullPath)
	return string(content), err
}

// saveContent 保存文件内容
// 会自动创建必要的父目录
func saveContent(relPath, content string) error {
	// 严格验证路径
	fullPath, err := validatePath(relPath, hugoPath)
	if err != nil {
		return fmt.Errorf("path validation failed: %v", err)
	}

	// 检查文件扩展名
	if !strings.HasSuffix(strings.ToLower(fullPath), ".md") {
		return fmt.Errorf("only .md files allowed")
	}

	// 记录审计日志
	log.Printf("[AUDIT] saveContent: path=%s", relPath)
	
	// 设置严格的文件权限（只有所有者可读写）
	return os.WriteFile(fullPath, []byte(content), 0600)
}

// deletePost 删除一篇文章文件
// 需要严格的安全检查，仅不能删除Hugo预定之外的文件
func deletePost(relPath string) error {
	// 规范化路径分隔符（Windows\\, Unix/）
	relPath = strings.ReplaceAll(relPath, "/", string(os.PathSeparator))
	fullPath := filepath.Join(hugoPath, relPath)

	// 安全检查: 必须是.md文件
	if !strings.HasSuffix(strings.ToLower(relPath), ".md") {
		return fmt.Errorf("only .md files can be deleted")
	}

	// 安全检查: 必须与hugoPath在同一新分支内
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return err
	}
	absHugo, _ := filepath.Abs(hugoPath)

	// Normalize paths for comparison (case-insensitive on Windows)
	absPathNorm := strings.ToLower(filepath.Clean(absPath))
	absHugoNorm := strings.ToLower(filepath.Clean(absHugo))

	if !strings.HasPrefix(absPathNorm, absHugoNorm) {
		return fmt.Errorf("path security violation: file must be within hugo directory")
	}

	// 比较前检查文件是否存在
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", relPath)
		}
		return err
	}

	// 删除文件
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	// 尝试删除空的上级目录
	parentDir := filepath.Dir(fullPath)
	entries, err := os.ReadDir(parentDir)
	if err == nil && len(entries) == 0 {
        _ = os.Remove(parentDir)
	}

	return nil
}

// parseFrontmatter 从提供Markdown文件中提取YAML元数据
// YAML格式: ---\n title: ...\n draft: ...\n ---
func parseFrontmatter(content string) Frontmatter {
	fm := Frontmatter{Title: "Untitled", Draft: false, Date: time.Now().Format("2006-01-02")}

	if !strings.HasPrefix(content, "---") {
		return fm
	}

	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		return fm
	}

	lines := strings.Split(parts[1], "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "title:") {
			fm.Title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			fm.Title = strings.Trim(fm.Title, `"`)
		}

		if strings.HasPrefix(line, "draft:") && strings.Contains(strings.ToLower(line), "true") {
			fm.Draft = true
		}

		if strings.HasPrefix(line, "pinned:") && strings.Contains(strings.ToLower(line), "true") {
			fm.Pinned = true
		}

		if strings.HasPrefix(line, "date:") {
			dateStr := strings.TrimSpace(strings.TrimPrefix(line, "date:"))
			dateStr = strings.Trim(dateStr, `"`)
			if len(dateStr) >= 10 {
				fm.Date = dateStr[:10]
			}
		}
	}

	return fm
}

// extractPostTitle 从文章路径提取标题
func extractPostTitle(postPath string) string {
	fullPath := filepath.Join(hugoPath, postPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		// 如果无法读取文件，返回路径作为标题
		return filepath.Base(postPath)
	}
	
	fm := parseFrontmatter(string(content))
	if fm.Title != "" && fm.Title != "Untitled" {
		return fm.Title
	}
	
	return filepath.Base(postPath)
}

// getGitStatus 获取Git仓库的状态映射
// 返回文件路径到英文管理状态（"M"、"A"等）
func getGitStatus() map[string]string {
	status := make(map[string]string)
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = hugoPath
	output, err := cmd.Output()
	if err != nil {
		return status
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) > 3 {
			stat := strings.TrimSpace(line[:2])
			path := strings.TrimSpace(line[3:])
			path = strings.ReplaceAll(path, `"`, "")
			status[path] = stat
		}
	}

	return status
}

// getPosts 返回文章列表
// 会遍历Hugo内容目录，转换成可序列化的Post对象
func getPosts() []Post {
	// P2优化：缓存文章列表（1分钟过期）
	postsCacheMutex.RLock()
	if time.Now().Before(postsCacheExpiry) && len(postsCache) > 0 {
		defer postsCacheMutex.RUnlock()
		log.Printf("[CACHE-HIT] Posts cache hit (expires in %v, %d posts)", 
			postsCacheExpiry.Sub(time.Now()), len(postsCache))
		return postsCache
	}
	postsCacheMutex.RUnlock()

	// 缓存未命中，执行实际查询
	var posts []Post
	gitStatus := getGitStatus()

	contentRoot := filepath.Join(hugoPath, "content")
	if _, err := os.Stat(contentRoot); err != nil {
		return posts
	}

	filepath.Walk(contentRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".md") || strings.HasPrefix(info.Name(), "_") {
			return nil
		}

		relPath, _ := filepath.Rel(hugoPath, path)
		pathParts := strings.Split(strings.ToLower(relPath), string(os.PathSeparator))

		// 筛选仅仅是文章（post类型），不包含页面
		hasPost := false
		for _, part := range pathParts {
			if part == "post" || part == "posts" {
				hasPost = true
				break
			}
		}

		if !hasPost {
			return nil
		}

		// 推批编程语言（根据路径邏辑）
		lang := "en"
		if len(pathParts) > 1 && (pathParts[1] == "zh-cn" || pathParts[1] == "zh") {
			lang = pathParts[1]
		}

		// 获取git文件管算状态
		gStatus := "clean"
		normPath := strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
		if s, ok := gitStatus[normPath]; ok {
			gStatus = s
		}

		// 从文件中读取YAML整方体
		content, _ := os.ReadFile(path)
		fm := parseFrontmatter(string(content))

		dateStr := time.Unix(info.ModTime().Unix(), 0).Format("2006-01-02")
		if fm.Date != "" {
			dateStr = fm.Date
		}

		// 判断文章发布状态（辸稿/表轿/挚换）
		status := "PUBLISHED"
		color := "#22c55e"
		if fm.Draft {
			status = "DRAFT"
			color = "#eab308"
		} else if gStatus != "clean" {
			status = "UNSAVED"
			color = "#f97316"
		}

		posts = append(posts, Post{
			Title:       fm.Title,
			Lang:        lang,
			Path:        relPath,
			Date:        dateStr,
			Status:      status,
			StatusColor: color,
			Pinned:      fm.Pinned,
		})

		return nil
	})

	// Sort by pinned first, then by date descending, limit to 50
	sort.Slice(posts, func(i, j int) bool {
		if posts[i].Pinned != posts[j].Pinned {
			return posts[i].Pinned // pinned posts come first
		}
		return posts[i].Date > posts[j].Date
	})

	if len(posts) > 50 {
		posts = posts[:50]
	}

	// P2优化：保存到缓存
	postsCacheMutex.Lock()
	postsCache = posts
	postsCacheExpiry = time.Now().Add(1 * time.Minute)  // 1分钟过期
	postsCacheMutex.Unlock()
	
	log.Printf("[CACHE-SAVE] Posts list cached (expires in 1 minute, %d posts)", len(posts))

	return posts
}

// getCommentStats 获取一篇文章的评论统计信息
// 包含pending绑定并未笄准的评论数量
func getCommentStats(postPath string) map[string]int {
	stats := map[string]int{
		"total":   0,
		"pending": 0,
	}

	comments, err := getComments(postPath)
	if err != nil {
		return stats
	}

	stats["total"] = len(comments)
	for _, c := range comments {
		if !c.Approved {
			stats["pending"]++
		}
	}

	return stats
}

// getAllCommentsStats 返回所有文章的评论统计信息
// 包含pending绑定并未笄准的评论数量
func getAllCommentsStats() map[string]interface{} {
	totalPending := 0
	totalComments := 0
	postStats := make(map[string]map[string]int)

	posts := getPosts()
	for _, post := range posts {
		stats := getCommentStats(post.Path)
		postStats[post.Path] = stats
		totalComments += stats["total"]
		totalPending += stats["pending"]
	}

	return map[string]interface{}{
		"total_comments": totalComments,
		"total_pending":  totalPending,
		"post_stats":     postStats,
	}
}

// createSyncPost 创建中英文平行 全文一起爲
// 自动用hugo new命令卫文章，然后自动翻译第二文文本
func createSyncPost(titleZh, categories string) (map[string]interface{}, error) {
	titleEn := translateText(titleZh, "zh", "en")
	filename := sanitizeFilename(titleEn)

	results := make(map[string]interface{})

	// 创建中文文犠
	zhPath := fmt.Sprintf("content/zh-cn/post/%s/index.md", filename)
	cmd := exec.Command("hugo", "new", zhPath)
	cmd.Dir = hugoPath
	if err := cmd.Run(); err == nil {
		updateFrontmatter(zhPath, titleZh, categories)
		results["zh_path"] = zhPath
	}

	// 创建英文文犠
	enPath := fmt.Sprintf("content/en/post/%s/index.md", filename)
	cmd = exec.Command("hugo", "new", enPath)
	cmd.Dir = hugoPath
	if err := cmd.Run(); err == nil {
		updateFrontmatter(enPath, titleEn, categories)
		results["en_path"] = enPath
	}

	return results, nil
}

// sanitizeFilename 将标题且沛成URL安全的文件名
func sanitizeFilename(title string) string {
	reg := regexp.MustCompile("[^a-z0-9]+")
	s := strings.ToLower(title)
	s = reg.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// getCommentsPath 计算评论文件的路径
// postPath格式: content/zh-cn/post/example/index.md
// 评论文件: content/zh-cn/post/example/comments.json
func getCommentsPath(postPath string) string {
	// postPath\u683c\u5f0f: content/zh-cn/post/example/index.md
	// \u8bc4\u8bba\u6587\u4ef6: content/zh-cn/post/example/comments.json
	dir := filepath.Dir(postPath)
	return filepath.Join(dir, "comments.json")
}

// getComments 从文件中读取一篇文章的所有评论
func getComments(postPath string) ([]Comment, error) {
	commentsPath := getCommentsPath(postPath)
	fullPath := filepath.Join(hugoPath, commentsPath)
	
	// 如果文件不存在，返回空序列
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return []Comment{}, nil
	}
	
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	
	var cf CommentsFile
	if err := json.Unmarshal(content, &cf); err != nil {
		return nil, err
	}
	
	return cf.Comments, nil
}

// saveComments 个文件另存所有评论
func saveComments(postPath string, comments []Comment) error {
	commentsPath := getCommentsPath(postPath)
	fullPath := filepath.Join(hugoPath, commentsPath)
	
	cf := CommentsFile{Comments: comments}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(fullPath, data, 0644)
}

// addComment 为一篇文章添加新评论
// 配置: IP地址、User-Agent、时间戳等信息用于审核
func addComment(postPath, author, email, content, ipAddress, userAgent, parentID string) (Comment, error) {
	comments, err := getComments(postPath)
	if err != nil {
        return Comment{}, err
	}
	
	// 生成唯一ID（当前时间戳-评论个数）
	id := fmt.Sprintf("%d-%d", time.Now().Unix(), len(comments))
	
	// 创建新的评论（默认未笄准，需要管理员仪审）
    comment := Comment{
		ID:        id,
		Author:    author,
		Email:     email,
		Content:   content,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Approved:  false,
		PostPath:  postPath,
		IPAddress: ipAddress,
		UserAgent: userAgent,
        ParentID:  parentID,
	}
	
	comments = append(comments, comment)
    return comment, saveComments(postPath, comments)
}

// approveComment 批准某条评论
// 它会查找文章中指定ID的评论，然后将 Approved 设为 true
func approveComment(postPath, commentID string) error {
	comments, err := getComments(postPath)
	if err != nil {
		return err
	}
	
	for i := range comments {
		if comments[i].ID == commentID {
			comments[i].Approved = true
			break
		}
	}
	
	return saveComments(postPath, comments)
}

// deleteGitHubIssue 删除或关闭GitHub Issue
func deleteGitHubIssue(issueNumber int, repo, token string) error {
	if issueNumber == 0 {
		return fmt.Errorf("invalid issue number")
	}
	
	// 使用PATCH请求关闭Issue（GitHub不支持真正删除Issue）
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", repo, issueNumber)
	
	// 创建关闭Issue的请求体
	reqBody := map[string]string{
		"state": "closed",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequest("PATCH", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "WSwriter")
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("failed to read response: %v", readErr)
	}
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	// 解析响应确认状态
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err == nil {
		log.Printf("[INFO] GitHub Issue #%d closed - Response state: %v", issueNumber, result["state"])
	} else {
		log.Printf("[INFO] GitHub Issue #%d closed (response parse failed)", issueNumber)
	}
	return nil
}

// updateGitHubIssueLabels 更新GitHub Issue的标签
func updateGitHubIssueLabels(issueNumber int, repo, token string, labels []string) error {
	if issueNumber == 0 {
		return fmt.Errorf("invalid issue number")
	}
	
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", repo, issueNumber)
	
	// 创建更新标签的请求体
	reqBody := map[string]interface{}{
		"labels": labels,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequest("PATCH", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "WSwriter")
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		bodyText, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(bodyText))
	}
	
	log.Printf("[INFO] GitHub Issue #%d labels updated to: %v", issueNumber, labels)
	return nil
}

// updateGitHubIssue 更新GitHub Issue的内容
func updateGitHubIssue(issueNumber int, repo, token string, comment Comment) error {
	if issueNumber == 0 {
		return fmt.Errorf("invalid issue number")
	}
	
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", repo, issueNumber)
	
	// 构建Issue body
	body := fmt.Sprintf(`**Author:** %s
**Email:** %s
**Post:** %s
**Timestamp:** %s

**Content:**
%s`,
		comment.Author,
		comment.Email,
		comment.PostPath,
		comment.Timestamp,
		comment.Content)
	
	// 创建更新请求体
	reqBody := map[string]interface{}{
		"body": body,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequest("PATCH", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "WSwriter")
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		bodyText, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(bodyText))
	}
	
	log.Printf("[INFO] GitHub Issue #%d updated successfully", issueNumber)
	return nil
}

// deleteComment 撤回一条评论
// 只有管理员才能执行此操作
func deleteComment(postPath, commentID string) error {
	comments, err := getComments(postPath)
	if err != nil {
		return err
	}
	
	var filtered []Comment
	for _, c := range comments {
		if c.ID != commentID {
			filtered = append(filtered, c)
		}
	}
	
	return saveComments(postPath, filtered)
}

// updateFrontmatter 更新文章的YAML元数据（标题、分类等）
// 仅是更新敶月抱月等特定字段，不是Draft状态
func updateFrontmatter(relPath, title, categories string) error {
	fullPath := filepath.Join(hugoPath, relPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	
	// 转义标题中的双引号
	escapedTitle := strings.ReplaceAll(title, `"`, `\"`)

	for _, line := range lines {
		if strings.HasPrefix(line, "title:") {
			newLines = append(newLines, fmt.Sprintf(`title: "%s"`, escapedTitle))
		} else if strings.HasPrefix(line, "categories:") {
			cats := strings.Split(categories, ",")
			for i := range cats {
				cats[i] = strings.TrimSpace(cats[i])
			}
			catsJSON, _ := json.Marshal(cats)
			newLines = append(newLines, fmt.Sprintf(`categories: %s`, catsJSON))
		} else {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(fullPath, []byte(strings.Join(newLines, "\n")), 0644)
}

// handleCommand 执行系统命令（hugo, git等）
// 主要用于Build站点或push代码到Git业务
func handleCommand(cmd string) (map[string]interface{}, error) {
	// 使用带超时的命令执行
	timeout := 5 * time.Minute // 默认5分钟超时
	
	switch cmd {
	case "preview":
		timeout = 10 * time.Second // 预览启动10秒超时
	case "deploy":
		timeout = 10 * time.Minute // 部署可能需要更长时间
	case "build":
		timeout = 5 * time.Minute
	case "sync":
		timeout = 3 * time.Minute
	}
	
	// 建立context用于超时控制
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// 后续命令都通过ctx执行
	switch cmd {
	case "preview":
		// 先杀死可能占用端口的 hugo 进程
		killCtx, killCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer killCancel()
		
		if runtime.GOOS == "windows" {
			exec.CommandContext(killCtx, "taskkill", "/F", "/IM", "hugo.exe").Run()
		} else {
			exec.CommandContext(killCtx, "pkill", "hugo").Run()
		}
		
		time.Sleep(500 * time.Millisecond)
		
		// 先构建一次（包括草稿），确保所有内容都是最新的
		buildCmd := exec.CommandContext(ctx, "hugo", "--buildDrafts", "--minify")
		buildCmd.Dir = hugoPath
		buildOutput, err := buildCmd.CombinedOutput()
		if err != nil {
			return map[string]interface{}{"message": fmt.Sprintf("Build failed: %s", string(buildOutput))}, err
		}
		
		// 启动预览服务器（后台运行，包括草稿）
		serverCmd := exec.CommandContext(ctx, "hugo", "server", 
			"--bind", "127.0.0.1",
			"--buildDrafts",           // 显示草稿文章
			"--disableFastRender",     // 完整渲染，不使用快速渲染
			"--navigateToChanged")     // 保存文件时自动导航
		serverCmd.Dir = hugoPath
		
		go func() {
			// 让 hugo 服务器在后台持续运行
			serverCmd.Start()
		}()
		
		// 等待服务器启动
		time.Sleep(3 * time.Second)
		
		// 在主线程打开浏览器
		openBrowser("http://localhost:1313/WangScape/")
		
		return map[string]interface{}{
			"message": "✅ 预览服务器已启动（包括草稿），浏览器正在打开...",
			"url":     "http://localhost:1313/WangScape/",
		}, nil

	case "deploy":
		// 1. 先编译网站 - 不包含草稿（生产环境）
		buildCmd := exec.CommandContext(ctx, "hugo", "--minify")
		buildCmd.Dir = hugoPath
		buildOutput, err := buildCmd.CombinedOutput()
		if err != nil {
			return map[string]interface{}{"message": fmt.Sprintf("❌ Hugo 构建失败:\n%s\n\n请检查文章格式是否正确。", string(buildOutput))}, err
		}
		
		// 2. 检查是否有变更
		statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
		statusCmd.Dir = hugoPath
		statusOutput, _ := statusCmd.Output()
		if len(strings.TrimSpace(string(statusOutput))) == 0 {
			return map[string]interface{}{"message": "ℹ️  没有任何文件变更，无需提交", "url": ""}, nil
		}
		
		// 3. Git 添加所有更改
		cmd := exec.CommandContext(ctx, "git", "add", ".")
		cmd.Dir = hugoPath
		if err := cmd.Run(); err != nil {
			return map[string]interface{}{"message": fmt.Sprintf("❌ Git add 失败: %v", err)}, err
		}

		// 4. 提交更改
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Web Update: %s", timestamp))
		cmd.Dir = hugoPath
		commitOutput, commitErr := cmd.CombinedOutput()
		if commitErr != nil && !strings.Contains(string(commitOutput), "nothing to commit") {
			return map[string]interface{}{"message": fmt.Sprintf("❌ Git commit 失败: %s", string(commitOutput))}, commitErr
		}

		// 5. 推送到远程
		cmd = exec.Command("git", "push")
		cmd.Dir = hugoPath
		pushOutput, pushErr := cmd.CombinedOutput()
		if pushErr != nil {
			errorMsg := string(pushOutput)
			if strings.Contains(errorMsg, "Permission denied") || strings.Contains(errorMsg, "authentication") {
				return map[string]interface{}{"message": "❌ 认证失败！\n\n请检查:\n1. SSH 密钥是否已配置\n2. GitHub 是否有访问权限\n3. 远程仓库地址是否正确", "url": ""}, pushErr
			} else if strings.Contains(errorMsg, "Connection refused") {
				return map[string]interface{}{"message": "❌ 网络连接失败！\n\n请检查:\n1. 网络是否正常\n2. 是否能访问 GitHub", "url": ""}, pushErr
			}
			return map[string]interface{}{"message": fmt.Sprintf("❌ Git push 失败:\n%s", errorMsg), "url": ""}, pushErr
		}

		return map[string]interface{}{"message": "✅ 构建成功！\n✅ 已提交文件\n✅ 已推送到 GitHub\n\n🎉 网站即将更新...", "url": ""}, nil

	default:
		return map[string]interface{}{"message": "Unknown command"}, nil
	}
}

// ==================== HTTP API处理函数 ====================

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, htmlTemplate)
}

// handleHugoURL 返回Hugo server的实际URL，供VS Code Live Preview使用
func handleHugoURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"url": hugoURL,
	}
	json.NewEncoder(w).Encode(response)
}

func handleGetPosts(w http.ResponseWriter, r *http.Request) {
	posts := getPosts()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func handleGetContent(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问
	if !requireLocal(w, r) {
		return
	}

	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}

	content, err := getContent(relPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"content": content})
}

// ==================== 登录API处理函数 ====================
// handleLogin 处理用户登录请求，验证凭证并生成JWT令牌对
// 演示了Go HTTP处理函数的典型模式：验证 -> 认证 -> 业务逻辑 -> 响应
func handleLogin(w http.ResponseWriter, r *http.Request) {
    // ======= Step 1: 验证HTTP方法 =======
    // 登录操作应使用POST方法（创建会话），而不是GET（仅读取）
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return // 早期返回（early return）是Go的推荐模式，避免深层嵌套
    }

    // ======= Step 2: 安全检查 =======
    if !isLocalRequest(r) {
        respondJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "仅允许本地登录"})
        return
    }

    // ======= Step 3: 限流防护 =======
    // 防止暴力破解：同一IP每分钟最多10次尝试
    ip := getRealClientIP(r)
    if !allowRequest("login:"+ip, 10, time.Minute) {
        respondJSON(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "请求过于频繁"})
        return
    }

    // ======= Step 4: 解析JSON请求体 =======
    // 在Go中，匿名struct用于一次性的临时数据结构
    // 无需为仅使用一次的结构创建顶级type
    var data struct {
        Username string `json:"username"` // 标签（tag）指定JSON字段映射
        Password string `json:"password"`
    }
    // json.NewDecoder从流式input解析，比ioutil.ReadAll + json.Unmarshal更高效
    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        log.Printf("[ERROR] Login request decode failed: %v", err)
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return // Go使用err != nil检查错误，而不是try/catch异常机制
    }
    // 清理用户输入：移除前后空白字符，防止隐藏的空格导致认证失败
    data.Username = strings.TrimSpace(data.Username)
    
    // ======= Step 5: 验证管理员凭证 =======
    if !verifyAdminCredentials(data.Username, data.Password) {
        log.Printf("[WARN] Login failed - Invalid credentials for user: %s", data.Username)
        writeAuditLog("login_failed", r, map[string]interface{}{"username": data.Username})
        respondJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "用户名或密码错误"})
        return
    }

    // ======= Step 6: 生成JWT令牌对 =======
    // 访问令牌（access token）：短期有效（8小时），用于一般API请求认证
    accessToken, err := createJWT(data.Username, "access")
    if err != nil {
        log.Printf("[ERROR] Failed to create access token: %v", err)
        writeAuditLog("login_error", r, map[string]interface{}{"username": data.Username, "error": err.Error()})
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "生成令牌失败"})
        return
    }

    // 刷新令牌（refresh token）：长期有效（30天），用于获取新的访问令牌
    refreshToken, err := createJWT(data.Username, "refresh")
    if err != nil {
        log.Printf("[ERROR] Failed to create refresh token: %v", err)
        writeAuditLog("login_error", r, map[string]interface{}{"username": data.Username, "error": err.Error()})
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "生成刷新令牌失败"})
        return
    }

    accessExpiresAt := time.Now().Add(getJWTExpiry()).Format(time.RFC3339)
    refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)
    log.Printf("[INFO] Login successful - User: %s", data.Username)
    writeAuditLog("login_success", r, map[string]interface{}{"username": data.Username})
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "success":              true,
        "access_token":         accessToken,
        "refresh_token":        refreshToken,
        "access_expires_at":    accessExpiresAt,
        "refresh_expires_at":   refreshExpiresAt,
        "token_type":           "Bearer",
    })
}

func handleRefreshToken(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if !isLocalRequest(r) {
        respondJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "仅允许本地刷新令牌"})
        return
    }

    var data struct {
        RefreshToken string `json:"refresh_token"`
    }
    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }

    // 验证刷新令牌
    claims, err := verifyJWT(data.RefreshToken)
    if err != nil {
        writeAuditLog("refresh_token_failed", r, map[string]interface{}{"error": err.Error()})
        respondJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "刷新令牌无效"})
        return
    }

    if claims.Typ != "refresh" {
        writeAuditLog("refresh_token_failed", r, map[string]interface{}{"error": "not a refresh token"})
        respondJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "令牌类型错误"})
        return
    }

    // 令牌轮转: 撤销旧刷新令牌并发放新的
    refreshTokenMutex.Lock()
    delete(refreshTokenStore, claims.Jti)
    refreshTokenMutex.Unlock()

    // 生成新的访问令牌和刷新令牌
    newAccessToken, err := createJWT(claims.Sub, "access")
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "生成令牌失败"})
        return
    }

    newRefreshToken, err := createJWT(claims.Sub, "refresh")
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "生成刷新令牌失败"})
        return
    }

    accessExpiresAt := time.Now().Add(getJWTExpiry()).Format(time.RFC3339)
    refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)
    writeAuditLog("refresh_token_success", r, map[string]interface{}{"username": claims.Sub})
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "success":              true,
        "access_token":         newAccessToken,
        "refresh_token":        newRefreshToken,
        "access_expires_at":    accessExpiresAt,
        "refresh_expires_at":   refreshExpiresAt,
        "token_type":           "Bearer",
    })
}

func handleSaveContent(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问
	if !requireLocal(w, r) {
		return
	}

    // 限流：防止文件系统被滥用
    ip := getRealClientIP(r)
	if !allowRequest("save_content:"+ip, 30, time.Minute) {
		respondJSON(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "请求过于频繁"})
		return
	}

	var data struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	if err := saveContent(data.Path, data.Content); err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
		return
	}
    writeAuditLog("save_content", r, map[string]interface{}{ "path": data.Path })
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Saved"})
}

func handleDeletePost(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问
	if !requireLocal(w, r) {
		return
	}

    // 限流：防止文件被滥用删除
    ip := getRealClientIP(r)
	if !allowRequest("delete_post:"+ip, 10, time.Minute) {
		respondJSON(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "请求过于频繁"})
		return
	}

	var data struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	if err := deletePost(data.Path); err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
		return
	}
    writeAuditLog("delete_post", r, map[string]interface{}{ "path": data.Path })
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Deleted"})
}

// getCommentsFromGitHub 从GitHub Issues获取评论
// checkGitHubRateLimit 检查GitHub API速率限制（P1优化）
// 防止因频繁调用而被限流
func checkGitHubRateLimit(resp *http.Response) error {
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	reset := resp.Header.Get("X-RateLimit-Reset")
	limit := resp.Header.Get("X-RateLimit-Limit")
	
	remainingInt := 0
	if remaining != "" {
		remainingInt, _ = strconv.Atoi(remaining)
	}
	
	limitInt := 0
	if limit != "" {
		limitInt, _ = strconv.Atoi(limit)
	}
	
	if remaining != "" && reset != "" {
		resetUnix, _ := strconv.ParseInt(reset, 10, 64)
		resetTime := time.Unix(resetUnix, 0)
		duration := resetTime.Sub(time.Now())
		
		log.Printf("[GITHUB-API] Rate Limit - Remaining: %d/%d, Reset in: %v", 
			remainingInt, limitInt, duration)
		
		if remainingInt == 0 {
			return fmt.Errorf("GitHub API rate limit exceeded - reset in %v", duration)
		}
		
		// 警告：剩余请求少于10个
		if remainingInt < 10 {
			log.Printf("[WARN] GitHub API rate limit low (remaining: %d)", remainingInt)
		}
	}
	
	return nil
}

func getCommentsFromGitHub(postPath, repo, token string) ([]Comment, error) {
	// 构建GitHub API URL - 只获取open状态的Issues
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues?state=open&labels=comment&per_page=100", repo)
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "WSwriter")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// P1优化：检查速率限制
	if err := checkGitHubRateLimit(resp); err != nil {
		log.Printf("[ERROR] %v", err)
		// 不中断，继续处理但记录警告
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var issues []struct {
		ID     int    `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		CreatedAt string `json:"created_at"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}
	
	var comments []Comment
	for _, issue := range issues {
		// 从issue body解析评论字段
		body := issue.Body
		
		// 提取Post路径
		issuePostPath := extractField(body, "Post")
		if issuePostPath != postPath {
			continue
		}
		
		// 检查是否approved
		approved := false
		for _, label := range issue.Labels {
			if label.Name == "approved" {
				approved = true
				break
			}
		}
		
		// 只返回approved的评论
		if !approved {
			continue
		}
		
		comment := Comment{
			ID:          fmt.Sprintf("%d", issue.ID),
			Author:      extractField(body, "Author"),
			Email:       extractField(body, "Email"),
			Content:     extractContent(body),
			Timestamp:   issue.CreatedAt,
			Approved:    approved,
			PostPath:    issuePostPath,
			IssueNumber: issue.Number, // 保存Issue编号用于删除
		}
		
		// 调试日志：输出Issue body原始内容
		log.Printf("[DEBUG] GitHub Issue #%d body (first 500 chars): %s", issue.Number, func() string {
			if len(body) > 500 {
				return body[:500] + "..."
			}
			return body
		}())
		log.Printf("[DEBUG] Extracted comment - Author: %s, Email: %s, Content length: %d", comment.Author, comment.Email, len(comment.Content))
		
		comments = append(comments, comment)
	}
	
	return comments, nil
}

// extractField 从Issue body提取字段（支持Markdown格式）
func extractField(body, field string) string {
	lines := strings.Split(body, "\n")
	// 支持两种格式: "Field: value" 和 "**Field:** value"
	prefix1 := field + ":"
	prefix2 := "**" + field + ":**"
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix2) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix2))
		}
		if strings.HasPrefix(line, prefix1) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix1))
		}
	}
	return ""
}

// extractContent 提取评论内容
func extractContent(body string) string {
	lines := strings.Split(body, "\n")
	inContent := false
	var contentLines []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 检查是否是元数据字段（Post, Author, Email, Timestamp等）
		if strings.HasPrefix(trimmed, "**Post:**") || 
		   strings.HasPrefix(trimmed, "**Author:**") || 
		   strings.HasPrefix(trimmed, "**Email:**") || 
		   strings.HasPrefix(trimmed, "**Timestamp:**") ||
		   strings.HasPrefix(trimmed, "Post:") || 
		   strings.HasPrefix(trimmed, "Author:") || 
		   strings.HasPrefix(trimmed, "Email:") || 
		   strings.HasPrefix(trimmed, "Timestamp:") {
			continue
		}
		
		// 支持两种格式: "Content:" 和 "**Content:**"
		if trimmed == "Content:" || trimmed == "**Content:**" {
			inContent = true
			continue
		}
		
		// 如果已经找到Content标记，或者是空行之后的内容（说明元数据结束了）
		if inContent || (trimmed != "" && len(contentLines) == 0 && !strings.Contains(trimmed, ":**")) {
			inContent = true
			contentLines = append(contentLines, line)
		} else if inContent || len(contentLines) > 0 {
			contentLines = append(contentLines, line)
		}
	}
	
	return strings.TrimSpace(strings.Join(contentLines, "\n"))
}

func handleGetComments(w http.ResponseWriter, r *http.Request) {
	postPath := r.URL.Query().Get("path")
	if postPath == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}

	// 尝试从GitHub Issues获取评论
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubRepo := "w2343419-del/WangScape" // 从配置读取
	
	log.Printf("[DEBUG] GITHUB_TOKEN存在: %v, 长度: %d", githubToken != "", len(githubToken))
	
	if githubToken != "" && githubRepo != "" {
		log.Printf("[DEBUG] 尝试从GitHub获取评论: repo=%s, post=%s", githubRepo, postPath)
		comments, err := getCommentsFromGitHub(postPath, githubRepo, githubToken)
		if err == nil {
			log.Printf("[DEBUG] GitHub评论获取成功，数量: %d", len(comments))
			respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: comments})
			return
		}
		log.Printf("[DEBUG] GitHub评论获取失败: %v，降级到本地存储", err)
		// 如果GitHub失败，继续使用本地存储
	}

	// 降级：使用本地comments.json
	comments, err := getComments(postPath)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
		return
	}

	// 仅返回对公众可见的笄准评论
	var approved []Comment
	for _, c := range comments {
		if c.Approved {
			approved = append(approved, c)
		}
	}

	respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: approved})
}

func handleAddComment(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

	var data struct {
		PostPath string   `json:"post_path"`
		Author   string   `json:"author"`
		Email    string   `json:"email"`
		Content  string   `json:"content"`
        ParentID string   `json:"parent_id"`
        Images   []string `json:"images"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

    // 使用新的IP获取函数，防止IP欺骗
    ipAddress := getRealClientIP(r)
    if !allowRequest("add_comment:"+ipAddress, 5, time.Minute) {
        respondJSON(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "请求过于频繁"})
        return
    }

    // 获取User-Agent（浏览器信息）
    userAgent := r.Header.Get("User-Agent")

    data.Author = strings.TrimSpace(data.Author)
    data.Email = strings.TrimSpace(data.Email)
    data.Content = strings.TrimSpace(data.Content)
    data.PostPath = strings.TrimSpace(data.PostPath)

    if data.Author == "" || data.Email == "" || data.Content == "" || data.PostPath == "" {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }
    cleanPostPath := filepath.Clean(data.PostPath)
    if filepath.IsAbs(cleanPostPath) || strings.HasPrefix(cleanPostPath, "..") || strings.Contains(cleanPostPath, ":") {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "路径非法"})
        return
    }
    if len(data.Author) > maxCommentNameLen || len(data.Email) > maxCommentEmailLen || len(data.Content) > maxCommentContentLen {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "内容过长"})
        return
    }
    if _, err := mail.ParseAddress(data.Email); err != nil {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "邮箱格式不正确"})
        return
    }
    if len(data.Images) > maxCommentImages {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "图片数量过多"})
        return
    }
    for _, img := range data.Images {
        if !strings.HasPrefix(img, "/img/comments/") || strings.Contains(img, "..") {
            respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "图片路径非法"})
            return
        }
    }

    settings := loadCommentSettings()
    if isCommentBlacklisted(settings, ipAddress, data.Author, data.Email, data.Content) {
        respondJSON(w, http.StatusOK, APIResponse{Success: false, Message: "评论被拦截"})
        return
    }

    // 获取现有评论
    comments, err := getComments(data.PostPath)
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
        return
    }
    
    // 生成唯一ID
    id := fmt.Sprintf("%d-%d", time.Now().Unix(), len(comments))
    
    // 安全转义用户输入
    comment := Comment{
        ID:        id,
        Author:    escapeHTML(data.Author),    // 防XSS
        Email:     escapeHTML(data.Email),     // 防XSS
        Content:   escapeHTML(data.Content),   // 防XSS
        Timestamp: time.Now().Format("2006-01-02 15:04:05"),
        Approved:  false,
        PostPath:  data.PostPath,
        IPAddress: ipAddress,
        UserAgent: escapeHTML(userAgent),      // 防XSS
        ParentID:  data.ParentID,
        Images:    data.Images,
    }
    
    // 保存评论
    comments = append(comments, comment)
    if err := saveComments(data.PostPath, comments); err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
		return
	}

    // 记录审计日志
    log.Printf("[AUDIT] addComment: author=%s ip=%s path=%s approved=%v", 
        data.Author, ipAddress, data.PostPath, comment.Approved)

    // 发送邮件通知（异步队列，不阻塞主流程，支持重试）
    // 改进：使用邮件队列而不是 go func()，支持失败重试
    postTitle := ""
    fullPath := filepath.Join(hugoPath, data.PostPath)
    if content, err := os.ReadFile(fullPath); err == nil {
        fm := parseFrontmatter(string(content))
        postTitle = fm.Title
    }
    queueEmailNotification(settings, comment, postTitle)

	respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "评论已提交，等待审核"})
}

func handleUploadCommentImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

    // 使用新的IP获取函数，防止IP欺骗
    ipAddress := getRealClientIP(r)
    if !allowRequest("upload_image:"+ipAddress, 10, time.Minute) {
        respondJSON(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "请求过于频繁"})
        return
    }

	// 解析multipart form (最大10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "文件过大"})
		return
	}

	file, handler, err := r.FormFile("image")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "无法读取文件"})
		return
	}
	defer file.Close()

    if handler.Size > maxImageSize {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "文件过大"})
        return
    }

	// 检查文件类型
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

    // 读取文件头判断真实类型（增强安全检查）
    head := make([]byte, 512)
    n, _ := file.Read(head)
    
    // 使用更严格的MIME类型检测
    contentType, err := detectImageMIME(head[:n])
    if err != nil {
        // 如果魔术字节检测失败，尝试标准检测
        contentType = http.DetectContentType(head[:n])
    }
    
    if !allowedTypes[contentType] {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "只支持 JPG, PNG, GIF, WebP 格式"})
		return
	}
    
    // 验证文件上传安全性
    if err := validateFileUpload(handler.Filename, handler.Size, contentType, allowedTypes, maxImageSize); err != nil {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: fmt.Sprintf("文件验证失败: %v", err)})
        return
    }

    var reader io.Reader = file
    if seeker, ok := file.(io.Seeker); ok {
        _, _ = seeker.Seek(0, io.SeekStart)
    } else {
        reader = io.MultiReader(bytes.NewReader(head[:n]), file)
    }

	// 生成唯一文件名（不使用用户提供的文件名）
    extMap := map[string]string{
        "image/jpeg": ".jpg",
        "image/jpg":  ".jpg",
        "image/png":  ".png",
        "image/gif":  ".gif",
        "image/webp": ".webp",
    }
    ext := extMap[contentType]
	filename := fmt.Sprintf("comment_%d%s", time.Now().UnixNano(), ext)
	
	// 确保目录存在，权限设置为0755（仅owner可写）
	uploadDir := filepath.Join(hugoPath, "static", "img", "comments")
	os.MkdirAll(uploadDir, 0755)
	
	// 保存文件，权限设置为0600（仅owner可读写）
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "保存失败"})
		return
	}
	defer dst.Close()
    
    // 修改文件权限为0600
    os.Chmod(filepath.Join(uploadDir, filename), 0600)

    limitReader := io.LimitReader(reader, maxImageSize+1)
    written, err := io.Copy(dst, limitReader)
    if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "保存失败"})
		return
	}
    if written > maxImageSize {
        _ = os.Remove(filepath.Join(uploadDir, filename))
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "文件过大"})
        return
    }

	// 返回图片URL
	imageURL := "/img/comments/" + filename
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true, 
		Message: "上传成功",
		Data:    map[string]string{"url": imageURL},
	})
}

func handleApproveComment(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问敏感操作
	if !requireLocal(w, r) {
		return
	}

	var data struct {
		PostPath    string `json:"post_path"`
		CommentID   string `json:"comment_id"`
		IssueNumber int    `json:"issue_number,omitempty"` // GitHub Issue编号
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	// 如果有IssueNumber，说明是从GitHub Issues批准
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubRepo := "w2343419-del/WangScape"
	
	if data.IssueNumber > 0 && githubToken != "" && githubRepo != "" {
		log.Printf("[DEBUG] 批准GitHub Issue #%d", data.IssueNumber)
		// 将pending标签改为approved
		if err := updateGitHubIssueLabels(data.IssueNumber, githubRepo, githubToken, []string{"approved", "comment"}); err != nil {
			log.Printf("[ERROR] 更新GitHub Issue标签失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "更新GitHub Issue失败: " + err.Error()})
			return
		}
		log.Printf("[INFO] 成功批准GitHub Issue #%d", data.IssueNumber)
	} else {
		// 否则从本地文件批准
		log.Printf("[DEBUG] 从本地文件批准评论: %s", data.CommentID)
		if err := approveComment(data.PostPath, data.CommentID); err != nil {
			respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
			return
		}
	}
	
	// 发送邮件通知（审批操作）- 异步队列，支持重试
	settings := loadCommentSettings()
	postTitle := extractPostTitle(data.PostPath)
	comment := Comment{
		Author:    "管理员",
		Email:     "admin@system",
		Content:   fmt.Sprintf("评论 #%s 已被批准", data.CommentID),
		PostPath:  data.PostPath,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	queueEmailNotification(settings, comment, "[批准] "+postTitle)
    
    writeAuditLog("approve_comment", r, map[string]interface{}{ "post_path": data.PostPath, "comment_id": data.CommentID, "issue_number": data.IssueNumber })
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "评论已批准"})
}

func handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问敏感操作
	if !requireLocal(w, r) {
		return
	}

	var data struct {
		PostPath    string `json:"post_path"`
		CommentID   string `json:"comment_id"`
		IssueNumber int    `json:"issue_number,omitempty"` // GitHub Issue编号
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	// 如果有IssueNumber，说明是从GitHub Issues删除
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubRepo := "w2343419-del/WangScape"
	
	if data.IssueNumber > 0 && githubToken != "" && githubRepo != "" {
		log.Printf("[DEBUG] 删除GitHub Issue #%d", data.IssueNumber)
		if err := deleteGitHubIssue(data.IssueNumber, githubRepo, githubToken); err != nil {
			log.Printf("[ERROR] 删除GitHub Issue失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "删除GitHub Issue失败: " + err.Error()})
			return
		}
		log.Printf("[INFO] 成功删除GitHub Issue #%d", data.IssueNumber)
	} else {
		// 否则从本地文件删除
		log.Printf("[DEBUG] 从本地文件删除评论: %s", data.CommentID)
		if err := deleteComment(data.PostPath, data.CommentID); err != nil {
			respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
			return
		}
	}
	
	// 发送邮件通知（删除操作）- 异步队列，支持重试
	settings := loadCommentSettings()
	postTitle := extractPostTitle(data.PostPath)
	comment := Comment{
		Author:    "管理员",
		Email:     "admin@system",
		Content:   fmt.Sprintf("评论 #%s 已被删除", data.CommentID),
		PostPath:  data.PostPath,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	queueEmailNotification(settings, comment, "[删除] "+postTitle)
	
    writeAuditLog("delete_comment", r, map[string]interface{}{ "post_path": data.PostPath, "comment_id": data.CommentID, "issue_number": data.IssueNumber })
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "评论已删除"})
}

func handleUpdateComment(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问敏感操作
	if !requireLocal(w, r) {
		return
	}

	var data struct {
		PostPath    string `json:"post_path"`
		CommentID   string `json:"comment_id"`
		IssueNumber int    `json:"issue_number,omitempty"`
		Author      string `json:"author"`
		Email       string `json:"email"`
		Content     string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	// 如果有IssueNumber，说明是GitHub Issues评论
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubRepo := "w2343419-del/WangScape"
	
	if data.IssueNumber > 0 && githubToken != "" && githubRepo != "" {
		log.Printf("[DEBUG] 更新GitHub Issue #%d", data.IssueNumber)
		
		// 构建更新的评论对象
		updatedComment := Comment{
			Author:    data.Author,
			Email:     data.Email,
			Content:   data.Content,
			PostPath:  data.PostPath,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		
		if err := updateGitHubIssue(data.IssueNumber, githubRepo, githubToken, updatedComment); err != nil {
			log.Printf("[ERROR] 更新GitHub Issue失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "更新GitHub Issue失败: " + err.Error()})
			return
		}
		log.Printf("[INFO] 成功更新GitHub Issue #%d", data.IssueNumber)
		
		// 发送邮件通知（编辑操作）- 异步队列，支持重试
		settings := loadCommentSettings()
		postTitle := extractPostTitle(data.PostPath)
		queueEmailNotification(settings, updatedComment, "[编辑] "+postTitle)
	} else {
		// 本地文件的更新逻辑（如果需要）
		log.Printf("[DEBUG] 本地评论更新功能尚未实现")
		respondJSON(w, http.StatusNotImplemented, APIResponse{Success: false, Message: "本地评论更新功能尚未实现"})
		return
	}
	
	writeAuditLog("update_comment", r, map[string]interface{}{ "post_path": data.PostPath, "comment_id": data.CommentID, "issue_number": data.IssueNumber })
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "评论已更新"})
}

func handleGetAllComments(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问敏感数据
	if !requireLocal(w, r) {
		return
	}

	postPath := r.URL.Query().Get("path")
	if postPath == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}

	comments, err := getComments(postPath)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
		return
	}

	// Return all comments (for admin view)
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: comments})
}

// ==================== 访问统计数据结构 ====================
// VisitorIP 代表访客IP信息
type VisitorIP struct {
	IP           string `json:"ip"`              // 访客IP地址
	CommentCount int    `json:"comment_count"`   // 该IP的评论数
	FirstSeen    string `json:"first_seen"`      // 首次出现时间
	LastSeen     string `json:"last_seen"`       // 最后出现时间
}

// PageStatistics 代表单个页面的访问统计
type PageStatistics struct {
	Path  string `json:"path"`        // 页面路径
	Title string `json:"title"`       // 页面标题
	Views int    `json:"views"`       // 页面访问次数
	UV    int    `json:"uv,omitempty"` // 独立访客数（如果有的话）
}

// SiteStatistics 代表整个站点的统计信息
type SiteStatistics struct {
	TotalPages      int                  `json:"total_pages"`      // 总页面数
	TotalViews      int                  `json:"total_views"`      // 站点总访问量
	TotalComments   int                  `json:"total_comments"`   // 评论总数
	PendingComments int                  `json:"pending_comments"` // 待审核评论数
	UniqueIPs       int                  `json:"unique_ips"`       // 独立访客IP数
	Pages           []PageStatistics     `json:"pages,omitempty"`  // 页面统计列表
	Visitors        []VisitorIP          `json:"visitors,omitempty"` // 访客IP列表（仅管理员可见）
}

// ==================== 访问统计收集 ====================
// collectPageStatistics 收集所有页面的访问统计
// 这个函数会遍历所有发布的文章，统计它们的浏览量
func collectPageStatistics() []PageStatistics {
	var stats []PageStatistics
	posts := getPosts()
	
	// 遍历所有文章
	for _, post := range posts {
		// 仅统计已发布的页面
		if post.Status == "DRAFT" {
			continue
		}
		
		commentStats := getCommentStats(post.Path)
		
		pageStats := PageStatistics{
			Path:  post.Path,
			Title: post.Title,
			Views: commentStats["total"], // 暂时用评论数作为访问指标
		}
		stats = append(stats, pageStats)
	}
	
	// 按访问量排序（降序）
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Views > stats[j].Views
	})
	
	return stats
}

// collectSiteStatistics 收集整个站点的统计信息
func collectSiteStatistics() SiteStatistics {
    allStats := getAllCommentsStats()
    pageStats := collectPageStatistics()
	
    totalViews := 0
    for _, p := range pageStats {
        totalViews += p.Views
    }
	
    // 收集访客IP信息
    visitorMap := make(map[string]*VisitorIP)
    posts := getPosts()
	
    for _, post := range posts {
        comments, err := getComments(post.Path)
        if err != nil {
            continue
        }
		
        for _, comment := range comments {
            if comment.IPAddress != "" {
                if visitor, exists := visitorMap[comment.IPAddress]; exists {
                    visitor.CommentCount++
                    visitor.LastSeen = comment.Timestamp
                } else {
                    visitorMap[comment.IPAddress] = &VisitorIP{
                        IP:           comment.IPAddress,
                        CommentCount: 1,
                        FirstSeen:    comment.Timestamp,
                        LastSeen:     comment.Timestamp,
                    }
                }
            }
        }
    }
	
    // 转换为列表并排序（按评论数降序）
    visitors := make([]VisitorIP, 0, len(visitorMap))
    for _, v := range visitorMap {
        visitors = append(visitors, *v)
    }
    sort.Slice(visitors, func(i, j int) bool {
        return visitors[i].CommentCount > visitors[j].CommentCount
    })
	
    stats := SiteStatistics{
        TotalPages:      len(getPosts()),
        TotalViews:      totalViews,
        TotalComments:   allStats["total_comments"].(int),
        PendingComments: allStats["total_pending"].(int),
        UniqueIPs:       len(visitorMap),
        Pages:           pageStats[:func() int {
            if len(pageStats) > 10 {
                return 10 // 只返回前10个最常访问的页面
            }
            return len(pageStats)
        }()],
        Visitors: visitors[:func() int {
            if len(visitors) > 50 {
                return 50 // 只返回前50个访客IP
            }
            return len(visitors)
        }()],
    }
	
    return stats
}

// ==================== 统计API处理函数 ====================
// handleStatistics 处理访问统计请求
// 需要认证访问，防止数据泄露
func handleStatistics(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问或已认证的用户
	if !requireLocal(w, r) {
		return
	}
	
	stats := collectSiteStatistics()
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
}

// handlePageStatistics 处理单个页面的统计请求
func handlePageStatistics(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问或已认证的用户
	if !requireLocal(w, r) {
		return
	}
	
	path := r.URL.Query().Get("path")
	if path == "" {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Missing path parameter"})
		return
	}
	
	// 获取页面信息
	posts := getPosts()
	var pageInfo *Post
	for i := range posts {
		if posts[i].Path == path {
			pageInfo = &posts[i]
			break
		}
	}
	
	if pageInfo == nil {
		respondJSON(w, http.StatusNotFound, APIResponse{Success: false, Message: "Page not found"})
		return
	}
	
	// 获取该页面的统计
	commentStats := getCommentStats(path)
	
	pageStat := PageStatistics{
		Path:  pageInfo.Path,
		Title: pageInfo.Title,
		Views: commentStats["total"],
	}
	
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: pageStat})
}

func handleCommentStats(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问敏感数据
	if !requireLocal(w, r) {
		return
	}

	// 尝试从GitHub获取统计
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubRepo := "w2343419-del/WangScape"
	
	if githubToken != "" && githubRepo != "" {
		stats, err := getCommentStatsFromGitHub(githubRepo, githubToken)
		if err == nil {
			respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
			return
		}
	}

	// 降级：使用本地统计
	stats := getAllCommentsStats()
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
}

// getCommentStatsFromGitHub 从GitHub获取评论统计
func getCommentStatsFromGitHub(repo, token string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues?state=all&labels=comment&per_page=100", repo)
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "WSwriter")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var issues []struct {
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}
	
	total := len(issues)
	pending := 0
	approved := 0
	
	for _, issue := range issues {
		isPending := false
		isApproved := false
		
		for _, label := range issue.Labels {
			if label.Name == "pending" {
				isPending = true
			}
			if label.Name == "approved" {
				isApproved = true
			}
		}
		
		if isPending {
			pending++
		}
		if isApproved {
			approved++
		}
	}
	
	return map[string]interface{}{
		"total":    total,
		"pending":  pending,
		"approved": approved,
	}, nil
}

func handleGetPendingComments(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问敏感数据
	if !requireLocal(w, r) {
		return
	}

	var pendingComments []CommentWithPost
	
	// 优先从GitHub Issues获取待审核评论
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubRepo := "w2343419-del/WangScape"
	
	if githubToken != "" && githubRepo != "" {
		log.Printf("[DEBUG] 从GitHub获取待审核评论")
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues?state=open&labels=pending,comment&per_page=100", githubRepo)
		
		req, err := http.NewRequest("GET", apiURL, nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+githubToken)
			req.Header.Set("Accept", "application/vnd.github+json")
			req.Header.Set("User-Agent", "WSwriter")
			
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				
				if resp.StatusCode == 200 {
					var issues []struct {
						ID     int    `json:"id"`
						Number int    `json:"number"`
						Title  string `json:"title"`
						Body   string `json:"body"`
						State  string `json:"state"`
						CreatedAt string `json:"created_at"`
						Labels []struct {
							Name string `json:"name"`
						} `json:"labels"`
					}
					
					if json.NewDecoder(resp.Body).Decode(&issues) == nil {
						log.Printf("[DEBUG] GitHub返回 %d 条待审核Issues", len(issues))
						for _, issue := range issues {
							log.Printf("[DEBUG] Issue #%d - state: %s", issue.Number, issue.State)
						}
						for _, issue := range issues {
							body := issue.Body
							postPath := extractField(body, "Post")
							author := extractField(body, "Author")
							email := extractField(body, "Email")
							content := extractContent(body)
							
							// 调试日志
							log.Printf("[DEBUG] GitHub Issue #%d - Author: %s, Content length: %d", issue.Number, author, len(content))
							log.Printf("[DEBUG] Issue body preview: %s...", func() string {
								if len(body) > 200 {
									return body[:200]
								}
								return body
							}())
							
							// 检查是否有pending标签
							isPending := false
							for _, label := range issue.Labels {
								if label.Name == "pending" {
									isPending = true
									break
								}
							}
							
							if isPending {
								comment := Comment{
									ID:          fmt.Sprintf("%d", issue.ID),
									Author:      author,
									Email:       email,
									Content:     content,
									Timestamp:   issue.CreatedAt,
									Approved:    false,
									PostPath:    postPath,
									IssueNumber: issue.Number, // 保存Issue编号用于删除
								}
								
								pendingComments = append(pendingComments, CommentWithPost{
									Comment:   comment,
									PostTitle: issue.Title,
								})
							}
						}
						
						log.Printf("[DEBUG] 解析出 %d 条待审核评论", len(pendingComments))
						respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: pendingComments})
						return
					}
				}
			}
		}
		log.Printf("[DEBUG] GitHub获取失败，降级到本地存储")
	}

	// 降级：从本地comments.json读取
	contentRoot := filepath.Join(hugoPath, "content")
	filepath.Walk(contentRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}

		// 查找 comments.json 文件
		commentsPath := filepath.Join(path, "comments.json")
		if _, err := os.Stat(commentsPath); err == nil {
			comments, err := getComments(filepath.Join(path, "index.md"))
			if err == nil {
				// 获取文章标题
				indexPath := filepath.Join(path, "index.md")
				content, err := os.ReadFile(indexPath)
				if err == nil {
					fm := parseFrontmatter(string(content))
					for _, c := range comments {
						if !c.Approved {
							relPath, _ := filepath.Rel(hugoPath, indexPath)
							c.PostPath = relPath
							pendingComments = append(pendingComments, CommentWithPost{
								Comment:   c,
								PostTitle: fm.Title,
							})
						}
					}
				}
			}
		}
		return nil
	})

	// 按时间倒序排序
	sort.Slice(pendingComments, func(i, j int) bool {
		return pendingComments[i].Timestamp > pendingComments[j].Timestamp
	})

	respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: pendingComments})
}

func handleGetCommentSettings(w http.ResponseWriter, r *http.Request) {
    // 仅允许本地访问敏感配置
    if !requireLocal(w, r) {
        return
    }

    settings := loadCommentSettings()
    respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: settings})
}

func handleTestMail(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        respondJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
        return
    }

    // 仅允许本地访问
    if !requireLocal(w, r) {
        return
    }

    var data struct {
        SMTPHost string `json:"smtp_host"`
        SMTPPort int    `json:"smtp_port"`
        SMTPUser string `json:"smtp_user"`
        SMTPPass string `json:"smtp_pass"`
        SMTPFrom string `json:"smtp_from"`
        SMTPTo   string `json:"smtp_to"`
    }

    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }

    // 验证必要字段
    if data.SMTPHost == "" || data.SMTPPort == 0 || data.SMTPUser == "" || data.SMTPPass == "" || data.SMTPTo == "" {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "缺少必要的SMTP配置参数"})
        return
    }

    // 创建临时配置用于测试
    testSettings := CommentSettings{
        SMTPEnabled: true,
        SMTPHost:    data.SMTPHost,
        SMTPPort:    data.SMTPPort,
        SMTPUser:    data.SMTPUser,
        SMTPPass:    data.SMTPPass,
        SMTPFrom:    data.SMTPFrom,
        SMTPTo:      []string{data.SMTPTo},
    }

    // 创建测试邮件
    testComment := Comment{
        Author:    "测试用户",
        Email:     "test@example.com",
        Content:   "这是一封测试邮件，说明您的SMTP配置正确。",
        Timestamp: time.Now().Format("2006-01-02 15:04:05"),
        IPAddress: "127.0.0.1",
    }

    // 发送测试邮件
    if err := sendCommentNotification(testSettings, testComment, "测试文章"); err != nil {
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "邮件发送失败: " + err.Error()})
        log.Printf("[ERROR] Test email failed: %v", err)
        return
    }

    writeAuditLog("test_mail", r, map[string]interface{}{"smtp_host": data.SMTPHost})
    respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "测试邮件已发送，请检查收件箱"})
}

func handleSaveCommentSettings(w http.ResponseWriter, r *http.Request) {
    // 仅允许本地访问敏感配置
    if !requireLocal(w, r) {
        return
    }

    var settings CommentSettings
    if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }

    if err := saveCommentSettings(settings); err != nil {
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
        return
    }
    writeAuditLog("save_comment_settings", r, map[string]interface{}{"smtp_enabled": settings.SMTPEnabled})
    respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Saved"})
}

func handleBulkComments(w http.ResponseWriter, r *http.Request) {
    // 仅允许本地访问敏感操作
    if !requireLocal(w, r) {
        return
    }

    var data struct {
        Action string `json:"action"`
        Items  []struct {
            PostPath    string `json:"post_path"`
            CommentID   string `json:"comment_id"`
            IssueNumber int    `json:"issue_number,omitempty"`
        } `json:"items"`
    }

    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }

    if data.Action != "approve" && data.Action != "delete" {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid action"})
        return
    }

    githubToken := os.Getenv("GITHUB_TOKEN")
    githubRepo := "w2343419-del/WangScape"
    successCount := 0
    failCount := 0

    for _, item := range data.Items {
        if item.IssueNumber > 0 && githubToken != "" && githubRepo != "" {
            // 使用GitHub API批量操作
            if data.Action == "approve" {
                if err := updateGitHubIssueLabels(item.IssueNumber, githubRepo, githubToken, []string{"approved", "comment"}); err != nil {
                    log.Printf("[ERROR] 批量批准Issue #%d失败: %v", item.IssueNumber, err)
                    failCount++
                } else {
                    successCount++
                }
            } else {
                if err := deleteGitHubIssue(item.IssueNumber, githubRepo, githubToken); err != nil {
                    log.Printf("[ERROR] 批量删除Issue #%d失败: %v", item.IssueNumber, err)
                    failCount++
                } else {
                    successCount++
                }
            }
        } else {
            // 本地文件操作
            if data.Action == "approve" {
                if err := approveComment(item.PostPath, item.CommentID); err != nil {
                    failCount++
                } else {
                    successCount++
                }
            } else {
                if err := deleteComment(item.PostPath, item.CommentID); err != nil {
                    failCount++
                } else {
                    successCount++
                }
            }
        }
    }
    
    writeAuditLog("bulk_comments", r, map[string]interface{}{"action": data.Action, "count": len(data.Items), "success": successCount, "fail": failCount})
    
    var message string
    if failCount > 0 {
        message = fmt.Sprintf("批量操作完成：成功 %d 条，失败 %d 条", successCount, failCount)
    } else {
        message = fmt.Sprintf("批量操作成功：%d 条", successCount)
    }
    
    respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: message})
}

func handleExportComments(w http.ResponseWriter, r *http.Request) {
    // 仅允许本地访问敏感数据
    if !requireLocal(w, r) {
        return
    }

    comments, err := collectAllComments()
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
        return
    }

    writeAuditLog("export_comments", r, map[string]interface{}{"count": len(comments)})

    w.Header().Set("Content-Type", "text/csv; charset=utf-8")
    w.Header().Set("Content-Disposition", "attachment; filename=comments.csv")

    writer := csv.NewWriter(w)
    _ = writer.Write([]string{"post_path", "post_title", "id", "author", "email", "content", "timestamp", "approved", "ip_address", "user_agent", "parent_id"})
    for _, c := range comments {
        _ = writer.Write([]string{
            c.PostPath,
            c.PostTitle,
            c.ID,
            c.Author,
            c.Email,
            c.Content,
            c.Timestamp,
            strconv.FormatBool(c.Approved),
            c.IPAddress,
            c.UserAgent,
            c.ParentID,
        })
    }
    writer.Flush()
}

func handleCreateSync(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Title      string `json:"title"`
		Categories string `json:"categories"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	results, err := createSyncPost(data.Title, data.Categories)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
		return
	}
    writeAuditLog("create_sync_post", r, map[string]interface{}{ "title": data.Title, "categories": data.Categories })
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: results})
}

func handleCommandAPI(w http.ResponseWriter, r *http.Request) {
	// 仅允许本地访问敏感命令
	if !requireLocal(w, r) {
		return
	}

    // 限流：防止命令执行被滥用
    ip := getRealClientIP(r)
	if !allowRequest("command:"+ip, 10, time.Minute) {
		respondJSON(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "请求过于频繁"})
		return
	}

	cmd := r.URL.Query().Get("name")
	if cmd == "" {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Missing command"})
		return
	}

	// 限制命令为预定义的安全命令
	allowedCmds := map[string]bool{
		"preview": true,
		"deploy":  true,
		"build":   true,
		"sync":    true,
	}
	if !allowedCmds[cmd] {
		respondJSON(w, http.StatusForbidden, APIResponse{Success: false, Message: "Unknown command"})
		return
	}

	result, err := handleCommand(cmd)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
		return
	}
    writeAuditLog("command_exec", r, map[string]interface{}{ "command": cmd })
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
}

// handleSyncTranslate 先将中文markdown翻译为英文，然后同步到英文版本文章
// 接收中文文章和英文文章两个路径，翻译内容并同步格式
func handleSyncTranslate(w http.ResponseWriter, r *http.Request) {
	var data struct {
		ZhPath  string `json:"zhPath"`
		EnPath  string `json:"enPath"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

    // 读取英文版本（若不存在则在后续自动创建）
    enFullPath := filepath.Join(hugoPath, data.EnPath)
    enRaw, err := os.ReadFile(enFullPath)
    enExisting := ""
    if err == nil {
        enExisting = string(enRaw)
    } else if !os.IsNotExist(err) {
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: fmt.Sprintf("Failed to read English version: %v", err)})
        return
    }

    // 解析中文 frontmatter 和正文（格式异常时自动降级，避免同步中断）
    zhFrontmatter, zhBody, ok := splitMarkdownFrontmatter(data.Content)
    if !ok {
        log.Printf("[WARN] sync_translate: invalid markdown frontmatter, fallback to body-only translation: %s", data.ZhPath)
        zhFrontmatter = fmt.Sprintf("title: \"Untitled\"\ndate: %s\n", time.Now().Format(time.RFC3339))
        zhBody = data.Content
    }
    zhContentHash := computeSyncHash(data.Content)

    if enExisting != "" {
        if enFrontmatter, _, ok := splitMarkdownFrontmatter(enExisting); ok {
            if oldHash := getFrontmatterValue(enFrontmatter, "ws_sync_zh_hash"); oldHash != "" && oldHash == zhContentHash {
                respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "内容无变化，已跳过重复翻译"})
                return
            }
        }
    }

	// 翻译内容正文（保留代码块和特殊标记不翻译）
	translatedBody := translateMarkdownContent(zhBody, "zh", "en")

	// 生成英文版本的 frontmatter（翻译标题、描述、标签、分类）
	enFrontmatter := zhFrontmatter
	
	// 翻译title
	titleMatch := regexp.MustCompile(`title:\s*"?([^"\n]+)"?`).FindStringSubmatch(zhFrontmatter)
	if len(titleMatch) > 1 {
		zhTitle := titleMatch[1]
		enTitle := translateText(zhTitle, "zh", "en")
		enFrontmatter = regexp.MustCompile(`title:\s*"?[^"\n]+"?`).ReplaceAllString(enFrontmatter, fmt.Sprintf(`title: "%s"`, enTitle))
	}
	
	// 翻译description
	descMatch := regexp.MustCompile(`description:\s*"([^"]+)"`).FindStringSubmatch(zhFrontmatter)
	if len(descMatch) > 1 {
		zhDesc := descMatch[1]
		enDesc := translateText(zhDesc, "zh", "en")
		enFrontmatter = regexp.MustCompile(`description:\s*"[^"]+"`).ReplaceAllString(enFrontmatter, fmt.Sprintf(`description: "%s"`, enDesc))
	}
	
	// 翻译tags（数组形式）
	tagsRegex := regexp.MustCompile(`tags:\s*\n((?:\s+-\s+.+\n?)+)`)
	if tagsMatch := tagsRegex.FindStringSubmatch(zhFrontmatter); len(tagsMatch) > 1 {
		tagsList := tagsMatch[1]
		tagLines := strings.Split(strings.TrimSpace(tagsList), "\n")
		var translatedTags []string
		for _, line := range tagLines {
			tagValue := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if tagValue != "" {
				enTag := translateText(tagValue, "zh", "en")
				translatedTags = append(translatedTags, "    - "+enTag)
			}
		}
		if len(translatedTags) > 0 {
			enFrontmatter = tagsRegex.ReplaceAllString(enFrontmatter, "tags:\n"+strings.Join(translatedTags, "\n")+"\n")
		}
	}
	
	// 翻译categories（数组形式）
	catsRegex := regexp.MustCompile(`categories:\s*\n((?:\s+-\s+.+\n?)+)`)
	if catsMatch := catsRegex.FindStringSubmatch(zhFrontmatter); len(catsMatch) > 1 {
		catsList := catsMatch[1]
		catLines := strings.Split(strings.TrimSpace(catsList), "\n")
		var translatedCats []string
		for _, line := range catLines {
			catValue := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if catValue != "" {
				enCat := translateText(catValue, "zh", "en")
				translatedCats = append(translatedCats, "    - "+enCat)
			}
		}
		if len(translatedCats) > 0 {
			enFrontmatter = catsRegex.ReplaceAllString(enFrontmatter, "categories:\n"+strings.Join(translatedCats, "\n")+"\n")
		}
	}
	
    enFrontmatter = setFrontmatterValue(enFrontmatter, "ws_sync_zh_hash", zhContentHash)

	// 组装英文版本
	enContent := "---" + enFrontmatter + "---" + translatedBody

    // 确保英文目录存在（首次同步时自动创建）
    if err := os.MkdirAll(filepath.Dir(enFullPath), 0755); err != nil {
        respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: fmt.Sprintf("Failed to prepare directory: %v", err)})
        return
    }

	// 保存英文版本
	if err := os.WriteFile(enFullPath, []byte(enContent), 0644); err != nil {
		respondJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: fmt.Sprintf("Failed to save: %v", err)})
		return
	}
    writeAuditLog("sync_translate", r, map[string]interface{}{ "zh_path": data.ZhPath, "en_path": data.EnPath })
	respondJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Content translated and synced"})
}

// handleTranslateText 提供统一的文本翻译API（供前端评论翻译调用）
func handleTranslateText(w http.ResponseWriter, r *http.Request) {
    var data struct {
        Text       string `json:"text"`
        SourceLang string `json:"sourceLang"`
        TargetLang string `json:"targetLang"`
    }

    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }

    text := strings.TrimSpace(data.Text)
    if text == "" {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Text is required"})
        return
    }
    if len(text) > 5000 {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Text too long"})
        return
    }

    sourceLang := strings.TrimSpace(data.SourceLang)
    if sourceLang == "" {
        sourceLang = "auto"
    }

    targetLang := strings.TrimSpace(data.TargetLang)
    if targetLang == "" {
        respondJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Target language is required"})
        return
    }

    translated := translateText(text, sourceLang, targetLang)
    if strings.TrimSpace(translated) == "" || translated == text {
        respondJSON(w, http.StatusBadGateway, APIResponse{Success: false, Message: "Translation failed"})
        return
    }

    respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{
        "translated": translated,
    }})
}

func computeSyncHash(content string) string {
    normalized := strings.ReplaceAll(content, "\r\n", "\n")
    normalized = strings.TrimSpace(normalized)
    sum := sha256.Sum256([]byte(normalized))
    return hex.EncodeToString(sum[:])
}

func splitMarkdownFrontmatter(content string) (string, string, bool) {
    parts := strings.Split(content, "---")
    if len(parts) < 3 {
        return "", "", false
    }
    frontmatter := parts[1]
    body := strings.Join(parts[2:], "---")
    return frontmatter, body, true
}

func getFrontmatterValue(frontmatter, key string) string {
    pattern := fmt.Sprintf(`(?m)^%s:\s*"?([^"\n]+)"?\s*$`, regexp.QuoteMeta(key))
    match := regexp.MustCompile(pattern).FindStringSubmatch(frontmatter)
    if len(match) > 1 {
        return strings.TrimSpace(match[1])
    }
    return ""
}

func setFrontmatterValue(frontmatter, key, value string) string {
    pattern := fmt.Sprintf(`(?m)^%s:\s*"?[^"\n]*"?\s*$`, regexp.QuoteMeta(key))
    line := fmt.Sprintf(`%s: "%s"`, key, value)
    re := regexp.MustCompile(pattern)
    if re.MatchString(frontmatter) {
        return re.ReplaceAllString(frontmatter, line)
    }
    trimmed := strings.TrimRight(frontmatter, "\n")
    if trimmed == "" {
        return line + "\n"
    }
    return trimmed + "\n" + line + "\n"
}

// translateMarkdownContent 翻译Markdown体体制保“代码一块”不被翻译
func translateMarkdownContent(content, sourceLang, targetLang string) string {
	// 临时替换代码块
	codeBlocks := []string{}
	codeRegex := regexp.MustCompile("```[\\s\\S]*?```")
	content = codeRegex.ReplaceAllStringFunc(content, func(match string) string {
		codeBlocks = append(codeBlocks, match)
		return fmt.Sprintf("__CODE_BLOCK_%d__", len(codeBlocks)-1)
	})

	// 分段翻译（包括标题）
	paragraphs := strings.Split(content, "\n\n")
	for i, para := range paragraphs {
		trimmed := strings.TrimSpace(para)
		if len(trimmed) == 0 {
			continue
		}
		
		// 检查是否是Markdown标题（#开头）
		if strings.HasPrefix(trimmed, "#") {
			// 提取标题级别和内容
			headerMatch := regexp.MustCompile(`^(#+)\s+(.+)$`).FindStringSubmatch(trimmed)
			if len(headerMatch) > 2 {
				hashes := headerMatch[1]  // # 号的数量
				headerText := headerMatch[2]  // 标题文本
				translatedHeader := translateText(headerText, sourceLang, targetLang)
				paragraphs[i] = hashes + " " + translatedHeader
			}
		} else {
			// 普通段落直接翻译
			paragraphs[i] = translateText(para, sourceLang, targetLang)
		}
	}
	content = strings.Join(paragraphs, "\n\n")

	// 恢复代码块
	for i, block := range codeBlocks {
		placeholder := fmt.Sprintf("__CODE_BLOCK_%d__", i)
		content = strings.ReplaceAll(content, placeholder, block)
	}

	return content
}

// ==================== 标准化JSON响应 ====================
// respondJSON 返回标准化的JSON响应
// 会自动设置 Content-Type: application/json 头
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// ==================== 中间件：CORS跨域资源共享 ====================
// withCORS 为API端点添加CORS支持，支持跨域请求
// 实现了严格的安全策略：
//   1. Origin白名单验证（防止未授权网站访问）
//   2. 只允许本地访问和BASE_URL配置的domain
//   3. 预检请求(OPTIONS)自动处理
//   4. 返回多个安全响应头防止XSS和点击劫持
func withCORS(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 获取请求来源域名
        origin := r.Header.Get("Origin")
        if origin != "" {
            // 检查该源是否在白名单中
            if !isAllowedOrigin(origin, r) {
                // 拒绝不信任的origin，不暴露任何信息
                w.Header().Set("X-Frame-Options", "DENY")
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            // 只有白名单origin才允许跨域访问
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Vary", "Origin")
            
            // 严格的CORS策略
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Token")
            w.Header().Set("Access-Control-Max-Age", "3600")
            w.Header().Set("Access-Control-Allow-Credentials", "false")
            w.Header().Set("Access-Control-Expose-Headers", "Content-Length")
        }
        
        // 安全响应头：防止浏览器安全漏洞
        w.Header().Set("X-Content-Type-Options", "nosniff")          // 防止MIME嗅探攻击
        w.Header().Set("X-Frame-Options", "DENY")                    // 禁止iframe嵌入
        w.Header().Set("X-XSS-Protection", "1; mode=block")          // XSS防护
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin") // 限制referrer信息
        w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'") // 内容安全策略
        
        // 处理浏览器的跨域预检请求(CORS preflight)
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        
        handler(w, r)
    }
}

// ==================== CORS白名单验证 ====================
// isAllowedOrigin 检查请求源是否在CORS白名单中
// 支持多种配置方式：
//   1. 硬编码的本地开发地址
//   2. 当前请求的Host（防止同源策略问题）
//   3. BASE_URL环境变量配置的主地址
//   4. ALLOWED_ORIGINS环境变量的自定义列表（逗号分隔）
func isAllowedOrigin(origin string, r *http.Request) bool {
    // 本地开发白名单（Hugo预览和本应用）
    allowed := map[string]bool{
        "http://localhost:1313":  true,  // Hugo本地预览（开发）
        "http://127.0.0.1:1313": true,   // Hugo本地预览（回环地址）
        "http://localhost:8080":  true,  // 本应用（开发）
        "http://127.0.0.1:8080": true,   // 本应用（回环地址）
        "https://localhost:1313":  true, // Hugo本地预览（HTTPS）
        "https://127.0.0.1:1313": true,  // Hugo本地预览（HTTPS回环）
        "https://localhost:8080":  true, // 本应用（HTTPS）
        "https://127.0.0.1:8080": true,  // 本应用（HTTPS回环）
    }

    // 动态添加：允许与当前Host一致的来源（支持任意端口和域名）
    if r != nil {
        hostOriginHTTP := "http://" + r.Host
        hostOriginHTTPS := "https://" + r.Host
        allowed[hostOriginHTTP] = true
        allowed[hostOriginHTTPS] = true
    }

    // 允许 BASE_URL
    if baseURL := strings.TrimSpace(os.Getenv("BASE_URL")); baseURL != "" {
        allowed[baseURL] = true
    }

    // 允许自定义来源列表 ALLOWED_ORIGINS (逗号分隔)
    if raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")); raw != "" {
        parts := strings.Split(raw, ",")
        for _, p := range parts {
            p = strings.TrimSpace(p)
            if p != "" {
                allowed[p] = true
            }
        }
    }

    return allowed[origin]
}

func getClientIP(r *http.Request) string {
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress != "" {
		parts := strings.Split(ipAddress, ",")
		return strings.TrimSpace(parts[0])
	}
	if ipAddress = r.Header.Get("X-Real-IP"); ipAddress != "" {
		return ipAddress
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// ==================== 客户端住址判断 ====================
// isLocalRequest 判断是否是本地请求
// 本地请求可有更高的权限（如直接访问API而不需要认证）
func isLocalRequest(r *http.Request) bool {
    ip := getRealClientIP(r)
	// 判断是否是代办地址：127.0.0.1、localhost或IPv6的::1
	return ip == "127.0.0.1" || ip == "localhost" || ip == "::1"
}

// getEnv 获取环境变量，如果未设置则返回默认值
func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}

// ==================== 本地或认证管理 ====================
// requireLocal 检查是否允许本地或已认证的授权访问
// 用于保卸一些敏感操作，户菲超管也不能访问
func requireLocal(w http.ResponseWriter, r *http.Request) bool {
    // 优先检查是否是本地请求
    if isLocalRequest(r) {
        return true
    }
    // 仅胖陞超圣避坤不能陷忿对弟，也是可以非子搞得谈杭仪拦允许的朝代
    if requireAuth(w, r) {
        return true
    }
    return false
}

func requireAdminToken(r *http.Request) bool {
	if adminToken == "" {
		return true
	}
	token := r.Header.Get("X-Admin-Token")
	return token == adminToken
}

// ==================== 中间件：请求限流 ====================
// allowRequest 实现了基于时间窗口的请求限流（令牌桶算法的简化版本）
// 用于防止API被滥用或遭受DDoS攻击
// 参数说明：
//   key: 限流键，通常是 "操作:IP地址" 的组合，用于区分不同用户
//   limit: 时间窗口内允许的最大请求数
//   window: 时间窗口大小（如1分钟)
// 返回值：true表示请求被允许，false表示超过限流阈值
func allowRequest(key string, limit int, window time.Duration) bool {
    if limit <= 0 {
        return true  // 如果limit未配置或为0，不进行限流
    }
    
    now := time.Now()
    cutoff := now.Add(-window)  // 计算时间窗口的起始时刻

    rateLimiter.Lock()
    defer rateLimiter.Unlock()

    items := rateLimiter.records[key]
    
    // 过滤掉超时的记录，只保留在时间窗口内的请求记录
    filtered := items[:0]
    for _, t := range items {
        if t.After(cutoff) {
            filtered = append(filtered, t)
        }
    }
    
    // 检查是否达到限制
    if len(filtered) >= limit {
        rateLimiter.records[key] = filtered
        log.Printf("[RATE_LIMIT] Key=%s, Requests=%d, Limit=%d, Window=%v", key, len(filtered), limit, window)
        return false
    }
    
    // 添加新请求
    filtered = append(filtered, now)
    rateLimiter.records[key] = filtered
    
    // 定期清理过期记录（避免内存泄漏）
    if len(rateLimiter.records) > 10000 {
        // 清理所有过期的记录
        for k, v := range rateLimiter.records {
            newV := v[:0]
            for _, t := range v {
                if t.After(cutoff) {
                    newV = append(newV, t)
                }
            }
            if len(newV) == 0 {
                delete(rateLimiter.records, k)
            } else {
                rateLimiter.records[k] = newV
            }
        }
    }
    
    return true
}

// openBrowser 打开系统默认浏览器
func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Run()
	case "linux":
		exec.Command("xdg-open", url).Run()
	case "windows":
		exec.Command("cmd", "/c", "start", url).Run()
	}
}

// ==================== Hugo服务器启动（自动化预览） ====================
// startHugoServer 启动Hugo静态站点生成器并解析其输出来获取正确的访问地址
// 返回Hugo server的实际URL（如 http://localhost:14746/WangScape/ ）
// 该URL会被写入 .vscode/hugo-url.txt 供VS Code Live Preview扩展使用
func startHugoServer() string {
	tempHugoURL := ""
	hugoPathExec := "hugo" // 使用PATH中的hugo命令

	// 启动Hugo子进程，读取其输出
	cmd := exec.Command(hugoPathExec, "server", "-D")
	cmd.Dir = "." // 在当前目录执行

	// 创建输出管道并捕获stdout
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[WARN] Failed to capture Hugo stdout: %v", err)
		return ""
	}

	// 捕获stderr用于错误诊断
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("[WARN] Failed to capture Hugo stderr: %v", err)
		return ""
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		log.Printf("[WARN] Failed to start Hugo server: %v", err)
		return ""
	}

	// 在后台线程中读取输出
	go func() {
		scanner := bufio.NewScanner(outPipe)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("[HUGO] %s", line)

			// 解析Hugo输出中的URL信息
			// 输出格式: "Web Server is available at http://localhost:14746/WangScape/ (bind address 127.0.0.1)"
			if strings.Contains(line, "Web Server is available at") {
				parts := strings.Split(line, "Web Server is available at ")
				if len(parts) > 1 {
					urlPart := strings.Fields(parts[1])[0]
					tempHugoURL = urlPart
					hugoURL = urlPart  // 更新全局变量
					log.Printf("[HUGO-URL] Detected: %s", hugoURL)

					// 将URL写入文件，供VS Code Live Preview扩展读取
					urlFile := filepath.Join(".vscode", "hugo-url.txt")
					if err := os.WriteFile(urlFile, []byte(hugoURL), 0644); err != nil {
						log.Printf("[WARN] Failed to write Hugo URL to file: %v", err)
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("[HUGO-ERROR] stdout scan error: %v", err)
		}
	}()

	// 在另一个后台线程中读取stderr
	go func() {
		scanner := bufio.NewScanner(errPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				log.Printf("[HUGO-WARN] %s", line)
			}
		}
	}()

	log.Printf("[Hugo] Server started (PID: %d)", cmd.Process.Pid)
	return tempHugoURL
}

// ==================== 请求体大小限制 ====================
// limitRequestBody 中间件工厂：限制HTTP请求体大小，防止攻击者突破服务器
// 当做手为中离砲嗎者超过限制时，直接中断连接，防止客户端继续虚送数据
func limitRequestBody(h http.HandlerFunc, maxSize int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置http.MaxBytesReader限制请求体大小（不是仅什么可以算了）
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)
		h(w, r)
	}
}

// loadEnvFile 从.env文件加载环境变量
func loadEnvFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		// .env文件不存在时忽略，使用系统环境变量
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 解析 KEY=VALUE 格式
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// 移除引号（如果有）
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		
		// 设置环境变量（仅当未设置时）
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// ==================== 程序入口点 ====================
// main 是应用程序的主函数，负责初始化配置、设置路由和启动HTTP/HTTPS服务器
func main() {
	// 加载.env文件中的环境变量（用于开发环境配置）
	loadEnvFile(".env")
	
	// 启动邮件工作线程（P1优化：支持重试机制）
	startEmailWorkers()

	// 启动Hugo服务器（自动化预览）
	log.Println("[STARTUP] Starting Hugo server...")
	hugoURL := startHugoServer()
	if hugoURL != "" {
		log.Printf("[STARTUP] Hugo server URL: %s", hugoURL)
	} else {
		log.Printf("[WARN] Hugo server URL not detected, Live Preview may not work automatically")
	}
	
	// 调试：打印加载的凭据
	adminUser := os.Getenv("ADMIN_USERNAME")
	adminPass := os.Getenv("ADMIN_PASSWORD")
	githubToken := os.Getenv("GITHUB_TOKEN")
	if adminUser != "" || adminPass != "" {
		log.Printf("[AUTH] 凭据已加载 - User: %s, Password: %s", adminUser, "***")
	}
	if githubToken != "" {
		log.Printf("[AUTH] GITHUB_TOKEN已加载，长度: %d", len(githubToken))
	} else {
		log.Printf("[WARN] GITHUB_TOKEN未设置，无法从GitHub读取评论")
	}
	
	// ==================== 安全中间件设置 ====================
	
	// 添加HSTS和其他安全头
	hstsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// HSTS: 强制HTTPS连接 (1年有效期)
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			// 防止MIME嗅探
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// XSS防护
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			// 禁用iframe嵌入
			w.Header().Set("X-Frame-Options", "DENY")
			// 限制特性权限
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			// 内容安全策略（防XSS和点击劫持）
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://fonts.googleapis.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com")
			
			next.ServeHTTP(w, r)
		})
	}
	
	// 创建根mux并包装HSTS中间件
	rootMux := http.NewServeMux()
	wrappedMux := hstsMiddleware(rootMux)
	
	// 注册所有API路由
	rootMux.HandleFunc("/", handleIndex)
    rootMux.HandleFunc("/api/hugo-url", withCORS(handleHugoURL))
    rootMux.HandleFunc("/api/login", withCORS(limitRequestBody(handleLogin, 4<<10)))
    rootMux.HandleFunc("/api/refresh-token", withCORS(limitRequestBody(handleRefreshToken, 4<<10)))
    rootMux.HandleFunc("/api/posts", withCORS(handleGetPosts))
    rootMux.HandleFunc("/api/get_content", withCORS(withAuth(handleGetContent)))
    rootMux.HandleFunc("/api/save_content", withCORS(withAuth(limitRequestBody(handleSaveContent, 2<<20))))
    rootMux.HandleFunc("/api/delete_post", withCORS(withAuth(limitRequestBody(handleDeletePost, 1<<20))))
    rootMux.HandleFunc("/api/create_sync", withCORS(withAuth(limitRequestBody(handleCreateSync, 5<<20))))
    rootMux.HandleFunc("/api/sync_translate", withCORS(withAuth(limitRequestBody(handleSyncTranslate, 5<<20))))
    rootMux.HandleFunc("/api/translate_text", withCORS(limitRequestBody(handleTranslateText, 16<<10)))
    rootMux.HandleFunc("/api/command", withCORS(withAuth(limitRequestBody(handleCommandAPI, 512))))
    rootMux.HandleFunc("/api/comments", withCORS(handleGetComments))
    rootMux.HandleFunc("/api/add_comment", withCORS(limitRequestBody(handleAddComment, 1<<20)))
    rootMux.HandleFunc("/api/upload_comment_image", withCORS(limitRequestBody(handleUploadCommentImage, 12<<20)))
    rootMux.HandleFunc("/api/approve_comment", withCORS(withAuth(limitRequestBody(handleApproveComment, 512))))
    rootMux.HandleFunc("/api/delete_comment", withCORS(withAuth(limitRequestBody(handleDeleteComment, 512))))
    rootMux.HandleFunc("/api/update_comment", withCORS(withAuth(limitRequestBody(handleUpdateComment, 2<<20))))
    rootMux.HandleFunc("/api/all_comments", withCORS(withAuth(handleGetAllComments)))
    rootMux.HandleFunc("/api/comment_stats", withCORS(withAuth(handleCommentStats)))
    rootMux.HandleFunc("/api/statistics", withCORS(withAuth(handleStatistics)))
    rootMux.HandleFunc("/api/page_statistics", withCORS(withAuth(handlePageStatistics)))
    rootMux.HandleFunc("/api/pending_comments", withCORS(withAuth(handleGetPendingComments)))
    rootMux.HandleFunc("/api/comment_settings", withCORS(withAuth(handleGetCommentSettings)))
    rootMux.HandleFunc("/api/save_comment_settings", withCORS(withAuth(limitRequestBody(handleSaveCommentSettings, 1<<20))))
    rootMux.HandleFunc("/api/test_mail", withCORS(withAuth(limitRequestBody(handleTestMail, 1<<20))))
    rootMux.HandleFunc("/api/bulk_comments", withCORS(withAuth(limitRequestBody(handleBulkComments, 1<<20))))
    rootMux.HandleFunc("/api/export_comments", withCORS(withAuth(handleExportComments)))

	// 启动审计日志轮转
	go rotateAuditLogPeriodically()

	// 获取端口配置
    httpHost := getEnv("HTTP_HOST", "127.0.0.1")
    httpsHost := getEnv("HTTPS_HOST", httpHost)
	httpPort := getEnv("HTTP_PORT", "8080")
	httpsPort := getEnv("HTTPS_PORT", "443")
	tlsCertFile := getEnv("TLS_CERT_FILE", "")
	tlsKeyFile := getEnv("TLS_KEY_FILE", "")

	// 启动HTTP服务器监听
    openHost := httpHost
    if openHost == "0.0.0.0" || openHost == "::" {
        openHost = "127.0.0.1"
    }
    fmt.Printf("WangScape Writer Online: http://%s:%s\n", openHost, httpPort)
    openBrowser(fmt.Sprintf("http://%s:%s", openHost, httpPort))

	// 启动HTTP监听
	go func() {
        httpAddr := fmt.Sprintf("%s:%s", httpHost, httpPort)
		if err := http.ListenAndServe(httpAddr, wrappedMux); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[AUDIT] HTTP Server error: %v\n", err)
		}
	}()

	// 启动HTTPS监听 (如果配置了证书)
	if tlsCertFile != "" && tlsKeyFile != "" {
		if _, err := os.Stat(tlsCertFile); err == nil {
			if _, err := os.Stat(tlsKeyFile); err == nil {
				go func() {
                    httpsAddr := fmt.Sprintf("%s:%s", httpsHost, httpsPort)
					fmt.Printf("[AUDIT] HTTPS Server starting on %s\n", httpsAddr)
					if err := http.ListenAndServeTLS(httpsAddr, tlsCertFile, tlsKeyFile, wrappedMux); err != nil && err != http.ErrServerClosed {
						fmt.Printf("[AUDIT] HTTPS Server error: %v\n", err)
					}
				}()
			} else {
				fmt.Printf("[AUDIT] TLS key file not found: %s\n", tlsKeyFile)
			}
		} else {
			fmt.Printf("[AUDIT] TLS cert file not found: %s\n", tlsCertFile)
		}
	} else {
		fmt.Printf("[AUDIT] HTTPS not configured (set TLS_CERT_FILE and TLS_KEY_FILE to enable)\n")
	}

	// 持续运行
	select {}
}

var htmlTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WangScape Writer</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=Sitka+Small&family=Noto+Sans+SC:wght@300;400;500;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --dash-bg: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            --dash-sidebar: #1e293b;
            --dash-text: #0f172a;
            --dash-text-dim: #64748b;
            --dash-accent: #6366f1;
            --dash-border: #e2e8f0;
            --word-bg: #f1f5f9;
            --word-blue: #3b82f6;
            --word-paper: #ffffff;
            --word-text: #0f172a;
            --word-border: #e2e8f0;
            --font-main: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Microsoft YaHei', sans-serif;
            --success: #10b981;
            --warning: #f59e0b;
            --danger: #ef4444;
            --info: #3b82f6;
        }

        * {
            box-sizing: border-box;
        }
        
        body {
            margin: 0;
            font-family: var(--font-main);
            overflow: hidden;
            height: 100vh;
            background: #f8fafc;
        }
        
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        
        @keyframes slideIn {
            from { opacity: 0; transform: translateX(-20px); }
            to { opacity: 1; transform: translateX(0); }
        }
        
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        .view-section {
            display: none;
            width: 100%;
            height: 100%;
        }

        .view-section.active {
            display: flex;
        }

        #dashboard-view {
            background: var(--dash-bg);
            color: var(--dash-text);
        }

        .dash-sidebar {
            width: 280px;
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            border-right: 1px solid #e2e8f0;
            padding: 32px 24px;
            display: flex;
            flex-direction: column;
            gap: 16px;
            box-shadow: 4px 0 24px rgba(0, 0, 0, 0.06);
            --dash-text: #0f172a;
            --dash-text-dim: #64748b;
            position: relative;
            z-index: 10;
        }
        
        .dash-sidebar::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 200px;
            background: linear-gradient(180deg, rgba(99, 102, 241, 0.05) 0%, transparent 100%);
            pointer-events: none;
        }

        .dash-logo {
            font-size: 22px;
            font-weight: 800;
            background: linear-gradient(135deg, #a5b4fc 0%, #4f46e5 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 10px;
            letter-spacing: 0.2px;
        }

        .dash-btn {
            background: rgba(99, 102, 241, 0.05);
            border: 1px solid #e2e8f0;
            color: #0f172a;
            padding: 12px 16px;
            border-radius: 10px;
            cursor: pointer;
            text-align: left;
            font-size: 14px;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            display: flex;
            align-items: center;
            gap: 10px;
            font-weight: 500;
            position: relative;
            overflow: hidden;
        }
        
        .dash-btn::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(99, 102, 241, 0.1), transparent);
            transition: left 0.5s;
        }
        
        .dash-btn:hover::before {
            left: 100%;
        }

        .dash-btn:hover {
            border-color: #c7d2fe;
            background: rgba(99, 102, 241, 0.12);
            color: #4338ca;
            transform: translateX(4px);
        }
        
        .dash-btn:active {
            transform: scale(0.98);
        }

        .dash-btn.primary {
            background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
            color: #ffffff;
            border: none;
            font-weight: 600;
            box-shadow: 0 4px 20px rgba(99, 102, 241, 0.4), inset 0 1px 0 rgba(255,255,255,0.2);
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            position: relative;
        }
        
        .dash-btn.primary::after {
            content: '';
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            width: 0;
            height: 0;
            border-radius: 50%;
            background: rgba(255,255,255,0.3);
            transition: width 0.6s, height 0.6s;
        }
        
        .dash-btn.primary:hover::after {
            width: 300px;
            height: 300px;
        }

        .dash-btn.primary:hover {
            transform: translateY(-2px) translateX(0);
            box-shadow: 0 8px 30px rgba(99, 102, 241, 0.5), inset 0 1px 0 rgba(255,255,255,0.3);
        }

        .dash-main {
            flex: 1;
            padding: 44px 56px;
            overflow-y: auto;
            background: linear-gradient(180deg, #f8fafc 0%, #ffffff 65%);
            color: var(--dash-text);
        }

        .dash-header {
            font-size: 28px;
            font-weight: 800;
            margin-bottom: 28px;
            letter-spacing: -0.2px;
            color: #0f172a;
        }

        .post-list-card {
            background: #ffffff;
            border-radius: 16px;
            border: 1px solid #e2e8f0;
            overflow: hidden;
            box-shadow: 0 4px 20px rgba(15, 23, 42, 0.08);
            transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
            animation: fadeIn 0.5s ease;
        }

        .post-list-card:hover {
            box-shadow: 0 12px 40px rgba(99, 102, 241, 0.15);
            border-color: #a5b4fc;
            transform: translateY(-4px);
        }

        .dash-post-item {
            padding: 22px 26px;
            border-bottom: 1px solid #f1f5f9;
            display: flex;
            justify-content: space-between;
            align-items: center;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .dash-post-item:hover {
            background: linear-gradient(135deg, #f8fafc 0%, #eef2ff 100%);
        }

        .dash-post-item:last-child {
            border-bottom: none;
        }

        .dpi-title {
            font-size: 15px;
            font-weight: 600;
            color: #0f172a;
            margin-bottom: 5px;
        }

        .dpi-meta {
            font-size: 12px;
            color: #64748b;
            font-family: 'Inter', 'Noto Sans SC', sans-serif;
        }

        #editor-view {
            background: var(--word-bg);
            color: var(--word-text);
            flex-direction: column;
        }

        .word-topbar {
            background: linear-gradient(135deg, #4f46e5 0%, #2563eb 100%);
            color: white;
            height: 54px;
            display: flex;
            align-items: center;
            padding: 0 20px;
            justify-content: space-between;
            box-shadow: 0 6px 16px rgba(37, 99, 235, 0.25);
        }

        .word-back-btn {
            background: rgba(255,255,255,0.15);
            border: 1px solid rgba(255,255,255,0.3);
            color: white;
            padding: 8px 14px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 13px;
            display: flex;
            align-items: center;
            gap: 6px;
            font-weight: 600;
            transition: all 0.2s ease;
        }

        .word-back-btn:hover {
            background: rgba(255,255,255,0.25);
            transform: translateX(-2px);
        }

        .word-ribbon {
            background: #ffffff;
            border-bottom: 1px solid #e2e8f0;
            padding: 14px 20px;
            display: flex;
            gap: 10px;
            box-shadow: 0 2px 6px rgba(15, 23, 42, 0.04);
        }

        .word-rib-btn {
            border: 1px solid #e2e8f0;
            background: #f8fafc;
            padding: 10px 14px;
            border-radius: 10px;
            cursor: pointer;
            font-size: 12px;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 5px;
            color: #0f172a;
            transition: all 0.2s ease;
            position: relative;
            font-weight: 600;
        }
        
        .word-rib-btn span:first-child {
            font-size: 18px;
        }

        .word-rib-btn:hover {
            background: #4f46e5;
            border-color: #4338ca;
            color: white;
            transform: translateY(-2px);
            box-shadow: 0 6px 16px rgba(79, 70, 229, 0.35);
        }
        
        .word-rib-btn:active {
            transform: translateY(0);
        }

        .word-workspace {
            flex: 1;
            display: flex;
            overflow: hidden;
        }

        .word-canvas {
            flex: 1;
            background: linear-gradient(135deg, #eef2f7 0%, #ffffff 100%);
            padding: 36px;
            overflow-y: auto;
            display: flex;
            justify-content: center;
            gap: 24px;
            align-items: flex-start;
            max-width: 100%;
        }

        .word-paper {
            width: 800px;
            max-width: 800px;
            flex-shrink: 0;
            min-height: calc(100vh - 200px);
            background: white;
            box-shadow: 0 16px 36px rgba(15, 23, 42, 0.12), 0 2px 6px rgba(15, 23, 42, 0.08);
            padding: 56px 72px;
            box-sizing: border-box;
            display: flex;
            flex-direction: column;
            border-radius: 16px;
            position: relative;
        }
        

        .meta-panel {
            width: 360px;
            min-width: 360px;
            max-width: 360px;
            flex-shrink: 0;
            background: #ffffff;
            box-shadow: 0 12px 24px rgba(15, 23, 42, 0.1);
            padding: 26px;
            box-sizing: border-box;
            border-radius: 16px;
            max-height: calc(100vh - 200px);
            overflow-y: auto;
            position: sticky;
            top: 30px;
            border: 1px solid #e2e8f0;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        }
        
        .meta-panel:hover {
            box-shadow: 0 16px 32px rgba(15, 23, 42, 0.14);
            border-color: #cbd5f5;
        }

        #comments-panel {
            width: 360px !important;
            min-width: 360px !important;
            max-width: 360px !important;
            flex-shrink: 0 !important;
            background: white !important;
            box-shadow: 0 12px 24px rgba(15, 23, 42, 0.1) !important;
            padding: 24px !important;
            box-sizing: border-box !important;
            border-radius: 16px !important;
            max-height: calc(100vh - 200px) !important;
            overflow-y: auto !important;
            position: sticky !important;
            top: 30px !important;
            border: 1px solid #e2e8f0 !important;
            border-left: 4px solid #f59e0b !important;
            transition: all 0.3s ease !important;
        }

        #comments-panel.show {
            display: block !important;
        }

        #comments-panel.hide {
            display: none !important;
        }

        .meta-panel h3 {
            margin: 0 0 22px 0;
            font-size: 16px;
            font-weight: 700;
            color: #1a1a1a;
            border-bottom: 2px solid #4a90e2;
            padding-bottom: 12px;
            letter-spacing: 0.5px;
        }

        .meta-section {
            margin-bottom: 22px;
        }

        .meta-section label {
            display: block;
            font-size: 12px;
            color: #666;
            margin-bottom: 8px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .meta-input {
            width: 100%;
            padding: 12px 14px;
            border: 1.5px solid #dfe3ec;
            border-radius: 8px;
            font-size: 13px;
            box-sizing: border-box;
            font-family: var(--font-main);
            background: #fafbfc;
            color: #2c3e50;
            transition: all 0.2s ease;
        }

        .meta-input:focus {
            outline: none;
            border-color: #4a90e2;
            background: #ffffff;
            box-shadow: 0 0 0 3px rgba(74, 144, 226, 0.1);
        }

        .meta-input:hover {
            border-color: #c5d0e0;
        }

        .tag-container {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            margin-bottom: 12px;
            min-height: 44px;
            padding: 12px 14px;
            border: 1.5px solid #dfe3ec;
            border-radius: 8px;
            background: linear-gradient(135deg, #fafbfc 0%, #f5f7fb 100%);
            align-content: flex-start;
        }

        .tag-item {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            background: linear-gradient(135deg, #4a90e2 0%, #2e5bad 100%);
            color: white;
            padding: 7px 14px;
            border-radius: 18px;
            font-size: 12px;
            font-weight: 600;
            box-shadow: 0 2px 6px rgba(74, 144, 226, 0.25);
            transition: all 0.2s ease;
        }

        .tag-item:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 10px rgba(74, 144, 226, 0.35);
        }

        .tag-remove {
            cursor: pointer;
            font-weight: bold;
            font-size: 14px;
            opacity: 0.8;
        }

        .tag-remove:hover {
            opacity: 1;
        }

        .tag-input-row {
            display: flex;
            gap: 8px;
        }

        .tag-input-row input {
            flex: 1;
            padding: 10px 12px;
            border: 1.5px solid #dfe3ec;
            border-radius: 8px;
            font-size: 13px;
            background: #fafbfc;
            color: #2c3e50;
            transition: all 0.2s ease;
        }

        .tag-input-row input:focus {
            outline: none;
            border-color: #4a90e2;
            background: #ffffff;
            box-shadow: 0 0 0 3px rgba(74, 144, 226, 0.1);
        }

        .tag-input-row button {
            padding: 10px 16px;
            background: linear-gradient(135deg, #4a90e2 0%, #2e5bad 100%);
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 12px;
            font-weight: 600;
            transition: all 0.2s ease;
            box-shadow: 0 2px 8px rgba(74, 144, 226, 0.2);
        }

        .tag-input-row button:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(74, 144, 226, 0.35);
        }

        .tag-input-row button:active {
            transform: translateY(0);
        }

        .meta-checkbox {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-top: 10px;
            padding: 10px 12px;
            border-radius: 8px;
            background: #f5f7fb;
            transition: all 0.2s ease;
        }

        .meta-checkbox:hover {
            background: #eff3fb;
        }

        .meta-checkbox input[type="checkbox"] {
            width: 18px;
            height: 18px;
            cursor: pointer;
            accent-color: #4a90e2;
        }

        .meta-checkbox label {
            margin: 0;
            cursor: pointer;
            font-size: 13px;
            color: #2c3e50;
            font-weight: 500;
        }

        .wp-title {
            font-family: 'Sitka Small', serif;
            font-size: 36px;
            font-weight: 800;
            background: linear-gradient(135deg, #1a1a1a 0%, #333333 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            border-bottom: 2px solid #e8eef5;
            padding-bottom: 22px;
            margin-bottom: 32px;
            letter-spacing: -0.5px;
        }

        .modal-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(15, 23, 42, 0.7);
            backdrop-filter: blur(8px);
            display: none;
            align-items: center;
            justify-content: center;
            z-index: 2000;
        }

        .modal-card {
            background: white;
            color: #0f172a;
            width: 500px;
            padding: 30px;
            border-radius: 16px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 20px 60px rgba(15, 23, 42, 0.3);
        }

        .modal-card h2 {
            color: #0f172a;
            font-weight: 700;
        }

        .modal-card label {
            color: #475569;
            font-weight: 500;
            font-size: 14px;
        }

        .modal-card input {
            width: 100%;
            padding: 12px;
            background: #f8fafc;
            border: 1px solid #cbd5e1;
            color: #0f172a;
            border-radius: 8px;
            margin-top: 8px;
            margin-bottom: 20px;
            box-sizing: border-box;
            transition: all 0.2s;
        }

        .modal-card input:focus {
            outline: none;
            border-color: #8b5cf6;
            background: white;
            box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.1);
        }

        .modal-card button {
            padding: 10px 20px;
            border-radius: 8px;
            cursor: pointer;
            font-weight: 600;
            transition: all 0.2s;
            border: none;
        }

        .btn-confirm {
            background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
            color: white;
            border: none;
        }

        .btn-cancel {
            background: #f1f5f9;
            color: #475569;
            border: 1px solid #cbd5e1;
            margin-right: 10px;
        }

        .btn-cancel:hover {
            background: #e2e8f0;
            border-color: #94a3b8;
        }

        .btn-confirm:hover {
            background: linear-gradient(135deg, #7c3aed 0%, #6d28d9 100%);
            box-shadow: 0 4px 12px rgba(139, 92, 246, 0.4);
        }

        #editor-textarea {
            width: 100%;
            height: 100%;
            min-height: 600px;
            border: 1.5px solid #dfe3ec;
            resize: none;
            outline: none;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
            font-size: 15px;
            line-height: 1.8;
            color: #2c3e50;
            padding: 20px;
            tab-size: 4;
            border-radius: 12px;
            background: #ffffff;
            box-shadow: inset 0 2px 4px rgba(0,0,0,0.02);
            transition: all 0.2s ease;
        }
        
        #editor-textarea:focus {
            border-color: #4a90e2;
            box-shadow: inset 0 2px 4px rgba(0,0,0,0.02), 0 0 0 3px rgba(74, 144, 226, 0.1);
        }
        
        #editor-textarea::selection {
            background: #4f46e5;
            color: #ffffff;
        }
        
        #editor-textarea::-moz-selection {
            background: #4f46e5;
            color: #ffffff;
        }
        
        .pending-comment-card {
            background: #ffffff;
            border: 1px solid #e5e7eb;
            border-left: 4px solid #f59e0b;
            border-radius: 12px;
            padding: 0;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
            margin-bottom: 16px;
            overflow: hidden;
        }
        
        .pending-comment-card:hover {
            box-shadow: 0 8px 24px rgba(245, 158, 11, 0.15);
            border-left-color: #f97316;
            transform: translateY(-2px);
        }
        
        .comment-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 16px 20px;
            background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
            border-bottom: 1px solid #fde047;
        }
        
        .comment-number {
            font-size: 14px;
            font-weight: 700;
            color: #92400e;
            background: #fbbf24;
            padding: 4px 12px;
            border-radius: 20px;
        }
        
        .pending-select {
            width: 18px;
            height: 18px;
            cursor: pointer;
        }
        
        .comment-post-title {
            font-size: 14px;
            color: #6366f1;
            margin: 0;
            padding: 16px 20px 12px;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .comment-user-info {
            display: flex;
            align-items: center;
            gap: 14px;
            padding: 0 20px 16px;
        }
        
        .user-avatar {
            width: 48px;
            height: 48px;
            border-radius: 50%;
            background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 20px;
            font-weight: 700;
            flex-shrink: 0;
            box-shadow: 0 2px 8px rgba(99, 102, 241, 0.3);
        }
        
        .user-details {
            flex: 1;
            min-width: 0;
        }
        
        .comment-meta {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-top: 4px;
            font-size: 12px;
            color: #9ca3af;
            flex-wrap: wrap;
        }
        
        .meta-item {
            display: flex;
            align-items: center;
            gap: 4px;
        }
        
        .comment-author {
            font-weight: 700;
            color: #111827;
            font-size: 16px;
            margin-bottom: 4px;
        }
        
        .comment-content-preview {
            color: #374151;
            line-height: 1.7;
            margin: 0;
            padding: 20px;
            background: #f9fafb;
            font-size: 14px;
            border-top: 1px solid #e5e7eb;
            border-bottom: 1px solid #e5e7eb;
            min-height: 60px;
        }
        
        .comment-tech-info {
            font-size: 12px;
            color: #6b7280;
            padding: 16px 20px;
            background: #fafafa;
            display: grid;
            grid-template-columns: 1fr;
            gap: 8px;
        }
        
        .tech-item {
            display: flex;
            align-items: center;
            gap: 6px;
            line-height: 1.5;
        }
        
        .tech-item strong {
            color: #374151;
            min-width: 30px;
        }
        
        .tech-ua {
            word-break: break-all;
        }
        
        .comment-actions {
            display: flex;
            gap: 10px;
            padding: 16px 20px;
            background: #ffffff;
        }
        
        .comment-actions {
            display: flex;
            gap: 10px;
        }
        
        .btn-approve {
            padding: 10px 20px;
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 13px;
            font-weight: 600;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            box-shadow: 0 2px 8px rgba(16, 185, 129, 0.25);
        }
        
        .btn-approve:hover {
            background: linear-gradient(135deg, #059669 0%, #047857 100%);
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(16, 185, 129, 0.35);
        }
        
        .btn-edit {
            padding: 10px 20px;
            background: linear-gradient(135deg, #0ea5e9 0%, #0284c7 100%);
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 13px;
            font-weight: 600;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            box-shadow: 0 2px 8px rgba(14, 165, 233, 0.25);
        }
        
        .btn-edit:hover {
            background: linear-gradient(135deg, #0284c7 0%, #0369a1 100%);
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(14, 165, 233, 0.35);
        }
        
        .btn-delete {
            padding: 10px 20px;
            background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 13px;
            font-weight: 600;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            box-shadow: 0 2px 8px rgba(239, 68, 68, 0.25);
        }
        
        .btn-delete:hover {
            background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(239, 68, 68, 0.35);
        }
        
        .comment-tech-info {
            font-size: 12px;
            color: #9ca3af;
            margin-top: 14px;
            padding-top: 14px;
            border-top: 1px solid #e5e7eb;
            line-height: 1.6;
        }
        
        .comment-tech-info div {
            margin: 4px 0;
        }

        .pending-toolbar {
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 12px;
            margin-bottom: 24px;
            padding: 16px 20px;
            background: white;
            border-radius: 12px;
            border: 1px solid #e5e7eb;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
        }

        .pending-toolbar label {
            font-size: 13px;
            color: var(--dash-text);
        }
        
        .btn-approve-bulk, .btn-delete-bulk {
            padding: 10px 20px;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
            transition: all 0.2s;
            box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
        }
        
        .btn-approve-bulk {
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
            color: white;
        }
        
        .btn-approve-bulk:hover {
            background: linear-gradient(135deg, #059669 0%, #047857 100%);
            transform: translateY(-1px);
            box-shadow: 0 4px 10px rgba(16, 185, 129, 0.3);
        }
        
        .btn-delete-bulk {
            background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
            color: white;
        }
        
        .btn-delete-bulk:hover {
            background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
            transform: translateY(-1px);
            box-shadow: 0 4px 10px rgba(239, 68, 68, 0.3);
        }

        .settings-panel {
            margin-top: 20px;
            padding: 15px;
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid rgba(255, 255, 255, 0.08);
            border-radius: 10px;
        }

        .settings-title {
            font-size: 13px;
            font-weight: 700;
            color: var(--dash-text);
            margin: 10px 0;
        }

        .settings-row {
            margin-bottom: 10px;
        }

        .settings-row input,
        .settings-row textarea {
            width: 100%;
            padding: 8px 10px;
            border-radius: 8px;
            border: 1px solid rgba(255, 255, 255, 0.15);
            background: rgba(0, 0, 0, 0.2);
            color: #fff;
            font-size: 12px;
            box-sizing: border-box;
        }

        .settings-row textarea {
            min-height: 60px;
            resize: vertical;
        }
    </style>
</head>
<body>
    <div id="dashboard-view" class="view-section active">
        <div class="dash-sidebar">
            <div class="dash-logo">WangScape 写作助手</div>
            <button class="dash-btn primary" onclick="openCreateModal()">+ 新建文章 (双语同步)</button>
            <button class="dash-btn" onclick="runCommand('preview')">🌍 启动实时预览</button>
            <button class="dash-btn" onclick="runCommand('deploy')">🚀 一键提交推送</button>
            <button class="dash-btn" onclick="switchView('statistics')">📊 访问统计</button>
            <button class="dash-btn" onclick="switchView('pending-comments')">💬 未审核评论</button>
            <button class="dash-btn" onclick="switchView('history')">📋 操作历史</button>
            <button class="dash-btn" onclick="location.reload()">🔄 刷新列表</button>
            
            <div id="comment-stats-box" style="background: rgba(255,152,0,0.1); border: 1px solid rgba(255,152,0,0.3); border-radius: 12px; padding: 15px; margin-top: 20px; display: none;">
                <div style="font-size: 13px; color: #ff9800; font-weight: 600; margin-bottom: 8px;">💬 评论统计</div>
                <div style="font-size: 12px; color: var(--dash-text); line-height: 1.8;">
                    <div>待审核: <span id="pending-count" style="color: #ff9800; font-weight: 700;">0</span></div>
                    <div>总评论: <span id="total-count" style="color: var(--dash-text);">0</span></div>
                </div>
            </div>

            <div id="auth-panel" style="margin-top: 18px; padding: 12px; background: rgba(79,70,229,0.08); border: 1px solid rgba(79,70,229,0.2); border-radius: 12px;">
                <div style="font-size: 12px; color: #4f46e5; font-weight: 700; margin-bottom: 8px;">🔐 登录状态</div>
                <div id="auth-status" style="font-size: 12px; color: var(--dash-text); margin-bottom: 10px;">未登录</div>
                <div style="display:flex; gap:8px;">
                    <button id="login-btn" class="dash-btn" style="flex:1;" onclick="if(window.openLoginModal){openLoginModal();}else{var m=document.getElementById('login-modal');if(m){m.style.display='flex';}}">登录</button>
                    <button id="logout-btn" class="dash-btn" style="flex:1; display:none;" onclick="logout()">退出</button>
                </div>
            </div>
            
            <div style="margin-top:auto; font-size:12px; color:var(--dash-text-dim);">
                <span>系统状态: 在线</span><br>
                v3.0 Go Edition
            </div>
        </div>
        <div class="dash-main">
            <h1 class="dash-header">最新博文内容</h1>
            <div id="dash-post-list" class="post-list-card"></div>
        </div>
    </div>

    <div id="statistics-view" class="view-section">
                <div class="dash-sidebar">
                    <div class="dash-logo">📊 访问统计</div>
                    <button class="dash-btn" onclick="switchView('dashboard')">← 返回主面板</button>
                    <button class="dash-btn" onclick="loadStatistics()">🔄 刷新数据</button>
            
                    <div id="auth-panel-stats" style="margin: 12px 0 8px; padding: 12px; background: rgba(79,70,229,0.08); border: 1px solid rgba(79,70,229,0.2); border-radius: 12px;">
                        <div style="font-size: 12px; color: #4f46e5; font-weight: 700; margin-bottom: 8px;">🔐 登录状态</div>
                        <div id="auth-status-stats" style="font-size: 12px; color: var(--dash-text); margin-bottom: 10px;">未登录</div>
                        <div style="display:flex; gap:8px;">
                            <button id="login-btn-stats" class="dash-btn" style="flex:1;" onclick="if(window.openLoginModal){openLoginModal();}else{var m=document.getElementById('login-modal');if(m){m.style.display='flex';}}">登录</button>
                            <button id="logout-btn-stats" class="dash-btn" style="flex:1; display:none;" onclick="logout()">退出</button>
                        </div>
                    </div>
            
                    <div style="margin-top:auto; font-size:12px; color:var(--dash-text-dim); padding-top:20px; border-top: 1px solid var(--dash-border);">
                        <div style="margin-bottom: 8px;">最后更新:</div>
                        <div id="stats-update-time" style="font-weight: 600;">-</div>
                    </div>
                </div>
                <div class="dash-main" style="overflow-y: auto;">
                    <div style="padding: 20px;">
                        <h1 class="dash-header">📊 访问统计概览</h1>
                
                        <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 15px; margin-bottom: 30px;">
                            <div style="background: white; padding: 20px; border-radius: 12px; border: 1px solid var(--dash-border);">
                                <div style="font-size: 14px; color: var(--dash-text-dim); margin-bottom: 5px;">总页面数</div>
                                <div style="font-size: 32px; font-weight: 700; color: var(--dash-accent);" id="stat-total-pages">-</div>
                            </div>
                            <div style="background: white; padding: 20px; border-radius: 12px; border: 1px solid var(--dash-border);">
                                <div style="font-size: 14px; color: var(--dash-text-dim); margin-bottom: 5px;">总浏览量</div>
                                <div style="font-size: 32px; font-weight: 700; color: #10b981;" id="stat-total-views">-</div>
                            </div>
                            <div style="background: white; padding: 20px; border-radius: 12px; border: 1px solid var(--dash-border);">
                                <div style="font-size: 14px; color: var(--dash-text-dim); margin-bottom: 5px;">评论总数</div>
                                <div style="font-size: 32px; font-weight: 700; color: #f59e0b;" id="stat-total-comments">-</div>
                            </div>
                            <div style="background: white; padding: 20px; border-radius: 12px; border: 1px solid var(--dash-border);">
                                <div style="font-size: 14px; color: var(--dash-text-dim); margin-bottom: 5px;">独立IP数</div>
                                <div style="font-size: 32px; font-weight: 700; color: #8b5cf6;" id="stat-unique-ips">-</div>
                            </div>
                        </div>
                
                        <h2 style="margin-top: 30px; font-size: 18px; font-weight: 600;">🏆 最受欢迎的文章 (前10)</h2>
                        <div id="stat-top-pages" style="background: white; border-radius: 12px; border: 1px solid var(--dash-border); padding: 0; margin-bottom: 30px;">
                            <!-- 页面列表会插入这里 -->
                        </div>
                
                        <h2 style="margin-top: 30px; font-size: 18px; font-weight: 600;">👥 访客IP统计 (前50)</h2>
                        <div id="stat-visitors" style="background: white; border-radius: 12px; border: 1px solid var(--dash-border); padding: 0;">
                            <!-- 访客IP列表会插入这里 -->
                        </div>
                    </div>
                </div>
            </div>

    <div id="pending-comments-view" class="view-section">
        <div class="dash-sidebar">
            <div class="dash-logo">未审核评论</div>
            <button class="dash-btn" onclick="switchView('dashboard')">← 返回主面板</button>
            <button class="dash-btn" onclick="loadPendingComments()">🔄 刷新</button>
            <button class="dash-btn" onclick="exportCommentsCsv()">📥 导出CSV</button>

            <div id="auth-panel-pending" style="margin: 12px 0 8px; padding: 12px; background: rgba(79,70,229,0.08); border: 1px solid rgba(79,70,229,0.2); border-radius: 12px;">
                <div style="font-size: 12px; color: #4f46e5; font-weight: 700; margin-bottom: 8px;">🔐 登录状态</div>
                <div id="auth-status-pending" style="font-size: 12px; color: var(--dash-text); margin-bottom: 10px;">未登录</div>
                <div style="display:flex; gap:8px;">
                    <button id="login-btn-pending" class="dash-btn" style="flex:1;" onclick="if(window.openLoginModal){openLoginModal();}else{var m=document.getElementById('login-modal');if(m){m.style.display='flex';}}">登录</button>
                    <button id="logout-btn-pending" class="dash-btn" style="flex:1; display:none;" onclick="logout()">退出</button>
                </div>
            </div>

            <div class="settings-panel">
                <div class="settings-title">🔔 邮件通知</div>
                <div class="settings-row">
                    <label><input type="checkbox" id="smtp-enabled" /> 启用SMTP</label>
                </div>
                <div class="settings-row">
                    <input type="text" id="smtp-host" placeholder="SMTP Host" />
                </div>
                <div class="settings-row">
                    <input type="number" id="smtp-port" placeholder="SMTP Port" />
                </div>
                <div class="settings-row">
                    <input type="text" id="smtp-user" placeholder="SMTP 用户名" />
                </div>
                <div class="settings-row">
                    <input type="password" id="smtp-pass" placeholder="SMTP 密码" />
                </div>
                <div class="settings-row">
                    <input type="text" id="smtp-from" placeholder="发件人地址" />
                </div>
                <div class="settings-row">
                    <input type="text" id="smtp-to" placeholder="收件人(逗号分隔)" />
                </div>
                <div class="settings-row">
                    <label><input type="checkbox" id="notify-pending" /> 新评论提醒</label>
                </div>
                <div class="settings-row">
                    <button class="dash-btn" onclick="testMailConnection()" style="background:#3b82f6;">✉️ 发送测试邮件</button>
                </div>

                <div class="settings-title">⛔ 黑名单</div>
                <div class="settings-row">
                    <textarea id="blacklist-ips" placeholder="IP列表，一行一个"></textarea>
                </div>
                <div class="settings-row">
                    <textarea id="blacklist-words" placeholder="关键词列表，一行一个"></textarea>
                </div>

                <button class="dash-btn" onclick="saveCommentSettings()">💾 保存设置</button>
            </div>
            
            <div style="margin-top:auto; font-size:12px; color:var(--dash-text-dim);">
                <span id="pending-total-count">加载中...</span>
            </div>
        </div>
        <div class="dash-main">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                <h1 class="dash-header" style="margin: 0;">📬 待审核评论</h1>
                <div style="display: flex; align-items: center; gap: 10px;">
                    <span id="pending-total-count-header" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 6px 14px; border-radius: 20px; font-size: 13px; font-weight: 600; box-shadow: 0 2px 8px rgba(102,126,234,0.3);">0 条待审核</span>
                    <button onclick="loadPendingComments()" style="background: white; border: 1px solid #e5e7eb; padding: 8px 16px; border-radius: 8px; cursor: pointer; font-size: 14px; transition: all 0.2s;" onmouseover="this.style.background='#f9fafb'" onmouseout="this.style.background='white'">🔄 刷新</button>
                </div>
            </div>
            
            <div class="pending-toolbar">
                <div style="display: flex; align-items: center; gap: 10px;">
                    <label style="display: flex; align-items: center; gap: 8px; cursor: pointer; padding: 8px 12px; background: #f9fafb; border-radius: 8px; transition: all 0.2s;" onmouseover="this.style.background='#f3f4f6'" onmouseout="this.style.background='#f9fafb'">
                        <input type="checkbox" id="pending-select-all" onchange="toggleSelectAllPending()" style="width: 16px; height: 16px; cursor: pointer;" /> 
                        <span style="font-weight: 500; font-size: 14px;">全选</span>
                    </label>
                    <span id="selected-count" style="color: #6b7280; font-size: 13px; margin-left: 5px;">未选择</span>
                </div>
                <div style="display: flex; gap: 10px;">
                    <button class="btn-approve-bulk" onclick="bulkApprovePending()">✅ 批量批准</button>
                    <button class="btn-delete-bulk" onclick="bulkDeletePending()">🗑 批量删除</button>
                </div>
            </div>
            
            <div id="pending-comments-list" style="display:flex; flex-direction:column; gap:16px;"></div>
        </div>
    </div>

    <div id="history-view" class="view-section">
        <div class="dash-sidebar">
            <div class="dash-logo">📊 操作历史</div>
            <button class="dash-btn" onclick="switchView('dashboard')">← 返回主面板</button>
            <button class="dash-btn" onclick="loadOperationHistory()">🔄 刷新</button>
            <button class="dash-btn" onclick="exportHistoryCsv()">📥 导出CSV</button>
            
            <div class="settings-panel">
                <div class="settings-title">📁 筛选类型</div>
                <label style="display:block; margin:8px 0;">
                    <input type="checkbox" id="filter-comments" checked onchange="loadOperationHistory()" /> 评论操作
                </label>
                <label style="display:block; margin:8px 0;">
                    <input type="checkbox" id="filter-posts" checked onchange="loadOperationHistory()" /> 文章操作
                </label>
            </div>

            <div class="settings-panel">
                <div class="settings-title">🕐 日期范围</div>
                <input type="date" id="history-date-from" style="width:100%; padding:8px; margin:5px 0; border:1px solid #d1d5db; border-radius:6px;" onchange="loadOperationHistory()" />
                <input type="date" id="history-date-to" style="width:100%; padding:8px; margin:5px 0; border:1px solid #d1d5db; border-radius:6px;" onchange="loadOperationHistory()" />
            </div>
        </div>
        <div class="dash-main">
            <h1 class="dash-header">操作历史记录</h1>
            <div class="history-stats" style="display:grid; grid-template-columns:repeat(3,1fr); gap:15px; margin-bottom:20px;">
                <div style="background:rgba(79,70,229,0.1); padding:15px; border-radius:8px; border-left:4px solid #4f46e5;">
                    <div style="font-size:12px; color:#6366f1; font-weight:600; margin-bottom:5px;">总操作数</div>
                    <div style="font-size:24px; font-weight:700; color:#4f46e5;" id="total-ops">0</div>
                </div>
                <div style="background:rgba(34,197,94,0.1); padding:15px; border-radius:8px; border-left:4px solid #22c55e;">
                    <div style="font-size:12px; color:#16a34a; font-weight:600; margin-bottom:5px;">评论操作</div>
                    <div style="font-size:24px; font-weight:700; color:#22c55e;" id="total-comment-ops">0</div>
                </div>
                <div style="background:rgba(59,130,246,0.1); padding:15px; border-radius:8px; border-left:4px solid #3b82f6;">
                    <div style="font-size:12px; color:#1d4ed8; font-weight:600; margin-bottom:5px;">文章操作</div>
                    <div style="font-size:24px; font-weight:700; color:#3b82f6;" id="total-post-ops">0</div>
                </div>
            </div>
            <div id="history-list" style="display:flex; flex-direction:column; gap:12px;"></div>
        </div>
    </div>

    <div id="editor-view" class="view-section">
        <div class="word-topbar">
            <div style="display:flex; align-items:center; gap:15px;">
                <button class="word-back-btn" onclick="switchView('dashboard')">← 返回主面板</button>
                <strong style="font-size:16px;">WangScape 写作器</strong>
                <span id="current-doc-name" style="opacity:0.9; font-size:14px; font-weight:500;"></span>
            </div>
            <div style="display:flex; align-items:center; gap:20px;">
                <span id="word-count" style="font-size:13px; color:rgba(255,255,255,0.9);">字数: 0</span>
                <span id="save-status" style="font-size:13px; color:rgba(255,255,255,0.8);"></span>
            </div>
        </div>
        <div class="word-ribbon">
            <button class="word-rib-btn" onclick="saveDocument()" title="保存文档 (Ctrl+S)">
                <span>💾</span>
                <span>保存</span>
            </button>
            <button class="word-rib-btn" onclick="toggleMetadataPanel()" title="编辑文章信息">
                <span>📋</span>
                <span>元数据</span>
            </button>
            <button class="word-rib-btn" onclick="switchCommentView()" title="管理评论">
                <span>💬</span>
                <span>评论</span>
            </button>
            <div style="width:1px; height:30px; background:#e0e0e0; margin:0 5px;"></div>
            <button class="word-rib-btn" onclick="insertCodeBlock()" title="插入代码块">
                <span>💻</span>
                <span>代码</span>
            </button>
            <button class="word-rib-btn" onclick="insertImage()" title="插入图片">
                <span>🖼</span>
                <span>图片</span>
            </button>
            <button class="word-rib-btn" onclick="insertTable()" title="插入表格">
                <span>📊</span>
                <span>表格</span>
            </button>
            <div style="width:1px; height:30px; background:#e0e0e0; margin:0 5px;"></div>
            <button class="word-rib-btn" onclick="runCommand('preview')" title="实时预览">
                <span>👁</span>
                <span>预览</span>
            </button>
            <button class="word-rib-btn" onclick="runCommand('deploy')" title="发布到网站">
                <span>🚀</span>
                <span>发布</span>
            </button>
        </div>
        <div class="word-workspace">
            <div class="word-canvas">
                <div class="word-paper" id="paper-content">
                    <div style="text-align:center; color:#999; margin-top:100px;">
                        请选择左侧文章进行编辑
                    </div>
                </div>
                <div class="meta-panel" id="meta-panel" style="display:none;">
                    <h3>📋 文章信息</h3>
                    
                    <div class="meta-section">
                        <label>标题 (Title)</label>
                        <input type="text" class="meta-input" id="title-input" placeholder="文章标题" />
                    </div>

                    <div class="meta-section">
                        <label>日期 (Date)</label>
                        <input type="datetime-local" class="meta-input" id="date-input" />
                    </div>

                    <div class="meta-section">
                        <label>分类 (Categories)</label>
                        <div class="tag-container" id="categories-container"></div>
                        <div class="tag-input-row">
                            <input type="text" id="category-input" placeholder="添加分类..." onkeypress="if(event.key==='Enter'){addCategory();event.preventDefault();}" />
                            <button onclick="addCategory()">添加</button>
                        </div>
                    </div>

                    <div class="meta-section">
                        <label>标签 (Tags)</label>
                        <div class="tag-container" id="tags-container"></div>
                        <div class="tag-input-row">
                            <input type="text" id="tag-input" placeholder="添加标签..." onkeypress="if(event.key==='Enter'){addTag();event.preventDefault();}" />
                            <button onclick="addTag()">添加</button>
                        </div>
                    </div>

                    <div class="meta-section">
                        <label>描述 (Description)</label>
                        <textarea class="meta-input" id="description-input" rows="3" placeholder="文章简介..." style="resize: vertical; min-height: 60px;"></textarea>
                    </div>

                    <div class="meta-section">
                        <label>封面图片 URL (Image)</label>
                        <input type="text" class="meta-input" id="image-input" placeholder="/img/cover.jpg" />
                    </div>

                    <div class="meta-section">
                        <label>许可证 (License)</label>
                        <input type="text" class="meta-input" id="license-input" placeholder="CC BY-SA 4.0" />
                    </div>

                    <div class="meta-section">
                        <div class="meta-checkbox">
                            <input type="checkbox" id="draft-checkbox" />
                            <label for="draft-checkbox">📝 草稿状态</label>
                        </div>
                        <div class="meta-checkbox">
                            <input type="checkbox" id="math-checkbox" />
                            <label for="math-checkbox">📐 启用数学公式</label>
                        </div>
                        <div class="meta-checkbox">
                            <input type="checkbox" id="comments-checkbox" />
                            <label for="comments-checkbox">💬 允许评论</label>
                        </div>
                        <div class="meta-checkbox">
                            <input type="checkbox" id="hidden-checkbox" />
                            <label for="hidden-checkbox">🔒 隐藏文章</label>
                        </div>
                        <div class="meta-checkbox">
                            <input type="checkbox" id="pinned-checkbox" />
                            <label for="pinned-checkbox">📌 置顶文章</label>
                        </div>
                    </div>

                    <button class="dash-btn primary" style="width:100%; margin-top:10px;" onclick="applyMetadata()">💾 应用更改</button>
                </div>

                <div id="comments-panel" class="meta-panel hide">
                    <h3>💬 评论管理</h3>
                    <div id="comments-list" style="max-height: 500px; overflow-y: auto;"></div>
                </div>
            </div>
        </div>
    </div>

    <div class="modal-overlay" id="create-modal">
        <div class="modal-card">
            <h2 style="margin-top:0">创建新文章</h2>
            <label>中文标题</label>
            <input type="text" id="postTitle" placeholder="例如：冬日随笔">
            <label>分类（英文）</label>
            <input type="text" id="postCat" placeholder="Life, Code">
            <p style="font-size:12px; color:var(--dash-text-dim)">* 系统将自动翻译为英文并创建双语版本。</p>
            <div style="text-align:right">
                <button class="btn-cancel" onclick="closeCreateModal()">取消</button>
                <button class="btn-confirm" onclick="createPost()">创建</button>
            </div>
        </div>
    </div>

    <div class="modal-overlay" id="login-modal" style="display:none;">
        <div class="modal-card">
            <h2 style="margin-top:0">管理员登录</h2>
            <label>用户名</label>
            <input type="text" id="login-username" placeholder="admin">
            <label>密码</label>
            <input type="password" id="login-password" placeholder="请输入密码">
            <div style="display:flex; gap:10px; justify-content:flex-end; margin-top:10px;">
                <button class="btn-cancel" onclick="closeLoginModal()">取消</button>
                <button class="btn-confirm" onclick="if(window.performLogin){performLogin();}else{(async function(){var u=document.getElementById('login-username').value.trim();var p=document.getElementById('login-password').value;if(!u||!p){alert('请输入用户名和密码');return;}try{var r=await fetch('/api/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:u,password:p})});var d=await r.json().catch(function(){return {};});if(d&&d.access_token){localStorage.setItem('auth_token',d.access_token);var m=document.getElementById('login-modal');if(m){m.style.display='none';}alert('✅ 登录成功');location.reload();}else{alert(d.message||('登录失败: '+r.status));}}catch(e){alert('网络错误: '+e.message);}})();}">登录</button>
            </div>
            <p id="login-hint" style="font-size:12px; color:#64748b; margin-top:8px; display:none;"></p>
        </div>
    </div>

    <script>
        let postsData = [];
        let currentDocPath = '';
        let commentStatsData = null;
        let authToken = localStorage.getItem('auth_token') || '';

        function setAuthToken(token) {
            authToken = token || '';
            if (authToken) {
                localStorage.setItem('auth_token', authToken);
            } else {
                localStorage.removeItem('auth_token');
            }
            updateAuthStatus();
        }

        function getAuthHeaders() {
            if (!authToken) return {};
            return { 'Authorization': 'Bearer ' + authToken };
        }

        async function authFetch(url, options = {}) {
            const headers = Object.assign({}, options.headers || {}, getAuthHeaders());
            console.log('[DEBUG] authFetch:', { url, headers, authToken: authToken ? '***' : 'NONE' });
            const response = await fetch(url, Object.assign({}, options, { headers }));
            console.log('[DEBUG] Response status:', response.status);
            if (response.status === 401) {
                openLoginModal('需要登录才能继续操作');
            }
            return response;
        }

        async function fetchWithTimeout(url, options, timeoutMs) {
            const controller = new AbortController();
            const timer = setTimeout(function () {
                controller.abort();
            }, timeoutMs || 8000);

            try {
                const finalOptions = Object.assign({}, options || {}, { signal: controller.signal });
                return await fetch(url, finalOptions);
            } finally {
                clearTimeout(timer);
            }
        }

        async function loginWithFallback(payload) {
            const candidates = [];
            candidates.push('/api/login');

            const origin = window.location.origin || '';
            if (origin.indexOf('http://127.0.0.1:8080') !== 0) {
                candidates.push('http://127.0.0.1:8080/api/login');
            }
            if (origin.indexOf('http://localhost:8080') !== 0) {
                candidates.push('http://localhost:8080/api/login');
            }

            let lastError = null;
            for (let i = 0; i < candidates.length; i++) {
                const url = candidates[i];
                try {
                    const res = await fetchWithTimeout(url, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(payload)
                    }, 8000);
                    return res;
                } catch (err) {
                    lastError = err;
                    console.warn('[LOGIN] 请求失败，尝试下一个地址:', url, err && err.message ? err.message : err);
                }
            }

            throw lastError || new Error('登录请求失败');
        }

        function openLoginModal(message) {
            console.log('openLoginModal called with message:', message);
            const modal = document.getElementById('login-modal');
            const hint = document.getElementById('login-hint');
            
            if (!modal) {
                console.error('login-modal element not found!');
                alert('错误：登录窗口未找到，请刷新页面重试');
                return;
            }
            
            console.log('Modal element found, displaying...');
            
            if (message) {
                hint.textContent = message;
                hint.style.display = 'block';
            } else {
                hint.style.display = 'none';
            }
            modal.style.display = 'flex';
            
            console.log('Modal display set to flex');
        }

        function closeLoginModal() {
            const modal = document.getElementById('login-modal');
            if (modal) {
                modal.style.display = 'none';
            }
        }

        async function performLogin() {
            const username = document.getElementById('login-username').value.trim();
            const password = document.getElementById('login-password').value;
            if (!username || !password) {
                openLoginModal('请输入用户名和密码');
                return;
            }
            
            try {
                const res = await loginWithFallback({ username, password });
                
                if (!res.ok) {
                    const data = await res.json().catch(() => ({ message: '服务器错误' }));
                    openLoginModal(data.message || '登录失败: ' + res.status);
                    console.error('登录失败:', res.status, data);
                    return;
                }
                
                const data = await res.json();
                console.log('登录响应:', data);
                
                if (data && data.access_token) {
                    setAuthToken(data.access_token);
                    closeLoginModal();
                    updateAuthStatus();
                    alert('✅ 登录成功！');
                } else {
                    openLoginModal(data.message || '登录失败：未返回令牌');
                }
            } catch (e) {
                console.error('登录错误:', e);
                openLoginModal('网络错误: 无法连接后端，请确认 WSwriter 正在运行（127.0.0.1:8080）');
            }
        }

        function logout() {
            setAuthToken('');
        }

        function updateAuthStatus() {
            const statusText = authToken ? '已登录' : '未登录';
            const statusEl = document.getElementById('auth-status');
            const statusElPending = document.getElementById('auth-status-pending');
            const statusElStats = document.getElementById('auth-status-stats');
            if (statusEl) statusEl.textContent = statusText;
            if (statusElPending) statusElPending.textContent = statusText;
            if (statusElStats) statusElStats.textContent = statusText;
            
            // 根据登录状态切换按钮显示
            const loginBtn = document.getElementById('login-btn');
            const logoutBtn = document.getElementById('logout-btn');
            const loginBtnPending = document.getElementById('login-btn-pending');
            const logoutBtnPending = document.getElementById('logout-btn-pending');
            const loginBtnStats = document.getElementById('login-btn-stats');
            const logoutBtnStats = document.getElementById('logout-btn-stats');
            
            if (authToken) {
                // 已登录：隐藏登录按钮，显示退出按钮
                if (loginBtn) loginBtn.style.display = 'none';
                if (logoutBtn) logoutBtn.style.display = 'block';
                if (loginBtnPending) loginBtnPending.style.display = 'none';
                if (logoutBtnPending) logoutBtnPending.style.display = 'block';
                if (loginBtnStats) loginBtnStats.style.display = 'none';
                if (logoutBtnStats) logoutBtnStats.style.display = 'block';
            } else {
                // 未登录：显示登录按钮，隐藏退出按钮
                if (loginBtn) loginBtn.style.display = 'block';
                if (logoutBtn) logoutBtn.style.display = 'none';
                if (loginBtnPending) loginBtnPending.style.display = 'block';
                if (logoutBtnPending) logoutBtnPending.style.display = 'none';
                if (loginBtnStats) loginBtnStats.style.display = 'block';
                if (logoutBtnStats) logoutBtnStats.style.display = 'none';
            }
        }

        function switchView(view) {
            document.querySelectorAll('.view-section').forEach(e => e.classList.remove('active'));
            document.getElementById(view + '-view').classList.add('active');
            if (view === 'dashboard') {
                fetchPosts();
                fetchCommentStats();
            } else if (view === 'pending-comments') {
                loadPendingComments();
                loadCommentSettings();
            } else if (view === 'history') {
                loadOperationHistory();
            } else if (view === 'statistics') {
                loadStatistics();
            }
        }

        async function fetchCommentStats() {
            try {
                const res = await authFetch('/api/comment_stats');
                const data = await res.json();
                if (data.success && data.data) {
                    commentStatsData = data.data;
                    updateCommentStatsDisplay();
                    renderDashboardList();
                }
            } catch(e) {
                console.error('获取评论统计失败:', e);
            }
        }

        function updateCommentStatsDisplay() {
            if (!commentStatsData) return;
            
            const statsBox = document.getElementById('comment-stats-box');
            const pendingCount = document.getElementById('pending-count');
            const totalCount = document.getElementById('total-count');
            
            if (commentStatsData.total_pending > 0 || commentStatsData.total_comments > 0) {
                statsBox.style.display = 'block';
                pendingCount.textContent = commentStatsData.total_pending;
                totalCount.textContent = commentStatsData.total_comments;
            } else {
                statsBox.style.display = 'none';
            }
        }

        async function fetchPosts() {
            const res = await fetch('/api/posts');
            postsData = await res.json();
            renderDashboardList();
        }

        function renderDashboardList() {
            const list = document.getElementById('dash-post-list');
            if (postsData.length === 0) {
                list.innerHTML = '<div style="padding:40px; text-align:center; color:#555;">暂无文章</div>';
                return;
            }
            
            // 按路径分组（中英文版本）
            const grouped = {};
            postsData.forEach(p => {
                // 提取基础名称（去掉 zh-cn 或 en 前缀）
                const baseName = p.path.replace(/content\/(zh-cn|en)\/post\//, '');
                if (!grouped[baseName]) {
                    grouped[baseName] = { zh: null, en: null };
                }
                if (p.lang === 'zh-cn' || p.lang === 'zh') {
                    grouped[baseName].zh = p;
                } else if (p.lang === 'en') {
                    grouped[baseName].en = p;
                }
            });
            
            list.innerHTML = Object.entries(grouped).map(([baseName, versions]) => {
                const primaryVersion = versions.zh || versions.en;
                if (!primaryVersion) return '';
                
                const escapedPath = primaryVersion.path.replace(/\\/g, '\\\\');
                let html = '<div class="dash-post-item">' +
                    '<div onclick="openEditor(\'' + escapedPath + '\', \'' + primaryVersion.title.replace(/'/g, "\\'") + '\', \'' + primaryVersion.date + '\')" style="flex:1; cursor:pointer; display:flex; flex-direction:column; gap:4px;">' +
                    '<div style="display:flex; align-items:center; gap:10px;">' +
                    '<div class="dpi-title">' + primaryVersion.title + '</div>' +
                    '<span style="font-size:10px; padding:2px 6px; border-radius:4px; background:' + primaryVersion.status_color + '20; color:' + primaryVersion.status_color + ';">' +
                    primaryVersion.status +
                    '</span>';
                
                // 显示版本标签
                if (versions.zh && versions.en) {
                    html += '<span style="font-size:9px; padding:2px 4px; background:#4a90e2; color:#fff; border-radius:3px;">中英双版</span>';
                } else if (versions.zh) {
                    html += '<span style="font-size:9px; padding:2px 4px; background:#ff7f50; color:#fff; border-radius:3px;">中文版</span>';
                } else if (versions.en) {
                    html += '<span style="font-size:9px; padding:2px 4px; background:#50c878; color:#fff; border-radius:3px;">英文版</span>';
                }
                
                // 显示置顶标识
                if (primaryVersion.pinned) {
                    html += '<span style="font-size:9px; padding:2px 4px; background:#ff4444; color:#fff; border-radius:3px; margin-left:4px;">📌 置顶</span>';
                }
                
                // 显示评论统计
                if (commentStatsData && commentStatsData.post_stats) {
                    const stats = commentStatsData.post_stats[primaryVersion.path];
                    if (stats && stats.total > 0) {
                        const pendingBadge = stats.pending > 0 ? 
                            '<span style="font-size:9px; padding:2px 4px; background:#ff9800; color:#fff; border-radius:3px; margin-left:4px;">' + stats.pending + ' 待审</span>' : '';
                        html += '<span style="font-size:9px; padding:2px 4px; background:#9e9e9e; color:#fff; border-radius:3px; margin-left:4px;">💬 ' + stats.total + '</span>' + pendingBadge;
                    }
                }
                
                html += '</div>' +
                    '<div class="dpi-meta">' + primaryVersion.date + ' · ' + primaryVersion.path + '</div>' +
                    '</div>' +
                    '<div style="display:flex; gap:8px; align-items:center;">';
                
                // 显示切换按钮
                if (versions.zh && versions.en) {
                    const zhPath = versions.zh.path.replace(/\\/g, '\\\\');
                    const enPath = versions.en.path.replace(/\\/g, '\\\\');
                    const zhTitle = versions.zh.title.replace(/'/g, "\\'");
                    const enTitle = versions.en.title.replace(/'/g, "\\'");
                    
                        html += '<button onclick="openEditor(\'' + zhPath + '\', \'' + zhTitle + '\', \'' + versions.zh.date + '\')" style="background:#fff7ed; border:1px solid #fed7aa; color:#c2410c; padding:4px 8px; border-radius:6px; font-size:11px; cursor:pointer;">编辑中文</button>' +
                            '<button onclick="openEditor(\'' + enPath + '\', \'' + enTitle + '\', \'' + versions.en.date + '\')" style="background:#ecfdf3; border:1px solid #bbf7d0; color:#15803d; padding:4px 8px; border-radius:6px; font-size:11px; cursor:pointer;">编辑英文</button>';
                }
                
                    html += '<button onclick="deleteDocument(\'' + escapedPath + '\')" style="background:#fee2e2; border:1px solid #fecaca; color:#b91c1c; width:32px; height:32px; border-radius:8px; cursor:pointer;">🗑</button>' +
                        '<button onclick="openEditor(\'' + escapedPath + '\', \'' + primaryVersion.title.replace(/'/g, "\\'") + '\', \'' + primaryVersion.date + '\')" style="background:#eef2ff; border:1px solid #c7d2fe; color:#4338ca; width:32px; height:32px; border-radius:8px; cursor:pointer;">✎</button>' +
                        '</div>' +
                        '</div>';
                
                return html;
            }).join('');
        }

        async function openEditor(path, title, date) {
            currentDocPath = path;
            switchView('editor');
            
            // 检测当前编辑的语言版本
            const isZhCN = path.includes('zh-cn');
            const lang = isZhCN ? '中文版' : '英文版';
            const langColor = isZhCN ? '#ffa500' : '#50c878';
            
            document.getElementById('current-doc-name').textContent = title + ' (' + lang + ')';
            document.getElementById('current-doc-name').style.color = langColor;
            
            const paper = document.getElementById('paper-content');
            const metaPanel = document.getElementById('meta-panel');
            paper.innerHTML = '<div style="text-align:center; margin-top:50px; color:#888;">加载中...</div>';

            try {
                const res = await authFetch('/api/get_content?path=' + encodeURIComponent(path));
                const data = await res.json();
                
                // 解析frontmatter
                parseFrontmatter(data.content);
                
                paper.innerHTML = '<div class="wp-title">' + title + '</div>' +
                    '<div style="font-size:12px; color:#999; margin-bottom:20px;">版本: ' + lang + ' · 日期: ' + date + '</div>' +
                    '<textarea id="editor-textarea" spellcheck="false">' + data.content + '</textarea>';
                
                // 添加输入监听器
                const textarea = document.getElementById('editor-textarea');
                textarea.addEventListener('input', updateWordCount);
                textarea.addEventListener('input', function() {
                    document.getElementById('save-status').textContent = '⚠️ 未保存';
                    document.getElementById('save-status').style.color = 'rgba(255, 200, 100, 0.9)';
                });
                
                // 初始化字数统计
                updateWordCount();
                
                // 显示元数据面板
                metaPanel.style.display = 'block';
            } catch(e) {
                paper.innerHTML = '<div style="color:red">错误: ' + e + '</div>';
            }
        }

        let currentMetadata = {
            title: '',
            date: '',
            categories: [],
            tags: [],
            description: '',
            image: '',
            license: '',
            draft: false,
            math: false,
            comments: true,
            hidden: false,
            pinned: false
        };

        function parseFrontmatter(content) {
            // 提取frontmatter
            const fmMatch = content.match(/^---\n([\s\S]*?)\n---/);
            if (!fmMatch) {
                currentMetadata = { title: '', date: '', categories: [], tags: [], description: '', image: '', license: '', draft: false, math: false, comments: true, hidden: false, pinned: false };
                renderMetadata();
                return;
            }

            const fmContent = fmMatch[1];
            
            // 解析title
            const titleMatch = fmContent.match(/title:\s*["']?([^"'\n]+)["']?/);
            currentMetadata.title = titleMatch ? titleMatch[1].trim() : '';

            // 解析date
            const dateMatch = fmContent.match(/date:\s*([\w\-:+]+)/);
            if (dateMatch) {
                // 转换为datetime-local格式 (YYYY-MM-DDTHH:MM)
                const dateStr = dateMatch[1].replace(/([\d-]+)T([\d:]+).*/, '$1T$2');
                currentMetadata.date = dateStr.substring(0, 16);
            } else {
                currentMetadata.date = '';
            }
            
            // 解析categories
            const catMatch = fmContent.match(/categories:\s*\n((?:\s*-\s*.+\n)+)/);
            if (catMatch) {
                currentMetadata.categories = catMatch[1].split('\n')
                    .filter(l => l.trim().startsWith('-'))
                    .map(l => l.replace(/^\s*-\s*/, '').trim());
            } else {
                const catSingleMatch = fmContent.match(/categories:\s*\[([^\]]+)\]/);
                if (catSingleMatch) {
                    currentMetadata.categories = catSingleMatch[1].split(',').map(c => c.trim());
                } else {
                    currentMetadata.categories = [];
                }
            }

            // 解析tags
            const tagMatch = fmContent.match(/tags:\s*\n((?:\s*-\s*.+\n)+)/);
            if (tagMatch) {
                currentMetadata.tags = tagMatch[1].split('\n')
                    .filter(l => l.trim().startsWith('-'))
                    .map(l => l.replace(/^\s*-\s*/, '').trim());
            } else {
                const tagSingleMatch = fmContent.match(/tags:\s*\[([^\]]+)\]/);
                if (tagSingleMatch) {
                    currentMetadata.tags = tagSingleMatch[1].split(',').map(t => t.trim());
                } else {
                    currentMetadata.tags = [];
                }
            }

            // 解析description
            const descMatch = fmContent.match(/description:\s*["']?([^"'\n]+)["']?/);
            currentMetadata.description = descMatch ? descMatch[1].trim() : '';

            // 解析image
            const imgMatch = fmContent.match(/image:\s*["']?([^"'\n]+)["']?/);
            currentMetadata.image = imgMatch ? imgMatch[1].trim() : '';

            // 解析license
            const licenseMatch = fmContent.match(/license:\s*["']?([^"'\n]+)["']?/);
            currentMetadata.license = licenseMatch ? licenseMatch[1].trim() : '';

            // 解析draft
            const draftMatch = fmContent.match(/draft:\s*(true|false)/);
            currentMetadata.draft = draftMatch ? draftMatch[1] === 'true' : false;

            // 解析math
            const mathMatch = fmContent.match(/math:\s*(true|false)/);
            currentMetadata.math = mathMatch ? mathMatch[1] === 'true' : false;

            // 解析comments
            const commentsMatch = fmContent.match(/comments:\s*(true|false)/);
            currentMetadata.comments = commentsMatch ? commentsMatch[1] === 'true' : true;

            // 解析hidden
            const hiddenMatch = fmContent.match(/hidden:\s*(true|false)/);
            currentMetadata.hidden = hiddenMatch ? hiddenMatch[1] === 'true' : false;

            // 解析pinned
            const pinnedMatch = fmContent.match(/pinned:\s*(true|false)/);
            currentMetadata.pinned = pinnedMatch ? pinnedMatch[1] === 'true' : false;

            renderMetadata();
        }

        function renderMetadata() {
            // 渲染title和date
            document.getElementById('title-input').value = currentMetadata.title;
            document.getElementById('date-input').value = currentMetadata.date;

            // 渲染分类
            const catContainer = document.getElementById('categories-container');
            catContainer.innerHTML = currentMetadata.categories.map(cat =>
                '<span class="tag-item">' + cat + '<span class="tag-remove" onclick="removeCategory(\'' + cat + '\')">&times;</span></span>'
            ).join('');

            // 渲染标签
            const tagContainer = document.getElementById('tags-container');
            tagContainer.innerHTML = currentMetadata.tags.map(tag =>
                '<span class="tag-item">' + tag + '<span class="tag-remove" onclick="removeTag(\'' + tag + '\')">&times;</span></span>'
            ).join('');

            // 渲染其他字段
            document.getElementById('description-input').value = currentMetadata.description;
            document.getElementById('image-input').value = currentMetadata.image;
            document.getElementById('license-input').value = currentMetadata.license;
            document.getElementById('draft-checkbox').checked = currentMetadata.draft;
            document.getElementById('math-checkbox').checked = currentMetadata.math;
            document.getElementById('comments-checkbox').checked = currentMetadata.comments;
            document.getElementById('hidden-checkbox').checked = currentMetadata.hidden;
            document.getElementById('pinned-checkbox').checked = currentMetadata.pinned;
        }

        function addCategory() {
            const input = document.getElementById('category-input');
            const value = input.value.trim();
            if (value && !currentMetadata.categories.includes(value)) {
                currentMetadata.categories.push(value);
                renderMetadata();
                input.value = '';
            }
        }

        function removeCategory(cat) {
            currentMetadata.categories = currentMetadata.categories.filter(c => c !== cat);
            renderMetadata();
        }

        function addTag() {
            const input = document.getElementById('tag-input');
            const value = input.value.trim();
            if (value && !currentMetadata.tags.includes(value)) {
                currentMetadata.tags.push(value);
                renderMetadata();
                input.value = '';
            }
        }

        function removeTag(tag) {
            currentMetadata.tags = currentMetadata.tags.filter(t => t !== tag);
            renderMetadata();
        }

        function applyMetadata() {
            // 更新当前元数据
            currentMetadata.title = document.getElementById('title-input').value.trim();
            currentMetadata.date = document.getElementById('date-input').value;
            currentMetadata.description = document.getElementById('description-input').value.trim();
            currentMetadata.image = document.getElementById('image-input').value.trim();
            currentMetadata.license = document.getElementById('license-input').value.trim();
            currentMetadata.draft = document.getElementById('draft-checkbox').checked;
            currentMetadata.math = document.getElementById('math-checkbox').checked;
            currentMetadata.comments = document.getElementById('comments-checkbox').checked;
            currentMetadata.hidden = document.getElementById('hidden-checkbox').checked;
            currentMetadata.pinned = document.getElementById('pinned-checkbox').checked;

            // 获取当前文章内容
            const content = document.getElementById('editor-textarea').value;
            
            // 从content中提取body（排除frontmatter）
            const fmMatch = content.match(/^---\n[\s\S]*?\n---\n([\s\S]*)$/);
            if (!fmMatch) {
                alert('⚠️ 未找到frontmatter，无法更新');
                return;
            }
            const bodyContent = fmMatch[1];

            // 提取现有的同步哈希值（如果有）
            const oldFmMatch = content.match(/^---\n([\s\S]*?)\n---/);
            let syncHash = '';
            if (oldFmMatch) {
                const hashMatch = oldFmMatch[1].match(/ws_sync_zh_hash:\s*["']?([^"'\n]+)["']?/);
                if (hashMatch) {
                    syncHash = hashMatch[1];
                }
            }
            
            // 完全重建frontmatter，确保格式正确
            let newFmLines = ['---'];
            newFmLines.push('pinned: ' + currentMetadata.pinned);
            
            // tags（使用YAML列表格式）
            if (currentMetadata.tags.length > 0) {
                newFmLines.push('tags:');
                currentMetadata.tags.forEach(tag => {
                    newFmLines.push('    - ' + tag);
                });
            }
            
            // categories（使用YAML列表格式）
            if (currentMetadata.categories.length > 0) {
                newFmLines.push('categories:');
                currentMetadata.categories.forEach(cat => {
                    newFmLines.push('    - ' + cat);
                });
            }
            
            // title
            const escapedTitle = currentMetadata.title.replace(/"/g, '\\"');
            newFmLines.push('title: "' + escapedTitle + '"');
            
            // description
            if (currentMetadata.description) {
                const escapedDesc = currentMetadata.description.replace(/"/g, '\\"');
                newFmLines.push('description: "' + escapedDesc + '"');
            }
            
            // date（转换为Hugo格式）
            if (currentMetadata.date) {
                const hugoDate = currentMetadata.date + ':00+08:00';
                newFmLines.push('date: ' + hugoDate);
            }
            
            // image
            newFmLines.push('image: "' + (currentMetadata.image || '') + '"');
            
            // math
            newFmLines.push('math: ' + currentMetadata.math);
            
            // license
            newFmLines.push('license: "' + (currentMetadata.license || '') + '"');
            
            // hidden
            newFmLines.push('hidden: ' + currentMetadata.hidden);
            
            // comments
            newFmLines.push('comments: ' + currentMetadata.comments);
            
            // draft
            newFmLines.push('draft: ' + currentMetadata.draft);
            
            // 保留同步哈希值
            if (syncHash) {
                newFmLines.push('ws_sync_zh_hash: "' + syncHash + '"');
            }
            
            newFmLines.push('---');
            
            // 组合新内容
            const newContent = newFmLines.join('\n') + '\n' + bodyContent;
            
            // 更新编辑器内容
            document.getElementById('editor-textarea').value = newContent;
            
            alert('✅ 元数据已应用到编辑器，请点击保存按钮保存文件');
        }

        async function saveDocument() {
            if(!currentDocPath) return;
            const content = document.getElementById('editor-textarea').value;
            const statusEl = document.getElementById('save-status');
            statusEl.textContent = "💾 保存中...";
            statusEl.style.color = "#ffa500";

            try {
                const res = await authFetch('/api/save_content', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: currentDocPath, content: content })
                });
                const data = await res.json();
                if(data.success) {
                    statusEl.textContent = "✅ 已保存 " + new Date().toLocaleTimeString();
                    statusEl.style.color = "rgba(100, 255, 150, 0.9)";
                    
                    // 更新字数统计
                    updateWordCount();
                    
                    // 记录文章操作历史
                    const docName = document.getElementById('current-doc-name').textContent.trim();
                    const isNew = docName.includes('新建');
                    addOperationHistory('post', isNew ? 'create' : 'edit', currentDocPath, '字数: ' + content.length);
                    
                    // 如果是中文版本，自动同步翻译到英文版本
                    if(currentDocPath.includes('zh-cn')) {
                        statusEl.textContent = "⏳ 正在翻译英文版本...";
                        const enPath = currentDocPath.replace(/zh-cn/g, 'en');

                        const payload = JSON.stringify({
                            zhPath: currentDocPath,
                            enPath: enPath,
                            content: content
                        });

                        async function callSyncTranslateOnce() {
                            const res = await authFetch('/api/sync_translate', {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: payload
                            });

                            let body = null;
                            try {
                                body = await res.json();
                            } catch (_) {
                                body = null;
                            }

                            return { res, body };
                        }

                        // 网络抖动或临时认证状态异常时自动重试一次
                        let syncResult = await callSyncTranslateOnce();
                        if (!(syncResult.res.ok && syncResult.body && syncResult.body.success)) {
                            await new Promise(resolve => setTimeout(resolve, 800));
                            syncResult = await callSyncTranslateOnce();
                        }

                        if (syncResult.res.ok && syncResult.body && syncResult.body.success) {
                            const msg = syncResult.body.message ? String(syncResult.body.message) : '已同步翻译';
                            statusEl.textContent = "✅ 已保存（" + msg + "） " + new Date().toLocaleTimeString();
                        } else {
                            const backendMsg = (syncResult.body && syncResult.body.message) ? String(syncResult.body.message) : '';
                            const detail = backendMsg || ('HTTP ' + syncResult.res.status + ' ' + syncResult.res.statusText);
                            statusEl.textContent = "✅ 已保存（英文同步失败: " + detail + "）";
                        }
                    }
                    
                    setTimeout(() => statusEl.textContent = "", 3000);
                    fetchPosts();
                    return true;
                } else {
                    statusEl.textContent = "❌ 保存失败";
                    statusEl.style.color = "#ff5555";
                    alert("保存失败: " + data.message);
                    return false;
                }
            } catch(e) {
                statusEl.textContent = "❌ 网络错误";
                statusEl.style.color = "#ff5555";
                alert("网络错误: " + e);
                return false;
            }
        }

        async function deleteDocument(path) {
            if(!confirm("确定要删除这篇文章吗？操作不可恢复。")) return;
            try {
                const res = await authFetch('/api/delete_post', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: path })
                });
                const data = await res.json();
                if(data.success) {
                    alert('✅ 文章已删除');
                    addOperationHistory('post', 'delete', path, '文章删除');
                    fetchPosts();
                } else {
                    alert("删除失败: " + data.message);
                }
            } catch(e) {
                alert("网络错误: " + e);
            }
        }

        function openCreateModal() {
            document.getElementById('create-modal').style.display = 'flex';
        }

        function closeCreateModal() {
            document.getElementById('create-modal').style.display = 'none';
        }

        async function createPost() {
            const title = document.getElementById('postTitle').value.trim();
            const cat = document.getElementById('postCat').value.trim();
            if(!title) return alert('⚠️ 请输入文章标题');

            try {
                const res = await authFetch('/api/create_sync', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title, categories: cat || 'Uncategorized' })
                });
                const data = await res.json();
                if(data.success) {
                    closeCreateModal();
                    document.getElementById('postTitle').value = '';
                    document.getElementById('postCat').value = '';
                    await fetchPosts();
                    alert('✅ 双语文章创建成功！\n中文版: ' + (data.data?.zh_path || '已创建') + '\n英文版: ' + (data.data?.en_path || '已创建') + '\n\n💡 提示：英文版标题已自动翻译');
                } else {
                    alert('❌ 创建失败: ' + data.message);
                }
            } catch(e) {
                alert('❌ 网络错误: ' + e);
            }
        }

        function insertCodeBlock() {
            const textarea = document.getElementById('editor-textarea');
            if(!textarea) return;

            const language = prompt('请输入代码语言 (如: javascript, python, go, bash 等):', 'javascript');
            if(language === null) return;

            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            const selectedText = textarea.value.substring(start, end);
            
            const tick = String.fromCharCode(96);
            const codeBlock = tick + tick + tick + language + '\\n' + (selectedText || '// 在这里输入代码\\n') + '\\n' + tick + tick + tick + '\\n\\n';
            
            textarea.value = textarea.value.substring(0, start) + codeBlock + textarea.value.substring(end);
            
            const newCursorPos = start + language.length + 4;
            textarea.setSelectionRange(newCursorPos, newCursorPos);
            textarea.focus();
        }

        function insertImage() {
            const textarea = document.getElementById('editor-textarea');
            if(!textarea) return;

            const imageUrl = prompt('请输入图片 URL 或路径\n(例如: /img/photo.jpg 或 https://example.com/image.png):', '');
            if(!imageUrl) return;

            const altText = prompt('请输入图片描述 (可选):', '图片');
            const width = prompt('图片宽度 (如: 500px, 80%, 留空为原始大小):', '');
            const align = prompt('对齐方式\n输入: left (左对齐), center (居中), right (右对齐)\n留空为默认', 'center');
            
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            
            let imageHtml = '<div style="text-align: ' + (align || 'center') + ';">\n';
            imageHtml += '  <img src="' + imageUrl + '" alt="' + (altText || '图片') + '"';
            if(width) {
                imageHtml += ' style="width: ' + width + '; height: auto;"';
            }
            imageHtml += '>\n';
            imageHtml += '</div>\n\n';
            
            textarea.value = textarea.value.substring(0, start) + imageHtml + textarea.value.substring(end);
            
            const newCursorPos = start + imageHtml.length;
            textarea.setSelectionRange(newCursorPos, newCursorPos);
            textarea.focus();
        }

        async function runCommand(cmd) {
            // 对于预览命令，先自动保存当前编辑内容
            if(cmd === 'preview' && currentDocPath) {
                console.log('Preview: Auto-saving current document...');
                const saveOk = await saveDocument();
                if(!saveOk) {
                    alert('⚠️  预览前保存失败，请检查');
                    return;
                }
                // 等待保存完成
                await new Promise(resolve => setTimeout(resolve, 1500));
            }
            
            try {
                const res = await authFetch('/api/command?name=' + cmd);
                const data = await res.json();
                
                // 对于预览命令，直接打开本地浏览器
                if(cmd === 'preview') {
                    alert(data.message || '✅ 预览已启动！\n\n包括所有草稿文章和最新修改\n浏览器即将打开...');
                    // 给浏览器打开的时间
                    setTimeout(() => {
                        window.open('http://localhost:1313/WangScape/', '_blank');
                    }, 800);
                } else if(data.data && data.data.url) {
                    window.open(data.data.url, '_blank');
                } else {
                    alert('系统: ' + (data.message || data.data?.message || '命令已执行'));
                }
            } catch(e) {
                alert('❌ 命令执行失败: ' + e);
            }
        }

        function switchCommentView() {
            if (!currentDocPath) {
                alert('⚠️ 请先选择一篇文章');
                return;
            }
            const metaPanel = document.getElementById('meta-panel');
            const commentsPanel = document.getElementById('comments-panel');
            
            if (commentsPanel.classList.contains('hide')) {
                commentsPanel.classList.remove('hide');
                commentsPanel.classList.add('show');
                metaPanel.style.display = 'none';
                loadComments(currentDocPath);
            } else {
                commentsPanel.classList.remove('show');
                commentsPanel.classList.add('hide');
                metaPanel.style.display = 'block';
            }
        }

        async function loadComments(postPath) {
            try {
                const res = await authFetch('/api/all_comments?path=' + encodeURIComponent(postPath));
                const data = await res.json();
                
                let html = '';
                if (data.data && data.data.length > 0) {
                    data.data.forEach(comment => {
                        const statusBadge = comment.approved ? 
                            '<span style="color:#4CAF50; font-weight:bold;">已批准</span>' : 
                            '<span style="color:#FF9800; font-weight:bold;">待审核</span>';
                        
                        const bg = comment.approved ? '#f9f9f9' : '#fffbf0';
                        const approveBtn = !comment.approved ? 
                            '<button onclick="approveComment(\'' + postPath + '\', \'' + comment.id + '\')" style="padding: 5px 10px; background: #4CAF50; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 12px;">批准</button>' : '';
                        
                        const ipInfo = comment.ip_address ? '<div style="font-size: 11px; color: #999; margin-top: 5px;"><strong>IP:</strong> ' + escapeHtml(comment.ip_address) + '</div>' : '';
                        const uaInfo = comment.user_agent ? '<div style="font-size: 11px; color: #999; margin-top: 2px; word-break: break-all;"><strong>UA:</strong> ' + escapeHtml(comment.user_agent) + '</div>' : '';
                        
                        html += '<div style="border: 1px solid #ddd; padding: 15px; margin-bottom: 10px; border-radius: 6px; background: ' + bg + ';">' +
                            '<div style="display: flex; justify-content: space-between; align-items: start; margin-bottom: 10px;">' +
                            '<div>' +
                            '<strong>' + escapeHtml(comment.author) + '</strong>' +
                            '<span style="font-size: 12px; color: #999;"> · ' + comment.timestamp + '</span>' +
                            '</div>' +
                            statusBadge +
                            '</div>' +
                            '<p style="margin: 10px 0; color: #333; word-break: break-word;">' + escapeHtml(comment.content) + '</p>' +
                            ipInfo + uaInfo +
                            '<div style="display: flex; gap: 10px; margin-top: 10px;">' +
                            approveBtn +
                            '<button onclick="deleteCommentConfirm(\'' + postPath + '\', \'' + comment.id + '\')" style="padding: 5px 10px; background: #f44336; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 12px;">删除</button>' +
                            '</div>' +
                            '</div>';
                    });
                } else {
                    html = '<div style="text-align: center; color: #999; padding: 40px;">暂无评论</div>';
                }
                
                document.getElementById('comments-list').innerHTML = html;
            } catch (e) {
                document.getElementById('comments-list').innerHTML = '<div style="color: red;">加载失败: ' + e + '</div>';
            }
        }

        async function approveComment(postPath, commentId) {
            try {
                const res = await authFetch('/api/approve_comment', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ post_path: postPath, comment_id: commentId })
                });
                const data = await res.json();
                if (data.success) {
                    alert('✅ 评论已批准');
                    loadComments(postPath);
                } else {
                    alert('❌ 批准失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }

        function deleteCommentConfirm(postPath, commentId) {
            if (confirm('确定要删除这条评论吗？此操作不可恢复。')) {
                deleteCommentAction(postPath, commentId);
            }
        }

        async function deleteCommentAction(postPath, commentId) {
            try {
                const res = await authFetch('/api/delete_comment', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ post_path: postPath, comment_id: commentId })
                });
                const data = await res.json();
                if (data.success) {
                    alert('✅ 评论已删除');
                    loadComments(postPath);
                } else {
                    alert('❌ 删除失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }

        async function loadStatistics() {
                        try {
                            const resp = await authFetch('/api/statistics');
                
                            if (!resp.ok) {
                                alert('❌ 无权查看统计数据，请先登录');
                                switchView('dashboard');
                                return;
                            }
                
                            const result = await resp.json();
                            if (!result.success) {
                                alert('❌ 加载统计失败: ' + result.message);
                                return;
                            }
                
                            const stats = result.data;
                
                            // 更新概览数字
                            document.getElementById('stat-total-pages').innerText = stats.total_pages;
                            document.getElementById('stat-total-views').innerText = stats.total_views;
                            document.getElementById('stat-total-comments').innerText = stats.total_comments;
                            document.getElementById('stat-unique-ips').innerText = stats.unique_ips;
                            document.getElementById('stats-update-time').innerText = new Date().toLocaleString('zh-CN');
                
                            // 显示最受欢迎的文章
                            const pagesEl = document.getElementById('stat-top-pages');
                            if (stats.pages && stats.pages.length > 0) {
                                pagesEl.innerHTML = '<table style="width:100%; border-collapse: collapse;">' +
                                    '<tr style="background: #f8fafc; border-bottom: 1px solid var(--dash-border);">' +
                                    '<th style="padding: 12px; text-align: left;">📄 页面标题</th>' +
                                    '<th style="padding: 12px; text-align: center; width: 100px;">👁️ 浏览量</th>' +
                                    '</tr>' +
                                    stats.pages.map(function (p) {
                                        return '<tr style="border-bottom: 1px solid var(--dash-border);">' +
                                            '<td style="padding: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + p.title + '</td>' +
                                            '<td style="padding: 12px; text-align: center; color: var(--dash-accent); font-weight: 700;">' + p.views + '</td>' +
                                            '</tr>';
                                    }).join('') +
                                    '</table>';
                            } else {
                                pagesEl.innerHTML = '<div style="padding: 20px; text-align: center; color: var(--dash-text-dim);">暂无数据</div>';
                            }
                
                            // 显示访客IP统计
                            const visitorsEl = document.getElementById('stat-visitors');
                            if (stats.visitors && stats.visitors.length > 0) {
                                visitorsEl.innerHTML = '<table style="width:100%; border-collapse: collapse;">' +
                                    '<tr style="background: #f8fafc; border-bottom: 1px solid var(--dash-border);">' +
                                    '<th style="padding: 12px; text-align: left; width: 150px;">🌐 IP地址</th>' +
                                    '<th style="padding: 12px; text-align: center; width: 80px;">💬 评论数</th>' +
                                    '<th style="padding: 12px; text-align: left; flex: 1;">⏱️ 首次出现</th>' +
                                    '<th style="padding: 12px; text-align: left; flex: 1;">⏱️ 最后更新</th>' +
                                    '</tr>' +
                                    stats.visitors.map(function (v) {
                                        return '<tr style="border-bottom: 1px solid var(--dash-border);">' +
                                            '<td style="padding: 12px; font-family: monospace; font-weight: 600; color: #3b82f6;">' + v.ip + '</td>' +
                                            '<td style="padding: 12px; text-align: center; color: var(--dash-accent); font-weight: 700;">' + v.comment_count + '</td>' +
                                            '<td style="padding: 12px; font-size: 12px; color: var(--dash-text-dim);">' + v.first_seen + '</td>' +
                                            '<td style="padding: 12px; font-size: 12px; color: var(--dash-text-dim);">' + v.last_seen + '</td>' +
                                            '</tr>';
                                    }).join('') +
                                    '</table>';
                            } else {
                                visitorsEl.innerHTML = '<div style="padding: 20px; text-align: center; color: var(--dash-text-dim);">暂无访客数据</div>';
                            }
                        } catch (e) {
                            console.error('加载统计失败:', e);
                            alert('❌ 加载统计失败: ' + e.message);
                        }
                    }

        async function loadPendingComments() {
            const listEl = document.getElementById('pending-comments-list');
            const countEl = document.getElementById('pending-total-count');
            const selectAll = document.getElementById('pending-select-all');
            
            listEl.innerHTML = '<div style="text-align:center; padding:40px; color:#999;">加载中...</div>';
            if (selectAll) selectAll.checked = false;
            
            try {
                const res = await authFetch('/api/pending_comments');
                const data = await res.json();
                
                if (data.success && data.data) {
                    const comments = data.data;
                    console.log('[DEBUG] 加载了', comments.length, '条待审核评论');
                    console.log('[DEBUG] 第一条评论数据:', comments[0]);
                    
                    if (comments.length === 0) {
                        listEl.innerHTML = '<div style="text-align:center; padding:60px; color:#999; font-size:16px;">🎉 没有待审核的评论</div>';
                        countEl.textContent = '0 条待审核';
                        return;
                    }
                    
                    countEl.textContent = comments.length + ' 条待审核';
                    const headerCountEl = document.getElementById('pending-total-count-header');
                    if (headerCountEl) {
                        headerCountEl.textContent = comments.length + ' 条待审核';
                    }
                    
                    let html = '';
                    comments.forEach((item, index) => {
                        const c = item;
                        const commentNum = comments.length - index;
                        const truncatedContent = c.content ? (c.content.length > 150 ? c.content.substring(0, 150) + '...' : c.content) : '无内容';
                        
                        let formattedTime = '刚刚';
                        if (c.timestamp) {
                            try {
                                const date = new Date(c.timestamp);
                                if (!isNaN(date.getTime())) {
                                    formattedTime = date.toLocaleString('zh-CN', { 
                                        year: 'numeric',
                                        month: '2-digit',
                                        day: '2-digit',
                                        hour: '2-digit',
                                        minute: '2-digit'
                                    });
                                }
                            } catch(e) {
                                formattedTime = c.timestamp.substring(0, 19);
                            }
                        }
                        
                        html += '<div class="pending-comment-card">' +
                            '<div class="comment-header">' +
                            '<div class="comment-number">#' + commentNum + '</div>' +
                            '<input type="checkbox" class="pending-select" onchange="updateSelectedCount()" data-post="' + c.post_path.replace(/\\/g, '\\\\') + '" data-id="' + c.id + '" data-issue-number="' + (c.issue_number || '') + '" />' +
                            '</div>' +
                            '<div class="comment-post-title">' +
                            '<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M2 2.5A2.5 2.5 0 014.5 0h8.75a.75.75 0 01.75.75v12.5a.75.75 0 01-.75.75h-2.5a.75.75 0 110-1.5h1.75v-2h-8a1 1 0 00-.714 1.7.75.75 0 01-1.072 1.05A2.495 2.495 0 012 11.5v-9zm10.5-1V9h-8c-.356 0-.694.074-1 .208V2.5a1 1 0 011-1h8zM5 12.25v3.25a.25.25 0 00.4.2l1.45-1.087a.25.25 0 01.3 0L8.6 15.7a.25.25 0 00.4-.2v-3.25a.25.25 0 00-.25-.25h-3.5a.25.25 0 00-.25.25z"></path></svg>' +
                            escapeHtml(c.post_title || '未知文章') +
                            '</div>' +
                            '<div class="comment-user-info">' +
                            '<div class="user-avatar">' + (c.author ? c.author.substring(0, 1).toUpperCase() : 'U') + '</div>' +
                            '<div class="user-details">' +
                            '<div class="comment-author">' + escapeHtml(c.author || '匿名') + '</div>' +
                            '<div class="comment-meta">' +
                            '<span class="meta-item"><svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor"><path d="M1.75 2A1.75 1.75 0 003.5 3.75v8.5A1.75 1.75 0 001.75 14h-1.5A.25.25 0 010 13.75v-12A.25.25 0 01.25 1.5h1.5zM11 2a1 1 0 00-1 1v10a1 1 0 001 1h3a1 1 0 001-1V3a1 1 0 00-1-1h-3z"></path></svg> ' + escapeHtml(c.email || '未提供') + '</span>' +
                            '<span class="meta-item"><svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor"><path d="M1.5 8a6.5 6.5 0 1113 0 6.5 6.5 0 01-13 0zM8 0a8 8 0 100 16A8 8 0 008 0zm.5 4.75a.75.75 0 00-1.5 0v3.5a.75.75 0 00.471.696l2.5 1a.75.75 0 00.557-1.392L8.5 7.742V4.75z"></path></svg> ' + formattedTime + '</span>' +
                            '</div>' +
                            '</div>' +
                            '</div>' +
                            '<div class="comment-content-preview">' + escapeHtml(truncatedContent) + '</div>' +
                            '<div class="comment-tech-info">' +
                            '<div class="tech-item"><svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor"><path d="M8 0a8 8 0 110 16A8 8 0 018 0zM1.5 8a6.5 6.5 0 1013 0 6.5 6.5 0 00-13 0zm9.78-2.22a.75.75 0 00-1.06-1.06L6.75 8.19 5.28 6.72a.75.75 0 00-1.06 1.06l2 2a.75.75 0 001.06 0l3.5-3.5z"></path></svg> <strong>IP:</strong> ' + escapeHtml(c.ip_address || '未记录') + '</div>' +
                            '<div class="tech-item tech-ua"><svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor"><path d="M2 2.5A2.5 2.5 0 014.5 0h8.75a.75.75 0 01.75.75v12.5a.75.75 0 01-.75.75h-2.5a.75.75 0 110-1.5h1.75v-2h-8a1 1 0 00-.714 1.7.75.75 0 01-1.072 1.05A2.495 2.495 0 012 11.5v-9zm10.5-1V9h-8c-.356 0-.694.074-1 .208V2.5a1 1 0 011-1h8z"></path></svg> <strong>UA:</strong> ' + (c.user_agent ? escapeHtml(c.user_agent.length > 80 ? c.user_agent.substring(0, 80) + '...' : c.user_agent) : '未记录') + '</div>' +
                            '</div>' +
                            '<div class="comment-actions">' +
                            '<button class="btn-approve" data-action="approve" data-post-path="' + c.post_path + '" data-comment-id="' + c.id + '" data-issue-number="' + (c.issue_number || '') + '">✅ 批准</button>' +
                            '<button class="btn-edit" data-action="edit" data-post-path="' + c.post_path + '" data-comment-id="' + c.id + '" data-issue-number="' + (c.issue_number || '') + '" data-author="' + (c.author || '') + '" data-email="' + (c.email || '') + '" data-content="' + (c.content || '').replace(/"/g, '&quot;') + '">✏️ 编辑</button>' +
                            '<button class="btn-delete" data-action="delete" data-post-path="' + c.post_path + '" data-comment-id="' + c.id + '" data-issue-number="' + (c.issue_number || '') + '">🗑 删除</button>' +
                            '</div>' +
                            '</div>';
                    });
                    
                    listEl.innerHTML = html;
                    
                    // 添加按钮点击事件监听
                    listEl.querySelectorAll('.comment-actions button').forEach(btn => {
                        btn.addEventListener('click', function(e) {
                            e.preventDefault();
                            const action = this.getAttribute('data-action');
                            const postPath = this.getAttribute('data-post-path');
                            const commentId = this.getAttribute('data-comment-id');
                            const issueNumberStr = this.getAttribute('data-issue-number');
                            const issueNumber = issueNumberStr ? parseInt(issueNumberStr) : null;
                            
                            console.log('[DEBUG] 按钮点击:', { action, postPath, commentId, issueNumber, issueNumberStr });
                            
                            if (action === 'approve') {
                                approvePendingComment(postPath, commentId, issueNumber);
                            } else if (action === 'edit') {
                                const author = this.getAttribute('data-author');
                                const email = this.getAttribute('data-email');
                                const content = this.getAttribute('data-content');
                                editPendingComment(postPath, commentId, issueNumber, author, email, content);
                            } else if (action === 'delete') {
                                deletePendingComment(postPath, commentId, issueNumber);
                            }
                        });
                    });
                } else {
                    listEl.innerHTML = '<div style="text-align:center; padding:40px; color:red;">加载失败</div>';
                }
            } catch (e) {
                listEl.innerHTML = '<div style="text-align:center; padding:40px; color:red;">网络错误: ' + e + '</div>';
            }
        }

        function getSelectedPendingItems() {
            const checks = document.querySelectorAll('.pending-select:checked');
            const items = [];
            checks.forEach(ch => {
                const item = {
                    post_path: ch.getAttribute('data-post'),
                    comment_id: ch.getAttribute('data-id')
                };
                // 添加issue_number如果存在
                const issueNumber = ch.getAttribute('data-issue-number');
                if (issueNumber) {
                    item.issue_number = parseInt(issueNumber);
                }
                items.push(item);
            });
            return items;
        }

        function toggleSelectAllPending() {
            const selectAll = document.getElementById('pending-select-all');
            const checks = document.querySelectorAll('.pending-select');
            checks.forEach(ch => ch.checked = selectAll.checked);
            updateSelectedCount();
        }
        
        function updateSelectedCount() {
            const checks = document.querySelectorAll('.pending-select:checked');
            const countEl = document.getElementById('selected-count');
            if (countEl) {
                if (checks.length === 0) {
                    countEl.textContent = '未选择';
                    countEl.style.color = '#6b7280';
                } else {
                    countEl.textContent = '已选择 ' + checks.length + ' 条';
                    countEl.style.color = '#059669';
                    countEl.style.fontWeight = '600';
                }
            }
        }

        async function bulkApprovePending() {
            const items = getSelectedPendingItems();
            if (items.length === 0) {
                alert('请选择要批准的评论');
                return;
            }
            try {
                const res = await authFetch('/api/bulk_comments', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'approve', items: items })
                });
                const data = await res.json();
                if (data.success) {
                    alert('✅ 批量批准完成');
                    loadPendingComments();
                } else {
                    alert('❌ 批量批准失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }

        async function bulkDeletePending() {
            const items = getSelectedPendingItems();
            if (items.length === 0) {
                alert('请选择要删除的评论');
                return;
            }
            if (!confirm('确定要批量删除所选评论吗？此操作不可恢复。')) return;
            try {
                const res = await authFetch('/api/bulk_comments', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'delete', items: items })
                });
                const data = await res.json();
                if (data.success) {
                    alert('✅ 批量删除完成');
                    loadPendingComments();
                } else {
                    alert('❌ 批量删除失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }

        async function exportCommentsCsv() {
            try {
                const res = await authFetch('/api/export_comments');
                if (!res.ok) {
                    alert('❌ 导出失败');
                    return;
                }
                const blob = await res.blob();
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'comments.csv';
                document.body.appendChild(a);
                a.click();
                a.remove();
                URL.revokeObjectURL(url);
            } catch (e) {
                alert('❌ 导出失败: ' + e);
            }
        }

        async function loadCommentSettings() {
            try {
                const res = await authFetch('/api/comment_settings');
                const data = await res.json();
                if (data.success && data.data) {
                    const s = data.data;
                    document.getElementById('smtp-enabled').checked = !!s.smtp_enabled;
                    document.getElementById('smtp-host').value = s.smtp_host || '';
                    document.getElementById('smtp-port').value = s.smtp_port || 587;
                    document.getElementById('smtp-user').value = s.smtp_user || '';
                    document.getElementById('smtp-pass').value = s.smtp_pass || '';
                    document.getElementById('smtp-from').value = s.smtp_from || '';
                    document.getElementById('smtp-to').value = (s.smtp_to || []).join(',');
                    document.getElementById('notify-pending').checked = !!s.notify_on_pending;
                    document.getElementById('blacklist-ips').value = (s.blacklist_ips || []).join('\n');
                    document.getElementById('blacklist-words').value = (s.blacklist_keywords || []).join('\n');
                }
            } catch (e) {
                console.error('加载评论设置失败:', e);
            }
        }

        async function testMailConnection() {
            const smtpHost = document.getElementById('smtp-host').value.trim();
            const smtpPort = parseInt(document.getElementById('smtp-port').value || '587', 10);
            const smtpUser = document.getElementById('smtp-user').value.trim();
            const smtpPass = document.getElementById('smtp-pass').value.trim();
            const smtpFrom = document.getElementById('smtp-from').value.trim();
            const smtpTo = document.getElementById('smtp-to').value.trim();

            if (!smtpHost || !smtpPort || !smtpUser || !smtpPass || !smtpTo) {
                alert('❌ 请填写所有SMTP配置字段');
                return;
            }

            try {
                const res = await authFetch('/api/test_mail', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        smtp_host: smtpHost,
                        smtp_port: smtpPort,
                        smtp_user: smtpUser,
                        smtp_pass: smtpPass,
                        smtp_from: smtpFrom || smtpUser,
                        smtp_to: smtpTo
                    })
                });
                const data = await res.json();
                if (data.success) {
                    alert('✅ ' + data.message);
                } else {
                    alert('❌ 测试失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }

        async function saveCommentSettings() {
            const payload = {
                smtp_enabled: document.getElementById('smtp-enabled').checked,
                smtp_host: document.getElementById('smtp-host').value.trim(),
                smtp_port: parseInt(document.getElementById('smtp-port').value || '587', 10),
                smtp_user: document.getElementById('smtp-user').value.trim(),
                smtp_pass: document.getElementById('smtp-pass').value.trim(),
                smtp_from: document.getElementById('smtp-from').value.trim(),
                smtp_to: document.getElementById('smtp-to').value.split(',').map(s => s.trim()).filter(Boolean),
                notify_on_pending: document.getElementById('notify-pending').checked,
                blacklist_ips: document.getElementById('blacklist-ips').value.split('\n').map(s => s.trim()).filter(Boolean),
                blacklist_keywords: document.getElementById('blacklist-words').value.split('\n').map(s => s.trim()).filter(Boolean)
            };

            try {
                const res = await authFetch('/api/save_comment_settings', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                const data = await res.json();
                if (data.success) {
                    alert('✅ 设置已保存');
                } else {
                    alert('❌ 保存失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }
        
        async function approvePendingComment(postPath, commentId, issueNumber) {
            console.log('[DEBUG] approvePendingComment called:', { postPath, commentId, issueNumber });
            try {
                const payload = {
                    post_path: postPath,
                    comment_id: commentId
                };
                
                // 如果有issue_number，添加到请求中
                if (issueNumber) {
                    payload.issue_number = parseInt(issueNumber);
                }
                
                const res = await authFetch('/api/approve_comment', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                const data = await res.json();
                if (data.success) {
                    alert('✅ 评论已批准');
                    addOperationHistory('comment', 'approve', postPath, '评论ID: ' + commentId);
                    loadPendingComments();
                } else {
                    alert('❌ 批准失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }
        
        function editPendingComment(postPath, commentId, issueNumber, currentAuthor, currentEmail, currentContent) {
            const modal = document.createElement('div');
            modal.className = 'modal-overlay';
            modal.innerHTML = 
                '<div class="modal-card" style="max-width: 600px;">' +
                    '<h2 style="margin-top: 0;">✏️ 编辑评论</h2>' +
                    '<div style="margin-bottom: 15px;">' +
                        '<label style="display: block; margin-bottom: 5px; font-weight: bold;">作者名称：</label>' +
                        '<input type="text" id="edit-author" value="' + escapeHtml(currentAuthor) + '" style="width: 100%; padding: 8px; border: 1px solid #e2e8f0; border-radius: 4px;" />' +
                    '</div>' +
                    '<div style="margin-bottom: 15px;">' +
                        '<label style="display: block; margin-bottom: 5px; font-weight: bold;">邮箱：</label>' +
                        '<input type="email" id="edit-email" value="' + escapeHtml(currentEmail) + '" style="width: 100%; padding: 8px; border: 1px solid #e2e8f0; border-radius: 4px;" />' +
                    '</div>' +
                    '<div style="margin-bottom: 15px;">' +
                        '<label style="display: block; margin-bottom: 5px; font-weight: bold;">评论内容：</label>' +
                        '<textarea id="edit-content" rows="6" style="width: 100%; padding: 8px; border: 1px solid #e2e8f0; border-radius: 4px; resize: vertical;">' + escapeHtml(currentContent) + '</textarea>' +
                    '</div>' +
                    '<div style="display: flex; gap: 10px; justify-content: flex-end;">' +
                        '<button onclick="this.closest(\'.modal-overlay\').remove()" style="padding: 8px 16px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer;">取消</button>' +
                        '<button onclick="saveEditedComment(\'' + postPath.replace(/\\/g, '\\\\') + '\', \'' + commentId + '\', ' + (issueNumber || 'null') + ')" style="padding: 8px 16px; background: #0ea5e9; color: white; border: none; border-radius: 4px; cursor: pointer;">💾 保存</button>' +
                    '</div>' +
                '</div>';
            document.body.appendChild(modal);
            modal.style.display = 'flex';
        }
        
        async function saveEditedComment(postPath, commentId, issueNumber) {
            console.log('[DEBUG] saveEditedComment called:', { postPath, commentId, issueNumber });
            const author = document.getElementById('edit-author').value.trim();
            const email = document.getElementById('edit-email').value.trim();
            const content = document.getElementById('edit-content').value.trim();
            
            if (!author || !email || !content) {
                alert('请填写所有字段');
                return;
            }
            
            try {
                const payload = {
                    post_path: postPath,
                    comment_id: commentId,
                    author: author,
                    email: email,
                    content: content
                };
                
                if (issueNumber) {
                    payload.issue_number = parseInt(issueNumber);
                }
                
                const res = await authFetch('/api/update_comment', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                const data = await res.json();
                if (data.success) {
                    alert('✅ 评论已更新');
                    document.querySelector('.modal-overlay').remove();
                    addOperationHistory('comment', 'edit', postPath, '评论ID: ' + commentId);
                    loadPendingComments();
                } else {
                    alert('❌ 更新失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }
        
        function deletePendingComment(postPath, commentId, issueNumber) {
            if (confirm('确定要删除这条评论吗？此操作不可恢复。')) {
                deletePendingCommentAction(postPath, commentId, issueNumber);
            }
        }
        
        async function deletePendingCommentAction(postPath, commentId, issueNumber) {
            console.log('[DEBUG] deletePendingCommentAction called:', { postPath, commentId, issueNumber });
            try {
                const payload = { 
                    post_path: postPath, 
                    comment_id: commentId
                };
                
                // 如果有issue_number，添加到请求中
                if (issueNumber) {
                    payload.issue_number = parseInt(issueNumber);
                }
                
                const res = await authFetch('/api/delete_comment', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                console.log('[DEBUG] deletePendingCommentAction response status:', res.status);
                const data = await res.json();
                console.log('[DEBUG] deletePendingCommentAction response data:', data);
                if (data.success) {
                    alert('✅ 评论已删除');
                    addOperationHistory('comment', 'delete', postPath, '评论ID: ' + commentId);

                    loadPendingComments();
                } else {
                    alert('❌ 删除失败: ' + data.message);
                }
            } catch (e) {
                alert('❌ 错误: ' + e);
            }
        }

        function updateWordCount() {
            const textarea = document.getElementById('editor-textarea');
            if (!textarea) return;
            
            const content = textarea.value;
            const bodyContent = content.replace(/^---[\s\S]*?---\n/, '');
            const chineseChars = (bodyContent.match(/[\u4e00-\u9fa5]/g) || []).length;
            const englishWords = (bodyContent.match(/[a-zA-Z]+/g) || []).length;
            const totalWords = chineseChars + englishWords;
            
            const wordCountEl = document.getElementById('word-count');
            if (wordCountEl) {
                wordCountEl.textContent = '字数: ' + totalWords.toLocaleString();
            }
        }
        
        function toggleMetadataPanel() {
            const metaPanel = document.getElementById('meta-panel');
            const commentsPanel = document.getElementById('comments-panel');
            
            if (metaPanel.style.display === 'none') {
                metaPanel.style.display = 'block';
                commentsPanel.classList.add('hide');
                commentsPanel.classList.remove('show');
            } else {
                metaPanel.style.display = 'none';
            }
        }
        
        function insertTable() {
            const textarea = document.getElementById('editor-textarea');
            if (!textarea) return;
            
            const rows = prompt('请输入表格行数：', '3');
            const cols = prompt('请输入表格列数：', '3');
            
            if (!rows || !cols || isNaN(rows) || isNaN(cols)) return;
            
            const numRows = parseInt(rows);
            const numCols = parseInt(cols);
            
            let table = '\n| ';
            for (let i = 0; i < numCols; i++) {
                table += '列' + (i + 1) + ' | ';
            }
            table += '\n| ';
            for (let i = 0; i < numCols; i++) {
                table += '--- | ';
            }
            
            for (let i = 0; i < numRows; i++) {
                table += '\n| ';
                for (let j = 0; j < numCols; j++) {
                    table += '内容 | ';
                }
            }
            table += '\n\n';
            
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            textarea.value = textarea.value.substring(0, start) + table + textarea.value.substring(end);
            textarea.focus();
            updateWordCount();
        }
        
        function insertMarkdown(before, after) {
            const textarea = document.getElementById('editor-textarea');
            if (!textarea) return;
            
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            const selectedText = textarea.value.substring(start, end);
            
            const newText = before + (selectedText || '文本') + after;
            textarea.value = textarea.value.substring(0, start) + newText + textarea.value.substring(end);
            
            if (selectedText) {
                textarea.setSelectionRange(start, start + newText.length);
            } else {
                textarea.setSelectionRange(start + before.length, start + before.length + 2);
            }
            textarea.focus();
        }

        function escapeHtml(text) {
            const map = {
                '&': '&amp;',
                '<': '&lt;',
                '>': '&gt;',
                '"': '&quot;',
                "'": '&#039;'
            };
            return text.replace(/[&<>"']/g, m => map[m]);
        }

        fetchPosts();
        fetchCommentStats();
        updateAuthStatus();
        
        // 快捷键支持
        document.addEventListener('keydown', function(e) {
            if (e.ctrlKey && e.key === 's') {
                e.preventDefault();
                saveDocument();
            }
            if (e.ctrlKey && e.key === 'b') {
                e.preventDefault();
                insertMarkdown('**', '**');
            }
            if (e.ctrlKey && e.key === 'i') {
                e.preventDefault();
                insertMarkdown('*', '*');
            }
            if (e.ctrlKey && e.key === 'k') {
                e.preventDefault();
                const backtick = String.fromCharCode(96);
                insertMarkdown(backtick, backtick);
            }
        });

        // ===== 操作历史管理 =====
        let operationHistory = [];

        function loadOperationHistory() {
            const stored = localStorage.getItem('operationHistory');
            operationHistory = stored ? JSON.parse(stored) : [];
            
            const filterComments = document.getElementById('filter-comments')?.checked ?? true;
            const filterPosts = document.getElementById('filter-posts')?.checked ?? true;
            const dateFrom = document.getElementById('history-date-from')?.value;
            const dateTo = document.getElementById('history-date-to')?.value;
            
            let filtered = operationHistory.filter(op => {
                let typeMatch = (op.type === 'comment' && filterComments) || (op.type === 'post' && filterPosts);
                let dateMatch = true;
                if (dateFrom) {
                    dateMatch = dateMatch && new Date(op.timestamp) >= new Date(dateFrom);
                }
                if (dateTo) {
                    const nextDay = new Date(dateTo);
                    nextDay.setDate(nextDay.getDate() + 1);
                    dateMatch = dateMatch && new Date(op.timestamp) < nextDay;
                }
                return typeMatch && dateMatch;
            });
            
            // 按时间倒序排列
            filtered.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
            
            // 更新统计
            document.getElementById('total-ops').textContent = filtered.length;
            document.getElementById('total-comment-ops').textContent = filtered.filter(op => op.type === 'comment').length;
            document.getElementById('total-post-ops').textContent = filtered.filter(op => op.type === 'post').length;
            
            // 渲染列表
            const historyList = document.getElementById('history-list');
            historyList.innerHTML = filtered.length === 0 
                ? '<div style="text-align:center; padding:40px 20px; color:#999; font-size:14px;">暂无操作记录</div>'
                : filtered.map(op => renderHistoryItem(op)).join('');
        }

        function renderHistoryItem(op) {
            const date = new Date(op.timestamp);
            const timeStr = date.toLocaleString('zh-CN', {year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit'});
            const icon = getOperationIcon(op.action);
            const color = getOperationColor(op.action);
            const actionText = getOperationText(op.action);
            const typeText = op.type === 'comment' ? '💬评论' : '📝文章';
            const opTypeText = op.type === 'comment' ? '评论操作' : '文章操作';
            const titleDisplay = op.title ? ' - ' + escapeHtml(op.title.substring(0, 50)) : '';
            const detailsHtml = op.details ? '<div style="font-size:12px; color:#6b7280; padding:8px; background:#f9fafb; border-radius:4px; margin-top:8px;">' + escapeHtml(op.details) + '</div>' : '';
            
            let html = '<div style="background:white; padding:15px; border-radius:8px; border-left:4px solid ' + color + '; box-shadow:0 1px 3px rgba(0,0,0,0.1);">' +
                '<div style="display:flex; justify-content:space-between; align-items:start; margin-bottom:8px;">' +
                '<div style="display:flex; align-items:center; gap:10px; flex:1;">' +
                '<span style="font-size:24px;">' + icon + '</span>' +
                '<div style="flex:1;">' +
                '<div style="font-weight:600; color:#1f2937;">' +
                '[' + typeText + '] ' + actionText + titleDisplay +
                '</div>' +
                '<div style="font-size:12px; color:#6b7280; margin-top:4px;">' + timeStr + '</div>' +
                '</div>' +
                '</div>' +
                '<div style="text-align:right;">' +
                '<span style="display:inline-block; padding:4px 8px; background:' + color + '20; color:' + color + '; border-radius:4px; font-size:12px; font-weight:600;">' +
                opTypeText +
                '</span>' +
                '</div>' +
                '</div>' +
                detailsHtml +
                '</div>';
            return html;
        }

        function getOperationIcon(action) {
            const icons = {
                'approve': '✅',
                'reject': '❌',
                'delete': '🗑️',
                'publish': '📤',
                'create': '✨',
                'edit': '✏️',
                'unpublish': '🔒'
            };
            return icons[action] || '📌';
        }

        function getOperationColor(action) {
            const colors = {
                'approve': '#22c55e',
                'reject': '#ef4444',
                'delete': '#f97316',
                'publish': '#3b82f6',
                'create': '#8b5cf6',
                'edit': '#f59e0b',
                'unpublish': '#6b7280'
            };
            return colors[action] || '#6b7280';
        }

        function getOperationText(action) {
            const texts = {
                'approve': '已批准',
                'reject': '已拒绝',
                'delete': '已删除',
                'publish': '已发布',
                'create': '新建文章',
                'edit': '编辑文章',
                'unpublish': '已下架'
            };
            return texts[action] || action;
        }

        function addOperationHistory(type, action, title = '', details = '') {
            operationHistory.push({
                type: type, // 'comment' 或 'post'
                action: action, // 'approve', 'reject', 'delete', 'publish', 'create', 'edit', 'unpublish'
                title: title,
                details: details,
                timestamp: new Date().toISOString()
            });
            localStorage.setItem('operationHistory', JSON.stringify(operationHistory));
        }

        function exportHistoryCsv() {
            if (operationHistory.length === 0) {
                alert('没有操作记录可导出');
                return;
            }
            
            const headers = ['操作类型', '操作', '标题', '详情', '时间'];
            const rows = operationHistory.map(op => [
                op.type === 'comment' ? '评论' : '文章',
                getOperationText(op.action),
                op.title || '-',
                op.details || '-',
                new Date(op.timestamp).toLocaleString('zh-CN')
            ]);
            
            const csv = [headers, ...rows].map(row => 
                row.map(cell => '"' + String(cell).replace(/"/g, '""') + '"')
                   .join(',')
            ).join('\n');
            
            const blob = new Blob([csv], {type: 'text/csv;charset=utf-8;'});
            const link = document.createElement('a');
            link.href = URL.createObjectURL(blob);
            const dateStr = new Date().toISOString().split('T')[0];
            link.download = '操作历史_' + dateStr + '.csv';
            link.click();
        }

        // 初始化历史记录
        loadOperationHistory();
    </script>
</body>
</html>`
