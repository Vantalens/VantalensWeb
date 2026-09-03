# TalentWriter

TalentWriter 是 Vantalens 的本地管理工具。Windows 下只使用一个统一入口 `web.exe`，它在同一个后端进程内提供总控页、写作页、访问统计、评论管理和数据库同步能力。

## 构建

```bash
go build -o web.exe ./cmd/server
```

生产构建必须注入可追溯信息（PowerShell）：

```powershell
$sha = git rev-parse HEAD
$builtAt = (Get-Date).ToUniversalTime().ToString('o')
$dirty = if (git status --porcelain) { 'true' } else { 'false' }
go build -trimpath -ldflags "-X main.GitSHA=$sha -X main.BuildTime=$builtAt -X main.BuildDirty=$dirty" -o web.exe ./cmd/server
```

`GET /health` 会返回 `version`、`git_sha`、`build_time` 和 `dirty`，用于核对实际部署版本。

## 运行

```bash
HUGO_PATH=/path/to/hugo ADMIN_TOKEN=your-token ./web.exe
```

`web.exe` 同时提供总控页和写作页。

## 调试模式

调试时也使用同一个后端入口：

```bash
go run ./cmd/server
```

环境变量：

- `HTTP_PORT`：统一后端端口
- `ADMIN_TOKEN` 或 `ADMIN_PASSWORD`：管理员认证

## 主要接口分组

- `/api/login`
- `/api/posts`
- `/api/get_content`
- `/api/save_content`
- `/api/delete_post`
- `/api/create_post`
- `/api/comments`
- `/api/settings`
- `/api/control/status`
- `/api/control/command`
- `/platform`
- `/platform/posts`
- `/platform/comments`
- `/platform/analytics`

## 说明

- 启动器会从配置的 `HUGO_PATH` 读取 Hugo 内容。
- 评论和设置保存在 Hugo 站点目录中。
- 后台入口统一为 `/platform`，旧 `/platform/backend` 不再保留。

## 服务器目录约束

生产环境的 `HUGO_PATH` 必须指向由 TalentWriter 用户可写的独立站点工作树，不能直接指向只读的部署源码目录。`PUBLISH_OUTPUT_PATH` 必须是独立构建输出目录。systemd 模板只为评论库、文章库、统计库、站点工作树和构建输出开放写权限；静态站点发布到 nginx 根目录应由单独、可回滚的部署步骤完成。
