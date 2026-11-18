#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "$0")" && pwd)
output=${1:?usage: image.sh output-directory}
release=3.24.2
mirror=https://dl-cdn.alpinelinux.org/alpine
netboot=$mirror/v${release%.*}/releases/aarch64/netboot-$release
packages=$mirror/v${release%.*}/main/aarch64

mkdir -p "$output"
cd "$output"
for file in vmlinuz-virt initramfs-virt; do
  if ! grep " $file\$" "$root/SHA256SUMS" | shasum -a 256 -c --status 2>/dev/null; then
    curl -fL --retry 3 -o "$file" "$netboot/$file"
  fi
done

shasum -a 256 -c "$root/SHA256SUMS"
curl -fsSL -o APKINDEX.tar.gz "$packages/APKINDEX.tar.gz"
iperf=$(tar -xzOf APKINDEX.tar.gz APKINDEX | awk -F: '$1 == "P" { name = $2 } $1 == "V" && name == "iperf3" { version = $2 } END { print version }')
curl -fsSL -o iperf3.apk "$packages/iperf3-$iperf.apk"
echo "iperf3 $iperf"

rm -rf overlay
mkdir -p overlay
tar -xzf iperf3.apk -C overlay usr 2>/dev/null
cp "$root/init" overlay/init
chmod 0755 overlay/init
(cd overlay && find . | cpio -o -H newc -R 0:0 --quiet | gzip -9) > overlay.cpio.gz
cat initramfs-virt overlay.cpio.gz > initramfs.gz
