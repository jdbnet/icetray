#!/usr/bin/env bash
set -euo pipefail

# Sync the project version into Wails v3 metadata used for Windows exe
# resources, NSIS installers, and Android versionName/versionCode.
cd "$(dirname "$0")/.."

resolve_version() {
  if [ -n "${VERSION:-}" ]; then
    printf '%s' "${VERSION}"
    return
  fi

  if [ -f VERSION ]; then
    tr -d '[:space:]' < VERSION
    return
  fi

  if git describe --tags --abbrev=0 >/dev/null 2>&1; then
    git describe --tags --abbrev=0 | sed 's/^v//'
    return
  fi

  echo "Set VERSION, add a VERSION file, or create a git tag" >&2
  exit 1
}

version="$(resolve_version)"

python3 - "${version}" <<'PY'
import json
import re
import sys
from pathlib import Path

version = sys.argv[1]
config = Path("build/config.yml")
text = config.read_text()
updated, count = re.subn(
    r'(^info:\n(?:.*\n)*?  version: )".*"',
    rf'\1"{version}"',
    text,
    count=1,
    flags=re.M,
)
if count != 1:
    raise SystemExit("failed to update info.version in build/config.yml")
config.write_text(updated)

info_path = Path("build/windows/info.json")
if info_path.is_file():
    data = json.loads(info_path.read_text())
    data.setdefault("fixed", {})["file_version"] = version
    data.setdefault("info", {}).setdefault("0000", {})["ProductVersion"] = version
    info_path.write_text(json.dumps(data, indent="\t") + "\n")

nsh = Path("build/windows/nsis/wails_tools.nsh")
if nsh.is_file():
    nsh.write_text(re.sub(
        r'(!define INFO_PRODUCTVERSION )"\d+\.\d+\.\d+"',
        rf'\1"{version}"',
        nsh.read_text(),
        count=1,
    ))

nfpm = Path("build/linux/nfpm/nfpm.yaml")
if nfpm.is_file():
    nfpm.write_text(re.sub(
        r'^version: ".*"$',
        f'version: "{version}"',
        nfpm.read_text(),
        count=1,
        flags=re.M,
    ))
PY

echo "==> Version ${version} synced to build/config.yml"
