# Vantalens 项目落地标准

版本：v1.0.0  
状态：生效  
最后更新：2026-04-24

本文件按 `StandardDucument/个人开发者规范体系/16_图形界面优先与本地全栈项目落地规范_个人版.md` 落地到当前项目。

## 1. 日常入口

普通使用只打开一个程序：

```powershell
D:\Vantalens\TalentWriter\web.exe
```

打开后使用浏览器访问：

- 写作后台：`http://127.0.0.1:9090/platform/backend`
- 总控平台：`http://127.0.0.1:9090/platform/control`
- 健康检查：`http://127.0.0.1:9090/health`

`control.exe`、`writer.exe`、`server.exe`、`wswriter.exe` 不作为日常入口保留。分离调试只使用：

```powershell
go run ./cmd/control
go run ./cmd/writer
```

## 2. 项目边界

- Hugo 前端根目录：`D:\Vantalens`
- Go 后端根目录：`D:\Vantalens\TalentWriter`
- 统一后端入口：`TalentWriter/cmd/server`
- 控制台调试入口：`TalentWriter/cmd/control`
- 写作端调试入口：`TalentWriter/cmd/writer`

## 3. 数据库与数据边界

后端运行时数据库位于 `.talentwriter/`，不得上传 Git：

- 访问统计：`.talentwriter/analytics/visits.db`
- 评论数据：`.talentwriter/comments/comments.db`
- 文章数据：`.talentwriter/articles/articles.db`

文章当前采用“双写”策略：

- 后端启动时从 Hugo `content/` 同步到 `articles.db`。
- 后台保存、新建、删除文章时同步写 Hugo 文件和 `articles.db`。
- Hugo 文件仍是静态站点构建来源，数据库是后端管理与查询来源。

评论当前采用数据库优先策略：

- 前端评论提交到 `/api/comments/add`。
- 公开页面只显示已审核评论。
- 审核、删除走后台接口。

## 4. Git 上传边界

不得上传：

- `.env`、`.env.*`
- `*.exe`
- `.talentwriter/`
- `.go-cache/`、`.go-mod-cache/`
- `public/`、`resources/`
- `build/`、`dist/`、`out/`
- `themes/stack/`
- 日志、临时文件、数据库文件

允许上传：

- Hugo 内容、模板、配置源码
- Go 源码、`go.mod`、`go.sum`
- 项目文档与规范文件

## 5. 安全基线

后端必须保持：

- 管理接口使用 JWT access token 鉴权。
- 登录、评论、邮箱验证码接口具有限流。
- 写接口使用请求体大小限制和服务端校验。
- HTTP 服务配置读写超时、安全响应头和 CORS 白名单。
- 评论提交包含验证码、邮箱验证、指纹与基础异常流量检测。
- 敏感字段不在公开评论接口返回。

## 6. 验证门禁

每次重构、后端改动、评论/文章/数据库改动后至少执行：

```powershell
cd D:\Vantalens\TalentWriter
$env:GOCACHE = (Join-Path (Get-Location) '.go-cache')
go build ./...
go build -o web.exe ./cmd/server
```

每次前端模板、样式、Hugo 配置改动后至少执行：

```powershell
cd D:\Vantalens
.\hugo.exe --minify
```

每次声明“可用”前必须真实启动 `web.exe` 并验证：

- `/health` 返回 `ok`
- `/api/posts` 能返回文章
- `/api/comments/challenge` 能返回验证码
- 三个数据库文件存在

## 7. 图形界面优先说明

面向日常使用者的最终说明必须回答：

- 打开什么程序。
- 浏览器访问哪个页面。
- 哪些操作在图形界面完成。
- 哪些命令仅供维护者验证使用。

## 8. 变更记录

- v1.0.0：新增 Vantalens 项目落地标准。
