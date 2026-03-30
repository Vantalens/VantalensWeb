# WangScape (Hugo Blog + WSwriter)

[![Hugo](https://img.shields.io/badge/Hugo-Extended-blueviolet?style=flat-square)](https://gohugo.io/)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

WangScape is a bilingual Hugo blog project with a local management tool, WSwriter (Go).

- Hosting: GitHub Pages
- Analytics: Busuanzi
- Comments: GitHub Issues with moderation-first workflow

## Key Features

- Bilingual content management (ZH/EN)
- Local visual editing and publishing workflow
- Comment moderation, bulk actions, and export
- Admin-only analytics dashboard including visitor IP stats

## Quick Start

### 1) Run WSwriter

Windows:

```bash
WSwriter.exe
```

Or run from source:

```bash
go run WSwriter.go
```

Open http://127.0.0.1:8080.

### 2) Preview Hugo site locally

```bash
hugo server
```

Open http://localhost:1313/WangScape/.

### 3) Build the executable

```bash
go build -o WSwriter.exe WSwriter.go
```

## Login and Authorization

WSwriter uses local backend auth with JWT:

- Admin credentials are configured via .env (ADMIN_USERNAME / ADMIN_PASSWORD)
- JWT secret is configured via JWT_SECRET
- Sensitive APIs (comments, stats, settings) require authentication

## Comment Workflow (GitHub Issues)

Default flow: submit -> issue (comment + pending) -> moderation -> approved -> visible on site.

Configuration files:

- [config/_default/params.toml](config/_default/params.toml)
- [config/comment_settings.json](config/comment_settings.json)

## Analytics

- Public site analytics: Busuanzi script
- Admin dashboard analytics: aggregated by WSwriter (includes visitor IP)

See [BUSUANZI_SETUP.md](BUSUANZI_SETUP.md) for details.

## Project Structure

```text
content/               # Bilingual blog content
assets/                # Frontend assets (JS/SCSS)
config/                # Hugo and comment config
layouts/               # Template overrides
static/                # Static files
WSwriter.go            # WSwriter source code
WSwriter.exe           # Windows executable
```

## Deployment

Recommended deployment:

1. Build static site with Hugo
2. Push to GitHub repository
3. Serve with GitHub Pages

## License

MIT License. See [LICENSE](LICENSE).
