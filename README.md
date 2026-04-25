# Vantalens（Hugo Blog + TalentWriter）

[![Hugo](https://img.shields.io/badge/Hugo-Extended-blueviolet?style=flat-square)](https://gohugo.io/)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

Vantalens 是一个基于 Hugo 的双语博客项目，配套本地管理工具 TalentWriter（Go）。当前日常使用只需要启动一个统一入口：`web.exe`。它同时提供总控页面和写作页面；如果需要排查问题，再单独运行 control 和 writer。

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

### 3. 可选调试

如果需要单独调试某个后台，可以分别运行：

```bash
go run ./cmd/control
go run ./cmd/writer
```

## 主要能力

- 双语内容管理（中文/英文）
- 本地可视化编辑与发布流程
- 评论审核、批量处理与导出
- 访问统计与访客 IP 统计

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
