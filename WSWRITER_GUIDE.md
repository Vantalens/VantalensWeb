# WSwriter 使用指南

WangScape的Go后端服务，提供博客管理、评论系统、内容同步等功能。

---

## 📋 快速开始

### 环境要求
- Go 1.16+ (编译运行)
- Hugo (可选，用于网站预览)
- 现代浏览器

### 启动方式

**方式1：编译运行（推荐）**
```bash
cd d:\WangScape
go build -o WSwriter.exe WSwriter.go
.\WSwriter.exe
```

**方式2：直接运行**
```bash
cd d:\WangScape
go run WSwriter.go
```

WSwriter 启动后会：
1. 自动启动Hugo服务器进行网站预览
2. 在 8080 端口启动API服务
3. 检测Hugo地址并保存供使用

**预期输出**：
```
[STARTUP] Starting Hugo server...
[HUGO] Web Server is available at http://localhost:14746/WangScape/
WangScape Writer Online: http://127.0.0.1:8080
```

---

## 🚀 核心功能

### 1. 博客管理 API

| 端点 | 方法 | 功能 |
|------|------|------|
| `/api/posts` | GET | 获取文章列表 |
| `/api/get_content` | GET | 获取文章内容 |
| `/api/save_content` | POST | 保存/更新文章 |
| `/api/delete_post` | POST | 删除文章 |

**示例**：获取所有文章
```bash
curl http://127.0.0.1:8080/api/posts
```

### 2. 评论管理

| 端点 | 方法 | 功能 |
|------|------|------|
| `/api/comments` | GET | 获取已绝准评论 |
| `/api/add_comment` | POST | 提交新评论 |
| `/api/approve_comment` | POST | 批准评论 |
| `/api/delete_comment` | POST | 删除评论 |

**示例**：获取评论
```bash
curl http://127.0.0.1:8080/api/comments?postURL=/path/
```

### 3. Hugo网站预览

WSwriter 会自动启动 Hugo 并检测其地址。关键功能：

- ✅ 自动启动 `hugo server -D`
- ✅ 解析实际地址（可能不是1313）
- ✅ 通过API提供给其他工具

---

## 🔗 使用Hugo预览

### 方式1：直接浏览器（最简单）

查看启动日志，找到这行：
```
[HUGO] Web Server is available at http://localhost:14746/WangScape/
```

复制地址到浏览器打开即可。

### 方式2：查询API获取地址

```powershell
# PowerShell 查询Hugo地址
$response = Invoke-RestMethod http://127.0.0.1:8080/api/hugo-url
Write-Host $response.url

# 输出示例：
# http://localhost:14746/WangScape/
```

### 方式3：从文件读取

WSwriter 启动时会将Hugo地址写入：
```
.vscode/hugo-url.txt
```

你可以读取此文件获得地址。

### 方式4：VS Code Live Preview

1. 打开任何 `.html` 文件
2. 右键 → "Show Preview" 或点击Live Preview图标
3. 注意：Live Preview 可能显示单个文件而非完整网站
4. 建议使用方式1或2获得完整网站预览

---

## 🔑 必需配置

### 1. JWT_SECRET（认证密钥）

**必需**（应用启动时检查）

生成方法：
```bash
# 生成64个十六进制字符（32字节）
openssl rand -hex 32
```

**配置方式**：
```bash
# Windows PowerShell
$env:JWT_SECRET = "your-64-hex-characters"

# Linux/macOS
export JWT_SECRET="your-64-hex-characters"

# 或在 .env 文件中
echo "JWT_SECRET=your-64-hex-characters" > .env
```

### 2. SMTP配置（邮件通知，可选）

编辑 `config/comment_settings.json`：
```json
{
  "smtp_host": "smtp.gmail.com",
  "smtp_port": 587,
  "smtp_user": "your-email@gmail.com",
  "smtp_password": "encrypted-password-or-env-var",
  "smtp_from": "noreply@example.com",
  "smtp_to": ["admin@example.com"]
}
```

**密码加密**：如果指定SMTP_ENCRYPTION_KEY环境变量，密码必须加密：
```bash
$env:SMTP_ENCRYPTION_KEY = "your-64-hex-characters"
```

### 3. GitHub Token（可选，用于读取评论）

如果启用了GitHub Issues评论同步，需要设置：
```bash
$env:GITHUB_TOKEN = "your-github-token"
```

---

## 🛠️ 管理员功能

### 登录

**端点**：`POST /api/login`

```bash
curl -X POST http://127.0.0.1:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-password"
  }'
```

**返回**：
```json
{
  "token": "eyJhbGc...",
  "refresh_token": "refresh_token_value"
}
```

### 认证请求

所有管理员API需要在请求头中包含令牌：
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://127.0.0.1:8080/api/all_comments
```

### 主要管理API

| 端点 | 功能 |
|------|------|
| `/api/pending_comments` | 获取待审核评论 |
| `/api/all_comments` | 获取所有评论 |
| `/api/approve_comment` | 批准评论 |
| `/api/delete_comment` | 删除评论 |
| `/api/comment_settings` | 获取评论配置 |
| `/api/save_comment_settings` | 更新评论配置 |

---

## 🧪 测试邮件

发送测试邮件验证SMTP配置：
```bash
curl -X POST http://127.0.0.1:8080/api/test_mail \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"recipient":"test@example.com"}'
```

---

## 📊 监控和日志

### 日志文件

- `config/audit.log` - 审计日志（所有操作记录）

### 关键日志标记

| 标记 | 含义 |
|------|------|
| `[HUGO]` | Hugo服务器日志 |
| `[HUGO-URL]` | 检测到的Hugo地址 |
| `[EMAIL-WORKER]` | 邮件处理逻辑 |
| `[CACHE-HIT]` | 缓存命中（翻译/文章列表） |
| `[GITHUB-API]` | GitHub API调用 |

查看运行日志：
```bash
# Linux/macOS
tail -f config/audit.log

# Windows PowerShell
Get-Content config/audit.log -Wait
```

---

## ⚙️ 高级配置

### 环境变量完整列表

```bash
# === 认证 ===
JWT_SECRET              # 必需：JWT签名密钥（64 hex字符）
ADMIN_USERNAME          # 管理员用户名（默认: admin）
ADMIN_PASSWORD          # 管理员密码（推荐用bcrypt哈希）

# === SMTP邮件 ===
SMTP_PASSWORD           # SMTP密码或加密值
SMTP_ENCRYPTION_KEY     # AES-256加密密钥（如使用加密密码）

# === GitHub ===  
GITHUB_TOKEN            # GitHub Personal Access Token

# === HTTP服务 ===
HTTP_HOST               # 绑定地址（默认: 127.0.0.1）
HTTP_PORT               # 端口（默认: 8080）
HTTPS_HOST              # HTTPS地址
HTTPS_PORT              # HTTPS端口（默认: 443）
TLS_CERT_FILE           # TLS证书路径
TLS_KEY_FILE            # TLS密钥路径

# === 其他 ===
BASE_URL                # 网站基础URL（默认: http://127.0.0.1:8080）
BEHIND_PROXY            # 是否在代理后面（true/false）
```

### 编辑.env文件

在项目根目录创建`.env`文件自动加载：
```bash
# .env 示例
JWT_SECRET=abc123...
GITHUB_TOKEN=ghp_...
ADMIN_USERNAME=myadmin
ADMIN_PASSWORD_HASH=hash...
```

WSwriter 启动时会自动读取此文件。

---

## 📈 性能优化

WSwriter 已启用以下优化：

### 缓存系统
- **翻译缓存**：相同文本翻译结果缓存
- **文章列表缓存**：1分钟内重复请求直接返回
- **效果**：首次请求 200ms，缓存命中 <1ms

### 邮件队列
- **异步发送**：邮件投递不阻塞主线程
- **自动重试**：失败自动重试3次（指数退避：1s→2s→4s）
- **队列容量**：支持100个并发邮件任务

### GitHub API监控
- **速率限制检查**：自动检测API限制
- **预警机制**：剩余请求 <10 时告警
- **日志输出**：实时显示API使用情况

---

## ❌ 故障排查

### 问题1：Hugo无法启动

**症状**：没有看到 `[HUGO] Web Server is available at` 日志

**解决方案**：
```powershell
# 检查1313端口是否被占用
netstat -ano | findstr "1313"

# 如果有进程占用，杀掉它
Stop-Process -Id <PID> -Force

# 重新启动WSwriter
go run WSwriter.go
```

### 问题2：邮件无法发送

**检查SMTP配置**：
```bash
# 1. 查看config/comment_settings.json中的SMTP设置
cat config/comment_settings.json

# 2. 发送测试邮件
curl -X POST http://127.0.0.1:8080/api/test_mail \
  -H "Authorization: Bearer TOKEN"

# 3. 查看日志
tail -f config/audit.log | grep -i "email\|mail\|smtp"
```

### 问题3：认证失败

**确保JWT_SECRET已设置**：
```bash
# 检查环境变量
$env:JWT_SECRET  # PowerShell
echo $JWT_SECRET # Bash

# 如果为空，重新设置
$env:JWT_SECRET = (openssl rand -hex 32)
go run WSwriter.go
```

### 问题4：Hugo预览地址错误

**手动获取地址**：
```bash
# 方法1：查看日志
go run WSwriter.go 2>&1 | grep "Web Server is available"

# 方法2：查询API
Invoke-RestMethod http://127.0.0.1:8080/api/hugo-url | Select -ExpandProperty url

# 方法3：读取文件
Get-Content .vscode/hugo-url.txt
```

---

## 📚 开发说明

### 代码结构

```
WSwriter.go
├── 包和导入 (1-70行)
├── 数据结构 (70-250行)
├── 邮件系统 (250-550行)  ← P1优化：重试队列
├── 认证系统 (550-850行)  ← P0优化：强制密钥
├── API处理函数 (850-3000行)
├── 缓存层 (1250-1500行) ← P2优化：翻译/文章列表
├── Hugo启动 (4100-4180行) ← 新增
└── main() 函数 (4200行+)
```

### 修改Hugo启动命令

编辑 `WSwriter.go` 的 `startHugoServer()` 函数，修改这一行：
```go
cmd := exec.Command(hugoPathExec, "server", "-D")
```

例如，添加其他Hugo参数：
```go
cmd := exec.Command(hugoPathExec, "server", "-D", "--disableFastRender", "--renderToMemory")
```

### 构建优化二进制

```bash
# 普通构建
go build -o WSwriter.exe WSwriter.go

# 优化构建（减小体积）
go build -ldflags "-s -w" -o WSwriter.exe WSwriter.go

# 交叉编译到Linux/macOS
GOOS=linux GOARCH=amd64 go build -o WSwriter_linux WSwriter.go
GOOS=darwin GOARCH=amd64 go build -o WSwriter_mac WSwriter.go
```

---

## 📝 更新日志

### 最近改进（P0-P2优化）

**P0 - 安全关键**
- ✅ JWT密钥强制环境变量配置
- ✅ SMTP密码强制加密，拒绝明文存储

**P1 - 可靠性**
- ✅ 邮件异步队列 + 3次自动重试
- ✅ GitHub API速率限制监控
- ✅ Hugo自动启动 + 地址检测

**P2 - 性能**
- ✅ 翻译内容缓存（减少API调用）
- ✅ 文章列表缓存（1分钟TTL）

详见 [OPTIMIZATION_SUMMARY.md](OPTIMIZATION_SUMMARY.md)

---

## 📞 支持

- 查看源码注释了解具体实现
- 检查 `config/audit.log` 诊断问题
- 查看日志中的 `[ERROR]` 和 `[WARN]` 标记

---

**最后更新**：2026年3月4日  
**WSwriter版本**：v1.0 with P0-P2 Optimizations
