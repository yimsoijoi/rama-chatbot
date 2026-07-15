#!/usr/bin/env bash
# Create all rich menus via the LINE Messaging API, upload their images,
# set the DX selector as the account default, and write the returned IDs
# into configs/faq_seed.yaml.
#
# Prereqs:
#   - export LINE_CHANNEL_TOKEN="<long-lived channel access token>"
#   - put background images next to this script:
#       dx-select.png  dx1.png  dx2.png  dx3.png  dx4.png  dx5.png
#     (.jpg / .jpeg also accepted; 2500x1686, <=1MB each)
#
# Usage:  ./create-richmenus.sh
set -euo pipefail
cd "$(dirname "$0")"

: "${LINE_CHANNEL_TOKEN:?Please: export LINE_CHANNEL_TOKEN=...}"
API="https://api.line.me"
DATA="https://api-data.line.me"

ctype() { case "$1" in *.jpg|*.jpeg) echo "image/jpeg";; *) echo "image/png";; esac; }

img_for() { # base -> path of first existing image, or empty
  for e in png jpg jpeg; do [ -f "$1.$e" ] && { echo "$1.$e"; return; }; done
  echo ""
}

create() { # json image -> echoes richMenuId
  local json="$1" img="$2" id
  id=$(curl -sS -X POST "$API/v2/bot/richmenu" \
        -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" \
        -H "Content-Type: application/json" \
        -d @"$json" \
      | python3 -c 'import sys,json; print(json.load(sys.stdin).get("richMenuId",""))')
  [ -n "$id" ] || { echo "ERROR: create failed for $json" >&2; return 1; }
  curl -sS -X POST "$DATA/v2/bot/richmenu/$id/content" \
        -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" \
        -H "Content-Type: $(ctype "$img")" \
        --data-binary @"$img" >/dev/null
  echo "$id"
}

# --- preflight: all images present? ---
missing=0
for base in dx-select dx1 dx2 dx3 dx4 dx5; do
  if [ -z "$(img_for "$base")" ]; then echo "MISSING image: $base.png"; missing=1; fi
done
[ "$missing" = 0 ] || { echo "Add the missing images and re-run."; exit 1; }

# --- default selector menu ---
echo "Creating DX selector (default menu)..."
SELECT_ID=$(create dx-select.json "$(img_for dx-select)")
curl -sS -X POST "$API/v2/bot/user/all/richmenu/$SELECT_ID" \
     -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" >/dev/null
echo "  selector = $SELECT_ID (set as default)"

# --- per-DX menus ---
PAIRS=""
for n in 1 2 3 4 5; do
  echo "Creating DX$n menu..."
  ID=$(create "dx$n.json" "$(img_for "dx$n")")
  echo "  dx$n = $ID"
  PAIRS="$PAIRS d$n=$ID"
done

# --- write IDs into faq_seed.yaml ---
echo "Patching configs/faq_seed.yaml..."
# shellcheck disable=SC2086
python3 patch-richmenu-ids.py $PAIRS

cat <<EOF

Done. Next:
  1. git add configs/faq_seed.yaml && git commit && push  (triggers redeploy)
  2. Test in LINE: add friend -> selector menu -> tap เลือก DX1 -> DX1 menu appears
EOF
