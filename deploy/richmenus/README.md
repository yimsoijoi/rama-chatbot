<!-- AI-DRAFT -->

# Rich menu definitions + creator

Ready-to-use LINE rich menus for the choose-DX flow. Full explanation:
[../../docs/line-rich-menu-dx1-guide.md](../../docs/line-rich-menu-dx1-guide.md).

## Files

| File | What it is |
| --- | --- |
| `dx-select.json` | Default menu: 5 DX buttons + staff handoff |
| `dx1.json` … `dx5.json` | Per-DX menus: 4 FAQ buttons + handoff + emergency |
| `create-richmenus.sh` | Creates all menus, uploads images, sets default, writes IDs into `faq_seed.yaml` |
| `patch-richmenu-ids.py` | Helper: writes returned IDs into `configs/faq_seed.yaml` |

Each button uses a `message` action whose text is the **full question** — it matches the
`match_phrases` in [../../configs/faq_seed.yaml](../../configs/faq_seed.yaml). Edit a `.json`
to change which question a cell asks (paste the exact question text).

## How to run

1. **Prepare 6 background images** (2500×1686 px, ≤1 MB, PNG or JPG) in this folder:
   `dx-select.png`, `dx1.png`, `dx2.png`, `dx3.png`, `dx4.png`, `dx5.png`
   - Label the DX-selector cells with plain-language result groups (see the guide).
   - Cell layout = 3 columns × 2 rows, matching the `areas` order in each JSON.
2. **Set your token** (use the rotated one, keep it secret):
   ```bash
   export LINE_CHANNEL_TOKEN="<long-lived channel access token>"
   ```
3. **Run:**
   ```bash
   ./create-richmenus.sh
   ```
4. **Commit** the updated config and redeploy:
   ```bash
   git add ../../configs/faq_seed.yaml && git commit -m "chore: wire rich menu ids" && git push
   ```

## Re-running / updating a single menu

Editing a menu means creating a new one (rich menus are immutable). Re-run the script (it makes new
IDs and re-patches the config), then delete the old menus you no longer need:

```bash
curl -s https://api.line.me/v2/bot/richmenu/list \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" | python3 -m json.tool
curl -s -X DELETE https://api.line.me/v2/bot/richmenu/<OLD_ID> \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN"
```
