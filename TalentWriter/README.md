# TalentWriter

TalentWriter is the local management tool for Vantalens. The normal Windows workflow uses one unified launcher, `web.exe`, which provides both the control page and the writer page. Split services are still available through `go run` for debugging, but they should not be built as separate exe files for daily use.

## Build

```bash
go build -o web.exe ./cmd/server
```

## Run

```bash
HUGO_PATH=/path/to/hugo ADMIN_TOKEN=your-token ./web.exe
```

`web.exe` provides both the control and writer views.

## Optional Debug Mode

If you need to troubleshoot one service at a time, run them separately:

```bash
go run ./cmd/control
go run ./cmd/writer
```

Environment variables:

- `CONTROL_PORT` for the control backend port
- `WRITER_PORT` for the writer backend port
- `ADMIN_TOKEN` or `ADMIN_PASSWORD` for admin authentication

## Main API Groups

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

## Notes

- The launcher reads Hugo content from the configured `HUGO_PATH`.
- Comments and settings are stored inside the Hugo site tree.
- Control and writer pages share the same authentication token namespace in the browser.
