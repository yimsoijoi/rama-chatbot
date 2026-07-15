#!/usr/bin/env python3
"""Write rich menu IDs into configs/faq_seed.yaml (preserves comments/formatting).

Usage:
    python3 patch-richmenu-ids.py d1=richmenu-aaa d2=richmenu-bbb ...
"""
import os
import re
import sys

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
CONFIG = os.path.join(REPO_ROOT, "configs", "faq_seed.yaml")


def main(pairs):
    with open(CONFIG, encoding="utf-8") as f:
        text = f.read()

    for pair in pairs:
        if "=" not in pair:
            print(f"skip (bad arg): {pair}", file=sys.stderr)
            continue
        dx, menu_id = pair.split("=", 1)
        dx, menu_id = dx.strip(), menu_id.strip()
        # Scope to the dx block, then replace its first rich_menu_id value only.
        pat = re.compile(
            r'(^  %s:\n(?:.*\n)*?    rich_menu_id: )"[^"]*"' % re.escape(dx),
            re.MULTILINE,
        )
        text, n = pat.subn(r'\1"%s"' % menu_id, text, count=1)
        if n:
            print(f"  {dx} -> {menu_id}")
        else:
            print(f"  WARNING: could not find block for {dx}", file=sys.stderr)

    with open(CONFIG, "w", encoding="utf-8") as f:
        f.write(text)
    print(f"Updated {CONFIG}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)
    main(sys.argv[1:])
