# Vantalens（Hugo Blog + TalentWriter）

[![Hugo](https://img.shields.io/badge/Hugo-Extended-blueviolet?style=flat-square)](https://gohugo.io/)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

Vantalens 是一个基于 Hugo 的中文博客项目，配套本地管理工具 TalentWriter（Go）。当前后端已整合为一个统一入口：`web.exe`。它在同一个进程内提供总控页面、写作页面、访问统计、评论管理和数据库同步能力。

## 快速开始

### 1. 预览站点

在仓库根目录运行：

```bash
hugo server
```

打开 http://localhost:1313/VantalensWeb/ 预览。

### 2. 运行统一入口

进入后端目录：

```bash
cd TalentWriter
```

构建并运行统一入口：

```bash
go build -o web.exe ./cmd/server
./web.exe
```

Windows 独立模式示例：

```powershell
$env:TALENTWRITER_APP_MODE="standalone"
$env:TALENTWRITER_AUTOSTART_HUGO="false"
./web.exe
```

`web.exe` 已包含总控和写作两个页面。

### 3. 后端数据库实时同步

后端启动时会从服务器 `wj` 拉取评论、访问统计、文章三类 SQLite 数据库到本地 `.talentwriter/`，之后默认每 5 分钟同步一次。同步只读取服务器文件，不会写入生产数据库。

可通过 `.env` 覆盖：

```env
DB_SYNC_ENABLED=true
DB_SYNC_REMOTE_HOST=wj
DB_SYNC_REMOTE_BASE=/var/lib/vantalens
DB_SYNC_SCP_BIN=scp
DB_SYNC_INTERVAL=5m
DB_SYNC_TIMEOUT=30s
```

登录后可调用 `/api/sync/status` 查看同步状态，或 `POST /api/sync/run` 手动触发一次同步。

### 4. 可选调试

后端调试也统一使用同一个入口：

```bash
go run ./cmd/server
```

## 主要能力

- 中文内容管理
- 本地可视化编辑与发布流程
- 评论审核、批量处理与导出
- 访问统计与访客 IP 统计

## 说明

当前站点仅保留中文界面与中文内容目录，不再提供中英文切换功能，也不维护英文站点内容。

## 部署

1. 使用 Hugo 构建静态站点
2. 推送到 GitHub 仓库
3. 由 GitHub Pages 托管

## 参考

- 项目落地标准：[PROJECT_STANDARD.md](PROJECT_STANDARD.md)
- 评论配置：[config/_default/params.toml](config/_default/params.toml)
- 评论设置：[config/comment_settings.json](config/comment_settings.json)
- 统计说明：[BUSUANZI_SETUP.md](BUSUANZI_SETUP.md)

## 许可证

MIT License，详见 [LICENSE](LICENSE)。
