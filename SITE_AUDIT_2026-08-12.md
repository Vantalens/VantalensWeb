# Vantalens 前后台全面故障审计报告

审计日期：2026-08-12（Asia/Shanghai）  
审计对象：`https://vantalens.com`、服务器主机 `wj`、本地工作树 `D:\Projects\Vantalens`  
审计方式：生产环境只读检查、真实浏览器多视口检查、本地构建与测试、隔离临时目录文章生命周期测试  
结论：**线上基础访问正常，但后台文章链路当前不可作为可靠生产工具；本地待部署源码也处于不可构建状态。应先修复 P0 发布门禁，再处理数据损坏风险、地图与响应式问题。**

## 1. 范围、边界与证据等级

- 未在生产环境创建、保存、删除、恢复或发布文章。
- 未修改或重启 nginx、TalentWriter、Hugo、systemd。
- 未输出或保存密码、JWT、访客 IP、邮箱等敏感信息。
- 生产后台使用现有凭据建立了只读会话，仅调用登录和 GET 接口。
- 浏览器访问公开网页时，网站自身的 `/api/analytics/collect` 会自动记录普通页面访问；因此统计库增加了本次自动化浏览的页面访问记录，但没有文章、评论或配置写入。
- 隔离文章测试使用 Go `t.TempDir()`、临时 SQLite 和临时回收站；测试文件已删除，工作树没有遗留审计临时文件。

证据等级：

- **生产确认**：在服务器当前运行环境或线上浏览器中直接复现。
- **源码确认**：由当前工作树代码、构建器或隔离测试直接复现。
- **推断**：证据支持，但因禁止生产写操作未触发对应生产动作。

## 2. 当前健康基线

| 项目 | 结果 |
|---|---|
| 公网首页 | `200 OK`，约 0.06–0.08 秒 |
| 关键公开页面 | 首页、归档、搜索、友链、文章页均返回 `200` |
| TalentWriter 健康检查 | `200 OK`，`{"status":"ok","version":"2.0.0"}` |
| nginx | `active`，`nginx -t` 通过 |
| TalentWriter | `active`，自 2026-06-11 运行 |
| 服务器 | 连续运行 119 天；负载约 `0.04/0.05/0.01` |
| 监听边界 | 80/443 公网；9090/1313 仅监听 `127.0.0.1` |
| 磁盘 | 40 GB，已用 13 GB（33%） |
| TLS | Let's Encrypt，有效至 2026-09-20；`certbot.timer` 已启用且 active |
| Hugo 本地构建 | `--renderToMemory --minify` 成功：34 页、5 个分页，263 ms |
| Go 全量构建/测试 | 失败，详见 P0-02 |

说明：服务进程存活和首页返回 `200` 不等于后台用户路径可用。

## 3. 发现汇总

| 编号 | 严重度 | 标题 | 证据 |
|---|---:|---|---|
| P0-01 | P0 | 生产进程内 Hugo/文章目录只读，文章写入和构建无可用路径 | 生产确认 |
| P0-02 | P0 | 当前源码引用不存在的发布处理器，后端无法构建 | 源码确认 |
| P1-01 | P1 | 生产二进制、静态站点和本地源码版本漂移且不可追溯 | 生产确认 |
| P1-02 | P1 | 后台保存文章会重建并丢弃未展示的 front matter 字段 | 源码确认 |
| P1-03 | P1 | 后台没有发送 revision，并发修改保护实际未启用 | 源码确认 |
| P1-04 | P1 | 生产 API 与当前后台目标契约不一致，新增路由在生产为 404 | 生产确认 |
| P1-05 | P1 | 重复标题的第二篇文章写入错误语言目录 | 隔离测试确认 |
| P1-06 | P1 | 地图使用手绘轮廓和有限硬编码坐标，数据会静默缺点 | 源码及浏览器确认 |
| P1-07 | P1 | 后台统计页在平板和手机宽度严重横向溢出 | 浏览器确认 |
| P1-08 | P1 | KaTeX 被 CSP 阻止，数学公式资源全部加载失败 | 浏览器确认 |
| P1-09 | P1 | 文章页在 768/390px 出现卡片与代码内容越界并被裁切 | 浏览器确认 |
| P2-01 | P2 | nginx 把不存在的路径和 robots.txt 回退为首页 200 | 生产确认 |
| P2-02 | P2 | `www` 不跳转主域，域名规范化不完整 | 生产确认 |
| P2-03 | P2 | 根路径 favicon 缺失并持续污染 nginx 错误日志 | 生产确认 |
| P2-04 | P2 | 后台删除语义、回收站能力和页面提示不一致 | 源码确认 |
| P2-05 | P2 | 样式层叠补丁规模过大，缺少浏览器回归门禁 | 源码确认 |
| P2-06 | P2 | `/health` 只有固定版本号，无法定位实际部署提交 | 生产确认 |

## 4. P0：发布与核心操作阻断

### P0-01 生产进程内 Hugo/文章目录只读

建议 Issue 标题：`[P0] TalentWriter systemd 沙箱使文章目录只读，保存/创建/删除/构建不可用`  
建议标签：`bug`、`backend`、`deployment`、`P0`

**复现/检查**

```text
systemctl show talentwriter -p ProtectSystem -p ReadOnlyPaths -p ReadWritePaths -p Environment
nsenter -t <talentwriter-pid> -m -- findmnt -T /opt/vantalens/site/content/zh-cn/post
```

**证据**

- 生效环境：`HUGO_PATH=/opt/vantalens/site`。
- `ProtectSystem=strict`。
- 唯一写路径：`ReadWritePaths=/var/lib/vantalens`。
- TalentWriter 进程挂载命名空间内，`/opt/vantalens/site/...` 所在根挂载为 `ro`。
- `/var/lib/vantalens` 为 `rw`。
- 当前文章处理器最终通过 `os.WriteFile`、原子重命名或 `os.Remove` 操作 Hugo 文件；构建命令也需要写 Hugo 输出目录。

**影响**

- 后台保存、新建、删除、恢复文章无法可靠完成。
- “构建前端”需要写输出目录，同样被权限策略阻断。
- SQLite 可能写入成功而 Hugo 文件失败，或反之，形成双写不一致风险。

**根因**

systemd 最小权限策略只开放了数据库目录，但后台设计仍要求直接修改 `/opt/vantalens/site`。

**修复建议**

先确定唯一权威写入模型，再配置最小写权限：

1. 将 Hugo 工作树放到明确的可写工作目录，例如 `/var/lib/vantalens/site-worktree`；不要直接让服务修改静态发布目录。
2. `ReadWritePaths` 仅加入该工作树、文章回收站和构建临时/输出目录。
3. 构建成功后使用原子目录切换或受控部署步骤更新 `/var/www/vantalens`。
4. 文章文件与 articles.db 的更新必须在失败时可回滚，不能只忽略其中一侧错误。

**验收标准**

- 在隔离/预发布环境完成创建、保存、删除、恢复、永久删除和构建。
- 任一文件或数据库步骤失败时，另一侧不留下半完成状态。
- 生产 systemd 仍保持 `ProtectSystem=strict`，只新增必要目录，不开放整个 `/opt` 或 `/var/www`。

### P0-02 当前源码无法构建

建议 Issue 标题：`[P0] routes.go 注册未实现的发布处理器导致 TalentWriter 无法编译`  
建议标签：`bug`、`backend`、`build`、`P0`

**复现**

```powershell
cd D:\Projects\Vantalens\TalentWriter
go test ./...
go build ./...
```

**证据**

```text
internal\server\routes.go:60:60: undefined: handlers.HandlePublish
internal\server\routes.go:61:67: undefined: handlers.HandlePublishStatus
```

位置：`TalentWriter/internal/server/routes.go:60-61`。

**影响**

- `cmd/server` 和 `internal/server` 均构建失败。
- 无法生成可部署的新后端，也无法执行完整测试集。
- 当前生产旧版无法通过重新构建得到，回滚和修复验证均受阻。

**根因**

路由已提前注册，发布状态机和处理器未同时提交；接口清单与实现失去同步。

**修复建议**

- 要么完成两个处理器、状态模型和测试，要么在功能完成前删除路由注册。
- 路由存在性测试必须与处理器编译、方法限制和鉴权测试同时落地。

**验收标准**

- `go test ./...`、`go build ./...` 均通过。
- `POST /api/publish` 和 `GET /api/publish/status` 有明确的成功、冲突、失败和鉴权行为。

## 5. P1：数据完整性、接口、地图与真实布局

### P1-01 部署版本漂移且不可追溯

建议 Issue 标题：`[P1] 生产部署物缺少 Git SHA，静态站点、后端和本地源码版本不一致`  
建议标签：`deployment`、`observability`、`P1`

**证据**

- `/opt/vantalens/site` 不是 Git 仓库。
- 生产后端二进制时间：2026-06-11 20:49:43。
- 生产静态首页时间：2026-05-12 15:36:29。
- 生产 `/health` 仅返回固定 `version=2.0.0`。
- 本地 `main` 比 `origin/main` 领先 2 个提交，且存在大量 staged、unstaged、untracked 改动。
- 生产 `/api/get_content` 只返回 `content`；当前源码目标已增加 `body/metadata/revision`。

**影响**

无法回答生产代码对应哪个提交，也不能可靠判断修复是否已部署或快速回滚。

**修复建议**

- 构建时注入 `GitSHA`、`BuildTime`、`Dirty`，在 `/health` 和启动日志中输出。
- 部署生成 manifest，记录二进制 SHA-256、静态产物 SHA、配置版本和数据库 schema 版本。
- 禁止从不可重现的脏工作树直接部署。

**验收标准**

任一线上响应都能关联到唯一 Git SHA；部署脚本可从该 SHA 重建相同版本。

### P1-02 后台保存会丢弃 front matter 字段

建议 Issue 标题：`[P1] 文章编辑器保存时重建 front matter，可能静默删除标签、描述、图片和数学配置`  
建议标签：`bug`、`article-editor`、`data-loss`、`P1`

**复现**

1. 读取带有 `tags`、`description`、`image`、`math`、`license`、`comments` 或 `hidden` 的文章。
2. 在当前后台修改正文并保存。
3. 比较保存前后的 front matter。

**证据**

- `TalentWriter/internal/handlers/platform_pages.go:170-174` 的 `rebuildContent()` 只重建 `title/date/draft/pinned/categories`。
- 页面将重建后的完整 `content` 发送给 `/api/save_content`，绕过后端的 YAML 合并分支。
- `TalentWriter/internal/handlers/page.go:465-496` 的旧后台页面也采用相同模式。
- 隔离测试确认 `mergeArticleDocument()` 本身能保留未知字段，说明丢失发生在前端调用方式，而不是 YAML 合并器。

**影响**

一次普通保存就可能改变文章展示、SEO、封面、公式、评论或授权配置，属于静默数据损坏。

**修复建议**

- 页面加载后保存 `body`、结构化 `metadata` 和 `revision`，不要重建完整 YAML 字符串。
- 后端以原 front matter AST 为基础只更新受控字段。
- 增加包含全部现有 front matter 字段的往返无损测试。

**验收标准**

编辑标题或正文后，所有未编辑字段逐字义等价保留；未知嵌套 YAML 也不能消失。

### P1-03 并发 revision 保护没有接入后台页面

建议 Issue 标题：`[P1] 后端支持 revision 冲突检测，但后台保存请求未发送 revision`  
建议标签：`bug`、`article-editor`、`concurrency`、`P1`

**证据**

- `HandleGetContent` 的目标响应包含 `revision`。
- `HandleSaveContent` 仅在请求携带 `revision` 时返回 `409 Conflict`。
- `platform_pages.go:174` 保存请求只发送 `{path, content}`。
- 生产旧接口甚至不返回 revision。

**影响**

两页同时编辑、外部 Git 修改或同步替换文章后，后保存者会静默覆盖先前修改。

**修复建议**

页面加载文章时保存 revision，每次保存必须发送；收到 409 时禁止覆盖并提供重新加载/差异查看。

**验收标准**

两个会话基于同一 revision 保存时，第二个保存稳定返回 409，原文件保持第一次保存结果。

### P1-04 生产 API 与当前目标契约不一致

建议 Issue 标题：`[P1] 生产后台 API 落后于当前页面契约，回收站和发布接口不存在`  
建议标签：`bug`、`api`、`deployment`、`P1`

**生产只读结果**

| 接口 | 生产结果 | 当前源码目标 |
|---|---|---|
| `GET /api/posts` | 200，返回 12 篇 | 已实现 |
| `GET /api/get_content` | 200，仅 `data.content` | `content/body/metadata/revision` |
| `GET /api/trash/posts` | 404 | 已注册并有处理器 |
| `POST /api/restore_post` | 未调用；生产路由版本中不存在 | 当前源码已实现 |
| `POST /api/purge_post` | 未调用；生产路由版本中不存在 | 当前源码已实现 |
| `GET /api/publish/status` | 404 | 当前源码已注册但无法编译 |
| `POST /api/publish` | 未调用 | 当前源码已注册但处理器缺失 |

**影响**

部署任一单独组件都可能造成页面调用 404、字段为空、保存策略退化或功能按钮失效。

**修复建议**

为 API 引入可测试的版本契约；前后端必须作为同一发布单元构建和部署，不能独立漂移。

**验收标准**

契约测试覆盖状态码、字段、方法和鉴权；部署前由同一二进制返回页面与 API，并通过浏览器 E2E。

### P1-05 重复标题写入错误目录

建议 Issue 标题：`[P1] 重复文章标题回退路径丢失 zh-cn 目录`  
建议标签：`bug`、`article-editor`、`content-path`、`P1`

**证据**

- 首篇路径：`content\zh-cn\post\重复标题\index.md`。
- 第二篇路径：`content\posts\重复标题-2\index.md`。
- 源码：`TalentWriter/internal/handlers/posts.go:294-302`；循环回退使用 `content/posts/...`。
- 隔离生命周期测试直接复现。

**影响**

重复标题文章可能离开站点实际内容目录，导致文章列表、语言路由、Hugo 构建和数据库路径不一致。

**修复建议**

冲突路径必须保持原目录：`content/zh-cn/post/<slug>-N/index.md`；路径生成应由单一函数负责。

**验收标准**

连续创建三个同名标题，路径均位于 `content/zh-cn/post`，且 Hugo 能生成三篇独立文章。

### P1-06 地图不是真实世界地图且大量地区无法定位

建议 Issue 标题：`[P1] 访问地图使用手绘多边形和硬编码坐标，地区点位缺失或偏差`  
建议标签：`bug`、`analytics`、`map`、`P1`

**证据**

- `TalentWriter/internal/handlers/analytics.go:391-435` 只硬编码少量城市、地区和国家坐标。
- `resolveMapPoint()` 在字典未命中时直接返回 `null`（436-437）。
- `renderWorldBase()`（450 起）用 9 个粗略 SVG path 模拟大陆，不是标准地理数据。
- 生产统计返回 43 个地区分组；浏览器渲染仅生成 24 个地图标记。
- 页面文字称其为“标准世界地图”，与实际实现不符。

**影响**

地区数据会静默从地图消失；中文名称、不同供应商命名和未列入字典的城市无法定位，底图和点位也缺乏可信度。

**修复建议**

- 使用本地打包、版本固定的 Natural Earth/GeoJSON 世界地图。
- 地理查询结果保存经纬度和标准 ISO 国家代码，不在浏览器维护城市字典。
- 未定位数据进入明确列表和计数，不得静默省略。
- 同坐标或近邻点进行聚合，提供键盘可访问的详情。

**验收标准**

43 个地区均被计入“已定位或未定位”，总数守恒；中英文地区名、未知地区和坐标边界均有测试。

### P1-07 后台统计页移动端严重横向溢出

建议 Issue 标题：`[P1] 访问统计表格在 768px/390px 将页面撑宽到 1007px/933px`  
建议标签：`bug`、`frontend`、`responsive`、`analytics`、`P1`

**浏览器结果**

| 视口 | 页面 scrollWidth | 结果 |
|---:|---:|---|
| 1440 | 1440 | 正常 |
| 1024 | 1024 | 正常 |
| 768 | 1007 | 横向溢出 239px |
| 390 | 933 | 横向溢出 543px |

**证据**

- `analytics.go:182-206` 的表格和 `.mono` 长路径缺少断行或局部滚动容器。
- 响应式规则（210-212）只改变 grid 列数，没有处理表格最小内容宽度。
- 溢出元素集中在“分页访问统计”的 table、长路径单元格和父 panel。

**修复建议**

表格放入 `overflow-x:auto` 的局部容器；路径列使用可复制的截断/换行策略；移动端改为卡片或关键列视图。

**验收标准**

390、768、1024、1440 下 `documentElement.scrollWidth <= clientWidth + 2`，表格内容仍可完整查看。

### P1-08 KaTeX 被 CSP 阻止

建议 Issue 标题：`[P1] nginx CSP 与 KaTeX CDN 配置冲突，公式资源全部被浏览器阻止`  
建议标签：`bug`、`frontend`、`csp`、`math`、`P1`

**复现**

打开任一启用数学组件的文章并查看浏览器控制台和失败请求。

**证据**

浏览器在四种视口均记录：

- cdnjs KaTeX CSS、JS、auto-render 被 CSP 阻止。
- unpkg 备用 KaTeX CSS、JS 同样被 CSP 阻止。
- nginx CSP 仅允许 `style-src 'self' 'unsafe-inline'` 和 `script-src 'self' 'unsafe-inline'`。
- 备用资源定义位于 `layouts/partials/article/components/math.html:4-6`。

**影响**

数学公式不能可靠渲染，且每个文章页产生多条控制台错误和失败请求。

**修复建议**

优先将固定版本 KaTeX 文件自托管并保持 CSP `self`；不要继续增加外部 CDN 白名单和双重回退。

**验收标准**

公式正确渲染；控制台无 CSP 错误；离线/外部 CDN 不可用时仍能工作。

### P1-09 文章页内容越界并被全局隐藏规则裁切

建议 Issue 标题：`[P1] 文章代码块和相关推荐在平板/手机视口越界并被裁切`  
建议标签：`bug`、`frontend`、`responsive`、`article`、`P1`

**浏览器证据**

- 768px：相关推荐 `.article-list--tile` 宽约 838px，子卡片右边界约 814px，超过 768px 视口。
- 390px：多个代码 token 右边界达到 429–501px。
- 页面整体没有报告横向滚动，因为 `html/body` 使用 `overflow-x:hidden !important`，越界内容被裁掉而不是可滚动。

**相关源码**

- `assets/scss/custom.scss:2294-2309`：highlight 外层隐藏、内层滚动。
- `assets/scss/custom.scss:4919`、`5043`、`5489`、`5497`：多处全局横向隐藏覆盖。

**影响**

代码内容和相关推荐在常用平板、手机宽度不可完整阅读；自动化若只检查页面 scrollWidth 会误判为“没有溢出”。

**修复建议**

- 删除全局掩盖问题的横向隐藏补丁。
- 代码块由单一容器负责 `overflow-x:auto`，内部 table/code 允许按内容展开。
- 相关推荐使用可收缩 grid，断点下切换为单列。

**验收标准**

所有代码字符可通过代码块自身滚动访问；页面主体无裁切；相关推荐卡片边界不超过视口。

## 6. P2：SEO、日志、语义和维护性

### P2-01 软 404 与 robots.txt 错误响应

建议 Issue 标题：`[P2] nginx SPA 回退导致任意死链和 robots.txt 返回首页 200`  
建议标签：`bug`、`nginx`、`seo`、`P2`

**生产复现**

- `/definitely-not-a-real-page-20260812`：`200 text/html`，正文为首页。
- `/robots.txt`：`200 text/html`，正文为首页。

**根因**

`deploy/vantalens-nginx.conf:145`：

```nginx
try_files $uri $uri/ /index.html;
```

Hugo 不是 SPA，不应把所有未知路径回退到首页。

**修复建议**

使用 `try_files $uri $uri/ =404;`，部署真实 `404.html` 和 `robots.txt`；通过 `error_page 404 /404.html` 返回正确状态。

**验收标准**

死链返回 404；robots 为 `text/plain`；sitemap 和 RSS 保持 200。

### P2-02 `www` 未跳转主域

建议 Issue 标题：`[P2] www.vantalens.com 与主域同时返回 200，缺少规范化重定向`  
建议标签：`nginx`、`seo`、`P2`

**证据**

- `https://www.vantalens.com/` 最终仍为 www 且返回 200。
- nginx 同一 server block 同时声明 `www.vantalens.com vantalens.com`（配置第 2 行）。
- 页面 canonical 指向主域，只能部分缓解重复入口。

**修复建议与验收**

为 www 建独立 server block，永久 301 到 `https://vantalens.com$request_uri`；所有路径只跳转一次。

### P2-03 根路径 favicon 缺失

建议 Issue 标题：`[P2] /favicon.ico 缺失导致 nginx 错误日志持续出现无效噪声`  
建议标签：`bug`、`frontend`、`operations`、`P2`

**证据**

- HTML 使用 `/img/favicon.png`，该资源正常。
- 浏览器和扫描器仍请求 `/favicon.ico`。
- nginx error.log 当天多次记录 `open() "/var/www/vantalens/favicon.ico" failed`。

**修复建议与验收**

部署兼容的根路径 favicon 或显式重定向；正常访问后 error.log 不再新增该错误。

### P2-04 删除语义和回收站能力不一致

建议 Issue 标题：`[P2] 文章删除已改为回收站，但后台没有回收站入口且提示仍称“已删除”`  
建议标签：`bug`、`article-editor`、`ux`、`P2`

**证据**

- 当前 `HandleDeletePost` 实际调用 `trashArticle()`。
- 页面只显示“删除文章”，成功提示“文章已删除”。
- 当前源码已实现 list/restore/purge 接口，但页面没有回收站列表、恢复或永久删除操作。
- 生产相关路由仍为 404。

**影响**

用户无法判断文章是可恢复还是永久删除，也无法通过图形界面完成恢复。

**验收标准**

操作名称明确为“移入回收站”；提供回收站列表、恢复和二次确认的永久删除；生产契约一致。

### P2-05 样式补丁规模过大且缺少回归门禁

建议 Issue 标题：`[P2] custom.scss 累积 6377 行和 874 个 !important，布局修复互相覆盖`  
建议标签：`maintainability`、`frontend`、`css`、`P2`

**证据**

- `assets/scss/custom.scss`：6377 行。
- 38 个 media query。
- 874 次 `!important`。
- 同一 html/body overflow、文章宽度、侧栏宽度规则在文件后部多次硬重置。
- 当前真实问题表现为“元素越界但页面滚动宽度正常”，即隐藏规则掩盖布局错误。

**修复建议**

按页面区域收敛样式所有权；删除重复 hard reset；建立 Playwright 视口断言后再逐段清理，避免一次性重写。

**验收标准**

关键页面四种视口截图和 DOM 边界测试稳定；不再依赖全局 `overflow-x:hidden` 隐藏错误。

### P2-06 健康检查不能标识部署版本

建议 Issue 标题：`[P2] /health 固定返回 2.0.0，无法用于部署确认和回滚`  
建议标签：`observability`、`deployment`、`P2`

**修复建议与验收**

返回 `version/git_sha/build_time/dirty`；部署完成后自动校验 SHA 与预期提交一致。

## 7. 浏览器矩阵结果

### 前台

| 页面 | 1440 | 1024 | 768 | 390 | 控制台/资源 |
|---|---|---|---|---|---|
| 首页 | 无整体溢出 | 无整体溢出 | 无整体溢出 | 无整体溢出 | 无错误 |
| 归档 | 无整体溢出 | 无整体溢出 | 无整体溢出 | 无整体溢出 | 无错误 |
| 搜索 | 可输入并显示结果 | 同左 | 同左 | 同左 | 无错误 |
| 友链 | 无整体溢出 | 无整体溢出 | 无整体溢出 | 无整体溢出 | 无错误 |
| 文章 | 主体可见 | 主体可见 | 相关推荐越界 | 代码 token 越界/裁切 | KaTeX 资源被 CSP 阻止 |
| 评论入口 | 弹窗可打开 | 弹窗可打开 | 弹窗可打开 | 弹窗可打开 | 未提交真实评论 |

### 后台（生产 HTML + 脱敏生产统计数据本地渲染）

| 页面 | 1440 | 1024 | 768 | 390 |
|---|---|---|---|---|
| `/platform` | 无整体溢出 | 无整体溢出 | 无整体溢出 | 无整体溢出 |
| `/platform/posts` | 12 项可渲染 | 无整体溢出 | 无整体溢出 | 无整体溢出 |
| `/platform/comments` | 初始布局正常 | 正常 | 正常 | 正常 |
| `/platform/analytics` | 24 个标记 | 24 个标记 | scrollWidth 1007 | scrollWidth 933 |

后台写按钮没有在生产触发；“能渲染按钮”不等于对应写操作可用。

## 8. 文章隔离测试结果

测试环境：临时 Hugo 根目录、临时 `articles.db`、临时回收站；未使用真实文章。

| 场景 | 结果 |
|---|---|
| 创建草稿 | 通过 |
| 读取创建后的正文 | 通过 |
| 保存正文并更新数据库 | 通过 |
| 移入回收站 | 通过 |
| 从回收站恢复 | 通过 |
| 永久删除回收站项 | 通过 |
| YAML 合并保留未知字段 | 通过 |
| revision 随内容变化 | 通过 |
| 同标题路径 | 失败：第二篇落入 `content/posts` |
| HTTP 409 并发链路 | 未完整执行：页面不发送 revision |
| 发布/发布状态 | 阻断：处理器缺失，项目无法编译 |

底层生命周期可工作不代表生产链路可用：生产 systemd 权限与页面请求格式仍然阻断或削弱这些能力。

## 9. 目标文章 API 契约

后续修复应锁定以下契约，并由同一版本的页面和后端共同发布：

| 方法与路径 | 目标行为 |
|---|---|
| `GET /api/posts` | 返回文章摘要列表和稳定 path |
| `GET /api/get_content` | 返回 `content/body/metadata/revision` |
| `POST /api/save_content` | 接收 `path/body/metadata/revision`；保留未知 YAML；冲突返回 409 |
| `POST /api/create_post` | 创建指定语言目录下的草稿；重复 slug 稳定追加序号 |
| `POST /api/delete_post` | 明确语义为移入回收站 |
| `GET /api/trash/posts` | 返回可恢复项，不暴露服务器绝对路径 |
| `POST /api/restore_post` | 目标已存在时拒绝覆盖 |
| `POST /api/purge_post` | 二次确认后永久删除 |
| `POST /api/publish` | 有状态、幂等、失败可诊断；不得只等同于运行 Hugo |
| `GET /api/publish/status` | 返回当前版本、阶段、最后结果和错误摘要 |

所有写接口必须保持 JWT 鉴权、请求体限制、路径限制和审计日志；不能以放宽 systemd 或目录到 `0777` 作为修复。

## 10. 推荐修复顺序

1. **P0 构建门禁**：处理或撤回未实现发布路由，使全量 Go 构建恢复。
2. **P0 写入架构**：确定可写工作树、数据库事务补偿和构建发布目录，修复 systemd 最小权限。
3. **文章数据安全**：接入结构化 metadata + revision，修复重复标题路径，补齐回收站页面。
4. **部署一致性**：加入 Git SHA/manifest，确保页面和 API 同版本发布。
5. **地图和统计布局**：替换真实地理数据，修复未定位守恒和响应式表格。
6. **前台内容可读性**：自托管 KaTeX，修复代码块与相关推荐裁切。
7. **nginx/SEO**：正确 404、robots、www 301、favicon。
8. **CSS 收敛**：在回归测试保护下逐段删除重复硬覆盖。

## 11. 修复后的强制门禁

### 后端

```powershell
cd D:\Projects\Vantalens\TalentWriter
go test ./...
go build ./...
go build -o web.exe ./cmd/server
```

必须增加：

- 文章 front matter 往返无损测试。
- revision 409 冲突测试。
- 同标题路径测试。
- 回收站恢复/覆盖/永久删除测试。
- 发布状态机和失败回滚测试。
- 页面/API 契约测试。

### Hugo 与前台

```powershell
cd D:\Projects\Vantalens
.\hugo.exe --renderToMemory --minify
```

Playwright 覆盖 1440/1024/768/390：

- 首页、归档、搜索、友链、文章、评论弹窗。
- 页面主体无非预期横向溢出。
- 代码块自身可滚动且不裁切。
- 相关推荐不越界。
- 数学公式无 CSP/资源错误。
- 后台统计表格和地图可用。

### 部署后只读验收

- `/health` SHA 与预期提交一致。
- 不存在路径返回 404。
- `/robots.txt` 为 `text/plain`。
- www 单次 301 到主域。
- nginx/TalentWriter 日志无新增错误。
- systemd 仅开放必要写路径。
- 生产写操作必须另行取得明确授权后，才使用专门测试草稿执行一次完整 canary。

## 12. 残余风险与未执行项

- 未在生产触发文章保存、删除、恢复、发布或构建；对应生产错误正文和数据库补偿行为未直接观测。
- 未提交真实评论或邮件验证码，避免影响用户和外部邮件系统。
- 未对真实评论内容做浏览器渲染，后台评论布局使用空/脱敏数据验证。
- 未进行高并发、压力、数据库故障注入或磁盘满测试。
- 未验证 TLS 自动续期的实际签发，只确认 timer 已启用并在调度。
- 当前脏工作树包含大量用户改动；后续修复必须逐项确认 staged/unstaged 边界，禁止整体重置或覆盖。


## 13. 2026-08-12 本地修复实施结果

本节记录审计后的本地修复，不代表生产环境已经更新。本轮未部署、未重启服务、未修改生产 nginx/systemd，也未对生产文章执行写操作。

| 问题 | 本地状态 | 实施结果 |
|---|---|---|
| P0-01 | 部分完成，待部署 | systemd 模板改用 `/var/lib/vantalens/site-worktree` 和独立发布输出，只开放评论、文章、统计、工作树和发布目录写权限；服务器仍需预置工作树并走受控发布。 |
| P0-02 | 已修复 | 已实现认证发布接口和状态接口；全量 `go test ./...`、`go build ./...` 通过。 |
| P1-01 | 部分完成，待部署 | 构建支持注入 Git SHA、构建时间和 dirty 状态，`/health` 暴露这些字段；当前生产版本仍未更新。 |
| P1-02/P1-03/P1-05 | 已修复 | 结构化保存保留未知 front matter，revision 冲突返回 409，同标题文章始终写入 `content/zh-cn/post`；新增隔离全流程测试。 |
| P1-04 | 本地已修复，待部署 | 本地目标文章、回收站和发布路由均存在；生产旧 API 在部署前仍保持原状。 |
| P1-06 | 已修复 | 底图改为本地 Natural Earth 世界地图；地理缓存保存经纬度并兼容旧库迁移；中文国家名归一化；移除 24 个标记上限，未知分组仍显式计数。 |
| P1-07 | 已修复 | 后台表格采用容器内横向滚动，文字允许换行；浏览器实测 1440/1024/768/390 均无页面级横向溢出。 |
| P1-08 | 已修复 | KaTeX 0.17.0 CSS、JS、字体改为同源自托管；数学文章实测生成 104 个 KaTeX 节点，无加载错误。 |
| P1-09 | 已修复 | 文章内容、代码高亮和相关推荐增加宽度收敛规则；四档浏览器实测无页面级溢出，代码块局部滚动。 |
| P2-01/P2-02/P2-03 | 配置已修复，待部署 | 新增真实 404 页面、`robots.txt`、根 favicon；nginx 模板使用 `=404`、独立 www 301 和 404 error page。 |
| P2-04 | 已修复 | 删除操作明确改为移入回收站，并提供恢复和永久删除接口/页面操作；文件与数据库失败路径加入补偿。 |
| P2-05 | 部分完成 | 已修复本次确认的真实越界规则并做四档浏览器回归；6000 余行历史样式尚未系统重构，也尚未加入 CI 浏览器门禁。 |
| P2-06 | 已修复，待部署 | `/health` 已返回 `version/git_sha/build_time/dirty`；生产部署必须使用带 ldflags 的构建命令。 |

本地验证结果：

- `go test ./...`：通过。
- `go build ./...`：通过。
- `hugo.exe --renderToMemory --minify`：通过，34 个页面、93 个静态文件。
- 前台 Playwright：1440/1024/768/390 下首页、列表、搜索、归档、友链及数学文章无页面级横向溢出；404、robots、favicon 状态正确。
- 后台 Playwright：1440/1024/768/390 下统计页无页面级横向溢出，表格局部滚动，地图使用本地 SVG。
- 隔离文章测试：创建、同标题路径、结构化保存、未知 front matter 保留、revision 409、草稿转发布、回收站、恢复、永久删除均通过。

仍需部署后只读验收 nginx 配置语法、www 单次跳转、生产 `/health` SHA、静态资源响应和服务写路径；生产写入 canary 仍需单独授权。