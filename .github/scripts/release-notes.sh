#!/usr/bin/env bash
# Prints the release notes for one version: that version's section of every package changelog,
# without Changesets' bookkeeping ("Updated dependencies", "No changes in this release"), under a
# heading per package when more than one has entries.
#
#   bash .github/scripts/release-notes.sh 1.2.0
set -euo pipefail

version="$1"

section() {
  awk -v heading="## $version" '
    $0 == heading { found = 1; next }
    /^## / { found = 0 }
    !found { next }
    /^No changes in this release\.$/ { next }
    /^- Updated dependencies/ { deps = 1; next }
    deps && /^  - / { next }
    { deps = 0; lines[++n] = $0 }
    END {
      for (i = 1; i <= n; i++) {
        if (lines[i] ~ /^### /) {
          keep = 0
          for (j = i + 1; j <= n && lines[j] !~ /^#/; j++) if (lines[j] ~ /^- /) { keep = 1; break }
          if (!keep) continue
        }
        print lines[i]
      }
    }' "$1" | cat -s | sed '/./,$!d'
}

names=()
bodies=()
for entry in "apps/web:App" "backend:API" "packages/ui:UI kit" "packages/api:API client"; do
  f="${entry%%:*}/CHANGELOG.md"
  [ -f "$f" ] || continue
  body=$(section "$f")
  [ -n "${body//[[:space:]]/}" ] || continue
  names+=("${entry#*:}")
  bodies+=("$body")
done

if [ "${#bodies[@]}" -eq 0 ]; then
  echo "no changelog entry for $version" >&2
  exit 1
fi
if [ "${#bodies[@]}" -eq 1 ]; then
  printf '%s\n' "${bodies[0]}"
  exit 0
fi
for i in "${!bodies[@]}"; do
  printf '## %s\n\n%s\n\n' "${names[$i]}" "${bodies[$i]}"
done
