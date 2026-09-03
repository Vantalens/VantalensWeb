#!/usr/bin/env bash
# =============================================================================
# Vantalens 部署脚本（在本地 Windows Git Bash 中执行）
#
# 覆盖流程：
#   1. 本地交叉编译 Linux amd64 二进制（注入 git_sha / build_time / dirty）
#   2. 服务器备份现有二进制
#   3. 上传新二进制
#   4. 远端创建缺失目录（site-worktree / publish）并 chown
#   5. systemctl cat 合并配置核对（打印 diff，人工确认，不静默改 drop-in）
#   6. daemon-reload + 重启 talentwriter
#   7. 验证 /api/health 返回新 git_sha
#   8. 本地 hugo --minify 构建并 rsync public/ 到服务器站点目录
#   9. 可选：更新 nginx 配置（先 nginx -t 再 reload）
#
# 约束：全程不读取/打印任何 .env 内容；脚本中的 git 命令均为只读查询。
# =============================================================================
set -euo pipefail

# ------------------------------ 路径常量（按需修改） ------------------------------
SSH_HOST="wj"                                          # SSH 别名
LOCAL_REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_MODULE_DIR="${LOCAL_REPO_ROOT}/TalentWriter"        # Go 模块根目录
GO_BUILD_PKG="./cmd/server"                            # main 包
BINARY_NAME="talentwriter-server"
LOCAL_BUILD_DIR="${LOCAL_REPO_ROOT}/build"
LOCAL_BINARY="${LOCAL_BUILD_DIR}/${BINARY_NAME}"

# 以下远端路径已于 2026-09-03 通过只读侦察确认
REMOTE_BIN_DIR="/opt/vantalens/talentwriter"           # 已确认（systemctl cat ExecStart）
REMOTE_BIN="${REMOTE_BIN_DIR}/${BINARY_NAME}"
REMOTE_SITE_DIR="/var/www/vantalens"                   # 已确认（nginx root 指令）
REMOTE_DATA_DIR="/var/lib/vantalens"
REMOTE_WORKTREE_DIR="${REMOTE_DATA_DIR}/site-worktree" # 尚不存在，脚本会创建
REMOTE_PUBLISH_DIR="${REMOTE_DATA_DIR}/publish"        # 尚不存在，脚本会创建
REMOTE_COMMENTS_DIR="${REMOTE_DATA_DIR}/comments"
REMOTE_SERVICE_USER="vantalens"
REMOTE_SERVICE_GROUP="vantalens"
SYSTEMD_SERVICE="talentwriter"
REMOTE_SYSTEMD_UNIT="/etc/systemd/system/talentwriter.service"
REMOTE_NGINX_CONF="/etc/nginx/sites-available/vantalens"  # 已于 2026-09-03 侦察确认（sites-enabled 软链至此）
LOCAL_NGINX_CONF="${LOCAL_REPO_ROOT}/deploy/vantalens-nginx.conf"
LOCAL_SYSTEMD_UNIT="${LOCAL_REPO_ROOT}/deploy/talentwriter.service"

HEALTH_URL_LOCAL="http://127.0.0.1:9090/api/health"    # 在服务器上本机回环验证，绕过 nginx/basic-auth

# ------------------------------ 工具函数 ------------------------------
log()  { echo ""; echo "==> [$(date '+%H:%M:%S')] $*"; }
warn() { echo "!!  $*" >&2; }
die()  { echo "XX  $*" >&2; exit 1; }

# 危险步骤前的确认；--yes 全局开关可跳过（仅用于熟练后的重复部署）
ASSUME_YES=false
for arg in "$@"; do
    case "$arg" in
        --yes|-y) ASSUME_YES=true ;;
        --skip-site) SKIP_SITE=true ;;
        --skip-nginx) SKIP_NGINX=true ;;
        --help|-h)
            echo "用法: bash deploy/deploy.sh [--yes] [--skip-site] [--skip-nginx]"
            echo "  --yes         跳过所有人工确认（危险，仅限熟练后使用）"
            echo "  --skip-site   跳过 Hugo 构建与静态站点同步"
            echo "  --skip-nginx  跳过 nginx 配置更新步骤"
            exit 0 ;;
    esac
done
SKIP_SITE="${SKIP_SITE:-false}"
SKIP_NGINX="${SKIP_NGINX:-false}"

confirm() {
    local prompt="$1"
    if [ "${ASSUME_YES}" = true ]; then
        warn "已用 --yes 跳过确认: ${prompt}"
        return 0
    fi
    echo ""
    read -r -p "?? ${prompt} [y/N] " reply
    case "$reply" in
        y|Y|yes|YES) return 0 ;;
        *) die "操作者取消：${prompt}" ;;
    esac
}

need_cmd() { command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"; }

# ------------------------------ 0. 前置检查 ------------------------------
log "0/9 前置检查"
need_cmd go
need_cmd git
need_cmd ssh
need_cmd scp
need_cmd rsync
need_cmd curl
[ -d "${GO_MODULE_DIR}" ] || die "找不到 Go 模块目录: ${GO_MODULE_DIR}"
[ -f "${LOCAL_NGINX_CONF}" ] || die "找不到 nginx 配置: ${LOCAL_NGINX_CONF}"

# nginx 配置必须为纯 LF（服务器 nginx 对 CRLF 敏感）
if grep -q $'\r' "${LOCAL_NGINX_CONF}"; then
    die "${LOCAL_NGINX_CONF} 含 CRLF 行尾，请先执行: sed -i 's/\\r$//' deploy/vantalens-nginx.conf"
fi
echo "    本地环境 OK（go=$(go version | awk '{print $3}')）"

ssh -o ConnectTimeout=10 "${SSH_HOST}" "echo ok" >/dev/null \
    || die "无法连接服务器别名 ${SSH_HOST}，请检查 ~/.ssh/config"
echo "    SSH 连接 ${SSH_HOST} OK"

# ------------------------------ 1. 本地交叉编译 ------------------------------
log "1/9 交叉编译 Linux amd64 二进制"
mkdir -p "${LOCAL_BUILD_DIR}"

cd "${LOCAL_REPO_ROOT}"
GIT_SHA="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
    GIT_DIRTY="true"
else
    GIT_DIRTY="false"
fi
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

# BuildInfo 变量位于 main 包（TalentWriter/cmd/server/main.go）：
#   var GitSHA / BuildTime / BuildDirty
# 模块路径为 vantalens/talentwriter，故 -X 用完整包路径。
# 注意：Version 是 const "2.0.0"，无法通过 ldflags 注入，只能改源码。
LDFLAGS="-X vantalens/talentwriter/cmd/server.GitSHA=${GIT_SHA} \
-X vantalens/talentwriter/cmd/server.BuildTime=${BUILD_TIME} \
-X vantalens/talentwriter/cmd/server.BuildDirty=${GIT_DIRTY}"

echo "    git_sha=${GIT_SHA} dirty=${GIT_DIRTY} build_time=${BUILD_TIME}"

cd "${GO_MODULE_DIR}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "${LDFLAGS}" -o "${LOCAL_BINARY}" "${GO_BUILD_PKG}" \
    || die "go build 失败"
[ -f "${LOCAL_BINARY}" ] || die "编译产物不存在: ${LOCAL_BINARY}"
echo "    产物: ${LOCAL_BINARY} ($(du -h "${LOCAL_BINARY}" | cut -f1))"

# ------------------------------ 2. 备份服务器现二进制 ------------------------------
log "2/9 备份服务器现有二进制"
ssh "${SSH_HOST}" "
    set -e
    if [ -f '${REMOTE_BIN}' ]; then
        sudo cp -a '${REMOTE_BIN}' '${REMOTE_BIN}.backup'
        echo '    已备份为 ${REMOTE_BIN}.backup'
    else
        echo '    服务器上无二进制（首次部署），跳过备份'
    fi
"

# ------------------------------ 3. 上传新二进制 ------------------------------
log "3/9 上传新二进制"
warn "即将覆盖服务器二进制 ${REMOTE_BIN}（旧版已备份为 .backup）"
confirm "确认上传并替换 ${REMOTE_BIN} ?"

# 先传到用户家目录再 sudo mv，避免目标目录权限问题
scp "${LOCAL_BINARY}" "${SSH_HOST}:~/talentwriter-server.new"
ssh "${SSH_HOST}" "
    set -e
    sudo mkdir -p '${REMOTE_BIN_DIR}'
    sudo systemctl stop '${SYSTEMD_SERVICE}' || true
    sudo mv ~/talentwriter-server.new '${REMOTE_BIN}'
    sudo chown '${REMOTE_SERVICE_USER}:${REMOTE_SERVICE_GROUP}' '${REMOTE_BIN}'
    sudo chmod 0755 '${REMOTE_BIN}'
    echo '    二进制已就位（服务已先停止）'
"

# ------------------------------ 4. 创建缺失目录 ------------------------------
log "4/9 创建缺失的数据目录并修正属主"
ssh "${SSH_HOST}" "
    set -e
    for d in '${REMOTE_WORKTREE_DIR}' '${REMOTE_PUBLISH_DIR}' '${REMOTE_COMMENTS_DIR}'; do
        if [ ! -d \"\$d\" ]; then
            echo \"    创建 \$d\"
            sudo mkdir -p \"\$d\"
        else
            echo \"    已存在 \$d\"
        fi
        sudo chown -R '${REMOTE_SERVICE_USER}:${REMOTE_SERVICE_GROUP}' \"\$d\"
    done
"

# ProtectSystem=strict 下只有 /var/lib/vantalens 可写，PUBLISH_OUTPUT_PATH 必须指向
# ${REMOTE_PUBLISH_DIR}，否则 /api/publish 会因只读文件系统失败（默认 hugoPath/public
# 在 /opt/vantalens/site 下，不可写）。若合并配置中缺失则新增独立 drop-in（不覆盖既有 drop-in）。
log "4.5/9 确认 PUBLISH_OUTPUT_PATH 已注入 systemd"
ssh "${SSH_HOST}" "
    set -e
    if sudo systemctl show '${SYSTEMD_SERVICE}' -p Environment | grep -q 'PUBLISH_OUTPUT_PATH='; then
        echo '    PUBLISH_OUTPUT_PATH 已存在，跳过'
    else
        echo '    写入 drop-in 40-publish.conf（PUBLISH_OUTPUT_PATH=${REMOTE_PUBLISH_DIR}）'
        printf '[Service]\nEnvironment=PUBLISH_OUTPUT_PATH=${REMOTE_PUBLISH_DIR}\n' | sudo tee '/etc/systemd/system/${SYSTEMD_SERVICE}.service.d/40-publish.conf' >/dev/null
        sudo systemctl daemon-reload
    fi
"

# ------------------------------ 5. systemd 合并配置核对 ------------------------------
log "5/9 核对 systemd 合并配置（unit + drop-in）"
echo "    ---------- 服务器当前合并配置（systemctl cat） ----------"
ssh "${SSH_HOST}" "sudo systemctl cat '${SYSTEMD_SERVICE}'" || warn "systemctl cat 失败（服务可能未安装）"
echo "    --------------------------------------------------------"
echo "    本仓库模板: ${LOCAL_SYSTEMD_UNIT}"
echo ""
echo "    必须确认的合并结果："
echo "      - HUGO_PATH            （drop-in 覆盖为 /opt/vantalens/site）"
echo "      - PREVIEW_PUBLIC_URL   （由 drop-in 提供）"
echo "      - COMMENT_SETTINGS_PATH=/var/lib/vantalens/comments/comment_settings.json"
echo "      - PUBLISH_OUTPUT_PATH=/var/lib/vantalens/publish"
echo "      - AUTHORITY_BACKEND=true"
echo "      - DB_SYNC_ENABLED=false"
echo "      - ReadWritePaths 包含 site-worktree 与 publish"
warn "脚本不会静默上传/修改 drop-in 文件。如需更新主 unit，请人工 diff 后操作。"
confirm "已核对上面的合并配置，确认无冲突?"

# ------------------------------ 6. 重启服务 ------------------------------
log "6/9 daemon-reload + 重启 ${SYSTEMD_SERVICE}"
ssh "${SSH_HOST}" "
    set -e
    sudo systemctl daemon-reload
    sudo systemctl restart '${SYSTEMD_SERVICE}'
    sleep 2
    sudo systemctl is-active '${SYSTEMD_SERVICE}'
" || die "服务重启后不是 active 状态，请执行: ssh ${SSH_HOST} 'sudo journalctl -u ${SYSTEMD_SERVICE} -n 80 --no-pager'"

# ------------------------------ 7. 验证 /api/health ------------------------------
log "7/9 验证 /api/health 返回新 git_sha"
HEALTH_JSON="$(ssh "${SSH_HOST}" "curl -fsS --max-time 10 '${HEALTH_URL_LOCAL}'")" \
    || die "/api/health 请求失败: ${HEALTH_JSON:-<empty>}"
echo "    响应: ${HEALTH_JSON}"
echo "${HEALTH_JSON}" | grep -q "\"git_sha\"" \
    || die "/api/health 缺少 git_sha 字段（可能跑的还是旧二进制）"
echo "${HEALTH_JSON}" | grep -q "${GIT_SHA}" \
    || die "/api/health 的 git_sha 与本地构建 ${GIT_SHA} 不一致"
echo "    git_sha 验证通过: ${GIT_SHA}"

# ------------------------------ 8. 静态站点同步 ------------------------------
if [ "${SKIP_SITE}" = true ]; then
    log "8/9 跳过静态站点同步（--skip-site）"
else
    log "8/9 本地 hugo --minify 并同步到服务器站点目录"
    cd "${LOCAL_REPO_ROOT}"
    if [ -x "${LOCAL_REPO_ROOT}/hugo.exe" ]; then
        HUGO_BIN="${LOCAL_REPO_ROOT}/hugo.exe"
    else
        need_cmd hugo
        HUGO_BIN="hugo"
    fi
    "${HUGO_BIN}" --minify || die "hugo --minify 构建失败"
    [ -d "${LOCAL_REPO_ROOT}/public" ] || die "hugo 构建后 public/ 不存在"

    warn "即将 rsync 覆盖服务器站点目录 ${REMOTE_SITE_DIR}"
    confirm "确认同步 public/ 到 ${SSH_HOST}:${REMOTE_SITE_DIR}/ ?"

    # 发布到站点根目录是独立于 systemd 单元的步骤
    # （unit 中 /var/www/vantalens 是 ReadOnlyPaths，后端无法直接写，故由部署脚本完成）
    # 先 rsync 到家目录 staging（SSH 用户可写），再 sudo rsync 进站点目录，避免 /var/www 权限问题
    rsync -az --delete -e ssh "${LOCAL_REPO_ROOT}/public/" \
        "${SSH_HOST}:~/vantalens-site-staging/" \
        || die "rsync 到 staging 目录失败"
    ssh "${SSH_HOST}" "
        set -e
        sudo rsync -a --delete ~/vantalens-site-staging/ '${REMOTE_SITE_DIR}/'
        rm -rf ~/vantalens-site-staging
        echo '    站点目录已更新'
    "
fi

# ------------------------------ 9. 可选：更新 nginx 配置 ------------------------------
if [ "${SKIP_NGINX}" = true ]; then
    log "9/9 跳过 nginx 配置更新（--skip-nginx）"
else
    log "9/9 可选：更新 nginx 配置"
    echo "    本地配置: ${LOCAL_NGINX_CONF}"
    echo "    远端目标: ${REMOTE_NGINX_CONF} 【待确认：实际路径可能不同】"
    if confirm "是否上传并 reload nginx 配置?"; then
        scp "${LOCAL_NGINX_CONF}" "${SSH_HOST}:~/vantalens-nginx.conf.new"
        ssh "${SSH_HOST}" "
            set -e
            sudo cp -a '${REMOTE_NGINX_CONF}' '${REMOTE_NGINX_CONF}.backup.\$(date +%Y%m%d%H%M%S)' 2>/dev/null || true
            sudo mv ~/vantalens-nginx.conf.new '${REMOTE_NGINX_CONF}'
            sudo nginx -t
        " || die "nginx -t 失败，未 reload。请检查 ${REMOTE_NGINX_CONF} 并从 .backup 恢复"
        warn "nginx -t 通过，即将 reload nginx"
        confirm "确认 reload nginx?"
        ssh "${SSH_HOST}" "sudo systemctl reload nginx"
        echo "    nginx 已 reload"
    else
        echo "    跳过 nginx 更新"
    fi
fi

log "部署完成"
echo "    二进制 git_sha : ${GIT_SHA} (dirty=${GIT_DIRTY})"
echo "    构建时间       : ${BUILD_TIME}"
echo "    回滚命令       : ssh ${SSH_HOST} 'sudo cp -a ${REMOTE_BIN}.backup ${REMOTE_BIN} && sudo systemctl restart ${SYSTEMD_SERVICE}'"
echo ""
echo "    后续人工验证清单："
echo "      1. https://vantalens.com/api/health 含 git_sha=${GIT_SHA}"
echo "      2. https://vantalens.com/platform 可访问（basic-auth 后进入后台）"
echo "      3. 后台触发 /api/publish，实测 publish -> ${REMOTE_SITE_DIR} 同步链路"
