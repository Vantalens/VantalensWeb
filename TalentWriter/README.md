# TalentWriter

TalentWriter 是 Vantalens 的本地管理工具。Windows 下的日常流程只使用一个统一入口 `web.exe`，它同时提供总控页和写作页。调试时仍然可以通过 `go run` 分别启动两个服务，但日常使用不再建议拆成独立 exe 文件。

## 构建

```bash
go build -o web.exe ./cmd/server
```

## 运行

```bash
HUGO_PATH=/path/to/hugo ADMIN_TOKEN=your-token ./web.exe
```

`web.exe` 同时提供总控页和写作页。

## 可选调试模式

如果需要单独排查某个服务，可以分别运行：

```bash
go run ./cmd/control
go run ./cmd/writer
```

环境变量：

- `CONTROL_PORT`：总控后端端口
- `WRITER_PORT`：写作后端端口
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
- `/platform/control`
- `/platform/backend`

## 说明

- 启动器会从配置的 `HUGO_PATH` 读取 Hugo 内容。
- 评论和设置保存在 Hugo 站点目录中。
- 总控页和写作页在浏览器里共享同一套认证令牌命名空间。
