# WangScape（Hugo Blog + WSwriter）

[![Hugo](https://img.shields.io/badge/Hugo-Extended-blueviolet?style=flat-square)](https://gohugo.io/)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

WangScape 是一个基于 Hugo 的双语博客项目，配套本地管理工具 WSwriter（Go）。

- 托管：GitHub Pages
- 统计：卜算子（Busuanzi）
- 评论：GitHub Issues 审核制（先审后显）

## 核心能力

- 双语内容管理（中文/英文）
- 本地可视化编辑与发布流程
- 评论审核、批量处理与导出
- 访问统计与访客 IP 统计（仅管理员可见）

## 快速开始

### 1. 运行 WSwriter

Windows：

```bash
WSwriter.exe
```

或源码运行：

```bash
go run WSwriter.go
```

打开 http://127.0.0.1:8080 进入管理界面。

### 2. 本地预览 Hugo 站点

```bash
hugo server
```

打开 http://localhost:1313/WangScape/ 预览。

### 3. 构建桌面工具

```bash
go build -o WSwriter.exe WSwriter.go
```

## 登录与权限

WSwriter 登录使用本地后端鉴权（JWT）：

- 管理员账号由 .env 中 ADMIN_USERNAME / ADMIN_PASSWORD 配置
- JWT 密钥由 JWT_SECRET 配置
- 敏感接口（评论管理、统计、设置）需要已登录

## 评论工作流（GitHub Issues）

默认流程：访客提交 -> 生成 Issue（comment + pending）-> 管理员审核 -> approved 后展示。

配置文件：

- [config/_default/params.toml](config/_default/params.toml)
- [config/comment_settings.json](config/comment_settings.json)

## 统计方案

- 前台站点统计：卜算子脚本
- 管理后台统计：WSwriter 聚合数据（含访客 IP）

详细说明见 [BUSUANZI_SETUP.md](BUSUANZI_SETUP.md)。

## 项目结构

```text
content/               # 博客内容（中英双语）
assets/                # 前端资源（JS/SCSS）
config/                # Hugo 与评论配置
layouts/               # 模板覆盖
static/                # 静态文件
WSwriter.go            # WSwriter 源码
WSwriter.exe           # Windows 可执行文件
```

## 部署

当前推荐部署方式：

1. 使用 Hugo 构建静态站点
2. 推送到 GitHub 仓库
3. 由 GitHub Pages 托管

## 许可证

MIT License，详见 [LICENSE](LICENSE)。
