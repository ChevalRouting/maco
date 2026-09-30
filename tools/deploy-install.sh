#!/bin/bash
set -euo pipefail

cd -- "$(dirname -- "$0")"
chmod +x maco
codesign --sign - --force --preserve-metadata=entitlements,requirements,flags,runtime maco
codesign --verify --strict maco
sudo -v
sudo ./maco service uninstall
sudo ./maco install "$@"
