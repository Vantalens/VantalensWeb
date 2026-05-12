#!/usr/bin/env bash
# Re-apply local fixes to theme-earth on the Halo server.
#
# Two upstream bugs we work around:
#
#   1) `#theme.assets('/images/default-avatar.svg')` resolves to /halo/assets/...
#      which Halo does not serve (404). The default-avatar file actually lives
#      at /halo/themes/theme-earth/assets/images/default-avatar.svg. We replace
#      the broken helper with a literal path; the corresponding nginx mirror
#      (`location ^~ /themes/`) routes it to the right backend path.
#
#   2) `${owner.permalink}` for users (authors) is stored by Halo with the
#      `/halo/` context-path baked in, while category/tag/post permalinks are
#      not. Wrapping it in Thymeleaf's `@{...}` adds the prefix again, yielding
#      `/halo/halo/authors/<name>`. Stripping the `@{...}` wrap is correct only
#      for owner.permalink — leave other permalink fields wrapped.
#
# Run on the server (wj) as root after a theme update or reinstall. Idempotent.

set -euo pipefail

TROOT="${TROOT:-/opt/halo/halo2/themes/theme-earth/templates}"
[ -d "$TROOT" ] || { echo "theme not found at $TROOT" >&2; exit 1; }

TS=$(date +%Y%m%d-%H%M%S)
cp -r "$TROOT" "$TROOT.bak-$TS"
echo "backup: $TROOT.bak-$TS"

# Bug 1: default-avatar
grep -rl "#theme.assets" "$TROOT" | xargs --no-run-if-empty sed -i \
  "s|#theme\.assets('/images/default-avatar.svg')|'/themes/theme-earth/assets/images/default-avatar.svg'|g"

# Bug 2: author permalink double-prefix (only owner.permalink fields)
grep -rl "owner.permalink" "$TROOT" | xargs --no-run-if-empty sed -i \
  -e 's|@{${post.owner.permalink}}|${post.owner.permalink}|g' \
  -e 's|@{${singlePage.owner.permalink}}|${singlePage.owner.permalink}|g'

echo "patches applied. Restart halo to clear Thymeleaf template cache:"
echo "  docker restart halo"
