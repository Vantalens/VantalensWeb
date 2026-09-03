# Vantalens 部署手册

本文档描述 Vantalens（Hugo 静态站 + TalentWriter Go 后端）的部署流程。
配套脚本：`deploy/deploy.sh`（在本地 Windows Git Bash 中执行，通过 SSH 别名 `wj` 操作服务器）。

> 【待确认】以下服务器路径为现有知识推断值，首次部署前必须登录服务器核对：
>
> | 项目 | 推断值 | 核对方式 |
> |---|---|---|
> | 后端二进制 | `/opt/vantalens/talentwriter/talentwriter-server` | `systemctl cat talentwriter` 看 ExecStart |
> | 站点根目录 | `/var/www/vantalens` | nginx 配置中的 `root` 指令 |
> | nginx 配置文件 | `/etc/nginx/conf.d/vantalens.conf` | `nginx -T \| grep -n vantalens` |
> | drop-in 目录 | `/etc/systemd/system/talentwriter.service.d/` | `systemctl cat talentwriter` 输出头部 |

---

## 1. 架构与发布链路

```
浏览器
  │ 443 (nginx, vantalens.com)
  ├─ 静态站点      → /var/www/vantalens（Hugo 构建产物，nginx 直接服务）
  ├─ /api/*        → 127.0.0.1:9090（TalentWriter，systemd 服务 talentwriter）
  ├─ /platform     → 127.0.0.1:9090（后台，basic-auth）
  └─ /preview/     → 127.0.0.1:1313（Hugo preview，basic-auth）

发布链路（新版代码）：
  后台 /api/publish → Hugo 构建 → PUBLISH_OUTPUT_PATH=/var/lib/vantalens/publish
  → 独立同步步骤 → /var/www/vantalens
```

注意：systemd 单元把 `/var/www/vantalens` 配为 `ReadOnlyPaths`，**后端进程无法直接写站点根目录**。
「publish → /var/www/vantalens」的最后一段同步必须由部署侧（本脚本的 rsync 步骤或人工）完成。

## 2. 前置检查清单

本地（Windows Git Bash）：

- [ ] 已安装 Go（`go version`，需 ≥ 1.23）
- [ ] Hugo 可用（项目根有 `hugo.exe`，或 PATH 中有 `hugo`）
- [ ] `git`、`ssh`、`scp`、`rsync`、`curl` 可用
- [ ] `~/.ssh/config` 中别名 `wj` 指向服务器，且 `ssh wj "echo ok"` 成功
- [ ] `deploy/vantalens-nginx.conf` 为纯 LF 行尾：`grep -c $'\r' deploy/vantalens-nginx.conf` 应为 0（脚本会自动检查）
- [ ] 工作区改动已提交或已知悉（脚本会把 dirty 状态注入二进制）

服务器（`ssh wj`）：

- [ ] 当前 SSH 用户具备免密或交互式 `sudo` 权限（脚本涉及 systemctl / chown / nginx -t）
- [ ] 存在 `vantalens` 用户和组：`id vantalens`
- [ ] `systemctl cat talentwriter` 可执行，且能看到三个 drop-in：
      `20-preview.conf`、`30-authority.conf`、`override.conf`
- [ ] 数据目录：`/var/lib/vantalens/{comments,articles,analytics}` 已存在且属主为 `vantalens:vantalens`
- [ ] 核对并补齐合并后环境变量（见下表）

### 2.1 systemd 合并配置要求

服务器存在 drop-in 覆盖，仓库里的 `deploy/talentwriter.service` **不是唯一生效来源**。
每次涉及配置变更的部署，必须执行 `sudo systemctl cat talentwriter` 核对合并结果，确认包含：

| 变量 | 要求值 | 来源 |
|---|---|---|
| `HTTP_PORT` / `HTTP_HOST` | `9090` / `127.0.0.1` | 主 unit |
| `HUGO_PATH` | `/opt/vantalens/site` | drop-in 覆盖（主 unit 中的 site-worktree 值不生效） |
| `PREVIEW_PUBLIC_URL` | 由 drop-in 提供 | drop-in `20-preview.conf` |
| `COMMENT_SETTINGS_PATH` | `/var/lib/vantalens/comments/comment_settings.json` | drop-in / 主 unit |
| `PUBLISH_OUTPUT_PATH` | `/var/lib/vantalens/publish` | 主 unit |
| `AUTHORITY_BACKEND` | `true` | 主 unit / `30-authority.conf` |
| `DB_SYNC_ENABLED` | `false` | 主 unit |
| `ReadWritePaths` | 含 comments / articles / analytics / site-worktree / publish | 主 unit |
| `ReadOnlyPaths` | 含 `/var/www/vantalens` | 主 unit |

**禁止在未核对合并配置的情况下直接覆盖 drop-in 文件。** 如需改主 unit，先 `diff` 再手工更新。

## 3. 首次部署

首次部署 = 全新服务器或目录尚未就绪的情况。按顺序执行：

1. **人工准备**（脚本不做）：
   - 创建 `vantalens` 用户：`sudo useradd -r -s /usr/sbin/nologin vantalens`
   - 安装主 unit：`sudo cp deploy/talentwriter.service /etc/systemd/system/talentwriter.service`
   - 确认/建立 drop-in 目录与三个 drop-in 文件（`20-preview.conf`、`30-authority.conf`、`override.conf`）
   - 安装 nginx 配置并申请证书（Let's Encrypt 路径已写死在配置中）
   - 准备站点源码：`HUGO_PATH=/opt/vantalens/site` 需为含 `content/` 的 Hugo 站点工作树
   - 准备 `.env` 类敏感配置（管理员账号、SMTP 等）——**脚本不会也不应触碰**
2. 运行脚本：
   ```bash
   bash deploy/deploy.sh
   ```
   脚本会自动创建缺失的 `/var/lib/vantalens/site-worktree` 和 `/var/lib/vantalens/publish` 并 chown。
3. 走完第 5 节的验证清单。
4. 实测一次完整发布流程（后台触发 `/api/publish`，确认产物落到
   `/var/lib/vantalens/publish`，再确认同步到 `/var/www/vantalens` 的步骤可用）。

## 4. 增量更新（日常）

```bash
cd /d/Projects/Vantalens
bash deploy/deploy.sh                 # 全量：二进制 + 站点 + （可选）nginx
bash deploy/deploy.sh --skip-nginx    # 不动 nginx 配置（最常用）
bash deploy/deploy.sh --skip-site --skip-nginx   # 只更新后端二进制
```

脚本流程（每步有进度输出，危险步骤需输入 `y` 确认）：

1. 前置检查（命令、SSH 连通性、nginx 配置 LF 检查）
2. 交叉编译：`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath`，
   通过 ldflags 注入 `GitSHA` / `BuildTime` / `BuildDirty`（只读 git 查询）
3. 备份服务器现二进制为 `talentwriter-server.backup`
4. 停服务 → 上传 → 恢复属主与权限
5. 创建缺失目录（site-worktree / publish / comments）并 chown
6. 打印 `systemctl cat talentwriter` 合并配置，**人工确认**后继续
7. `daemon-reload` + 重启，检查 active
8. 服务器本机请求 `http://127.0.0.1:9090/api/health`，校验 `git_sha` 字段与本地构建一致
9. `hugo --minify` → rsync 到家目录 staging → `sudo rsync --delete` 进 `/var/www/vantalens`
10. 可选：上传 nginx 配置（先备份 → `nginx -t` → 确认后 `reload`）

`--yes` 可跳过全部确认，仅限流程完全跑通后的重复部署使用。

## 5. 验证清单

部署完成后逐项确认：

- [ ] `curl -s https://vantalens.com/api/health` 返回 JSON 且含 `git_sha`，
      值等于本次构建注入的 SHA（脚本第 7 步已自动核对，这里从公网再验一次）
- [ ] `git_sha` 不是 `unknown`，`dirty` 与预期一致
- [ ] `https://vantalens.com/platform` 经 basic-auth 后可访问后台页面
- [ ] 首页及若干文章页正常（静态站点同步未丢文件）
- [ ] 评论读取正常：`https://vantalens.com/api/comments?...`
- [ ] 发布流程实测：后台触发 `/api/publish` → 检查 `/var/lib/vantalens/publish` 有新产物
      → 确认同步到 `/var/www/vantalens` 后线上内容更新
- [ ] `ssh wj "sudo journalctl -u talentwriter -n 50 --no-pager"` 无异常报错，
      启动日志中 `git_sha=...` 与健康检查一致

## 6. 回滚

### 6.1 后端二进制回滚

每次部署都会把旧二进制备份为 `.backup`（只保留上一份）：

```bash
ssh wj
sudo systemctl stop talentwriter
sudo cp -a /opt/vantalens/talentwriter/talentwriter-server.backup \
           /opt/vantalens/talentwriter/talentwriter-server
sudo systemctl start talentwriter
curl -s http://127.0.0.1:9090/api/health   # 确认 git_sha 回到旧版本
```

### 6.2 nginx 配置回滚

脚本上传前会自动备份为 `vantalens.conf.backup.<时间戳>`：

```bash
ssh wj
ls -t /etc/nginx/conf.d/vantalens.conf.backup.* | head -3   # 找到目标备份
sudo cp -a /etc/nginx/conf.d/vantalens.conf.backup.<时间戳> \
           /etc/nginx/conf.d/vantalens.conf
sudo nginx -t && sudo systemctl reload nginx
```

### 6.3 静态站点回滚

脚本不提供静态站点自动回滚（`rsync --delete` 不可逆）。
如需回滚：重新从本地对应 git 版本 `hugo --minify` 后重跑
`bash deploy/deploy.sh --skip-nginx`（只重发站点部分）。

### 6.4 systemd 配置回滚

若改动了主 unit 或 drop-in：从改动前的 `systemctl cat` 输出恢复对应文件，
然后 `sudo systemctl daemon-reload && sudo systemctl restart talentwriter`。
建议改动前手工 `sudo systemctl cat talentwriter > ~/talentwriter.unit.bak` 留档。

## 7. 常见问题

- **`/api/health` 没有 `git_sha` 字段**：跑的还是旧二进制，检查上传是否成功、服务是否真的重启了。
- **服务起不来**：`journalctl -u talentwriter -n 100` 看日志；常见原因是
  `ReadWritePaths` 未覆盖新目录，或 drop-in 把 `HUGO_PATH` 指到不存在的路径。
- **发布后台报权限错误**：确认 `PUBLISH_OUTPUT_PATH=/var/lib/vantalens/publish`
  存在、属主为 `vantalens`，且在 unit 的 `ReadWritePaths` 中。
- **nginx -t 失败**：脚本已阻止 reload；从 `.backup.<时间戳>` 恢复后排查。
