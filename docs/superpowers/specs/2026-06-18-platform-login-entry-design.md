# Platform 登录入口修复设计

## 目标

将 `/platform` 设为唯一后台入口。管理员在该页面登录成功后直接进入文章管理页 `/platform/posts`，并移除旧入口 `/platform/backend`。

## 路由设计

- 保留 `/platform`，作为后台登录与总览入口。
- 保留 `/platform/posts`、`/platform/comments`、`/platform/analytics` 等后台业务页面。
- 删除 `/platform/backend` 的路由注册及对应处理器；该路径之后返回 404。
- 将应用启动时自动打开的地址从 `/platform/backend` 改为 `/platform`。
- 从 API 路由说明和项目文档中移除 `/platform/backend`。

## 登录流程

1. 用户打开 `/platform`。
2. 页面调用 `/api/login` 提交用户名和密码。
3. 登录失败时保留当前页面并展示后端返回的错误。
4. 登录成功时保存 access token，然后使用 `window.location.assign('/platform/posts')` 进入文章管理页。
5. `/platform/posts` 使用已保存的 token 加载后台数据。

## 兼容性与错误处理

- 不保留旧入口重定向，避免继续暴露废弃路径。
- 登录请求异常或响应缺少 token 时不跳转。
- 已登录用户直接访问任一现有 `/platform/*` 页面时行为不变。

## 测试

- 页面生成测试验证登录成功路径包含到 `/platform/posts` 的跳转。
- 路由测试验证 `/platform` 可访问。
- 路由测试验证 `/platform/backend` 返回 404。
- 运行 `go test ./...`，确保现有后端测试无回归。

## 非目标

- 不改变 JWT、Basic Auth 或 Nginx 鉴权配置。
- 不调整后台页面视觉设计。
- 不重构其他后台 API 或数据同步逻辑。
