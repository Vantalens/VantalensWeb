#!/usr/bin/env bash
# Public-facing Halo diagnostic. Run from any host with internet access.
# Usage: bash halo_form_diag.sh
set -euo pipefail

BASE="${BASE:-https://vantalens.com}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsS "$BASE/halo/login?authentication_required" -o "$TMP/login.html"
curl -fsS "$BASE/halo/"                              -o "$TMP/root.html"

printf '== login form fields ==\n'
grep -nE '<form|name=|action=|csrf|remember|password|username' "$TMP/login.html" | head -n 40 || true

printf '\n== login page resource refs ==\n'
grep -Eo '(src|href)="[^"]+"' "$TMP/login.html" | sort -u

printf '\n== halo root resource refs (first 40) ==\n'
grep -Eo '(src|href)="[^"]+"' "$TMP/root.html" | sort -u | head -n 40

printf '\n== probe selected paths ==\n'
for path in \
  /halo/                                                       \
  /halo/login                                                  \
  /halo/console/                                               \
  /halo/styles/main.css                                        \
  /halo/themes/theme-earth/assets/main-BaWd_1hx.css            \
  /halo/plugins/PluginCommentWidget/assets/static/index.css    \
  /halo/apis/api.content.halo.run/v1alpha1/posts               \
  /halo/assets/images/default-avatar.svg                       \
  /styles/main.css                                             \
  /themes/theme-earth/assets/main-BaWd_1hx.css                 \
  /apis/api.content.halo.run/v1alpha1/posts                    ; do
  printf '%-72s ' "$path"
  curl -sS -o /dev/null -w '%{http_code} %{content_type}\n' "$BASE$path"
done
