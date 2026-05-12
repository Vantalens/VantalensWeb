#!/usr/bin/env bash
# Direct-to-Halo (loopback) probe. MUST run on the server hosting Halo.
# Confirms whether Halo backend serves a path before blaming nginx for it.
# Usage: bash halo_internal_paths.sh
set -euo pipefail

HALO="${HALO:-http://127.0.0.1:8090}"

for path in \
  /                                                            \
  /login                                                       \
  /console/                                                    \
  /styles/main.css                                             \
  /themes/theme-earth/assets/main-BaWd_1hx.css                 \
  /plugins/PluginCommentWidget/assets/static/index.css         \
  /apis/api.content.halo.run/v1alpha1/posts                    \
  /assets/images/default-avatar.svg                            \
  /halo/                                                       \
  /halo/login                                                  \
  /halo/themes/theme-earth/assets/main-BaWd_1hx.css            \
  /halo/apis/api.content.halo.run/v1alpha1/posts               ; do
  printf '%-60s ' "$path"
  curl -sS -o /dev/null -w '%{http_code} %{content_type}\n' "$HALO$path"
done

printf '\n== Halo HTML link prefix sample ==\n'
curl -fsS "$HALO/" | grep -Eo '(src|href)="[^"]+"' | sort -u | head -n 20
