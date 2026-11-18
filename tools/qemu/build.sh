#!/bin/bash
set -euo pipefail

root=$(cd -- "$(dirname -- "$0")" && pwd)
output=${1:?usage: build.sh output-directory}
prefix=/usr/local/lib/maco/qemu
work=$(cd -- "$root/../.." && pwd)/build/qemu
version=$(cat "$root/VERSION")
archive=$work/qemu-$version.tar.xz
url=https://download.qemu.org/qemu-$version.tar.xz
shared=(keymaps/en-us efi-virtio.rom)

mkdir -p "$work"
if ! (cd -- "$work" && shasum -a 256 -c --status "$root/SHA256SUMS"); then
  curl -fL --retry 3 -o "$archive" "$url"
  (cd -- "$work" && shasum -a 256 -c "$root/SHA256SUMS")
fi

brew=$(brew --prefix)
dtc=$(brew --prefix dtc)
rm -rf "$work/src" "$output"
mkdir -p "$work/src" "$output"
output=$(cd -- "$output" && pwd)
tar -xJf "$archive" -C "$work/src" --strip-components 1
for patch in "$root"/patches/*.patch; do
  patch -d "$work/src" -p1 --quiet < "$patch"
done

mkdir -p "$work/src/build"
cd "$work/src/build"
PKG_CONFIG_PATH="$dtc/lib/pkgconfig:$brew/lib/pkgconfig" ../configure \
  --prefix="$prefix" --target-list=aarch64-softmmu --enable-hvf --enable-fdt=system \
  --enable-slirp --enable-libusb --enable-vnc --enable-png --enable-vnc-jpeg --enable-zstd \
  --enable-vmnet --enable-tools --audio-drv-list= \
  --disable-docs --disable-guest-agent --disable-gtk --disable-sdl --disable-cocoa \
  --disable-curses --disable-dbus-display --disable-coreaudio --disable-pvg \
  --disable-gnutls --disable-gcrypt --disable-nettle --disable-libssh --disable-capstone --disable-curl --disable-vnc-sasl \
  --disable-auth-pam --disable-spice-protocol --disable-libiscsi --disable-usb-redir \
  --disable-virtfs --disable-plugins --disable-werror \
  --extra-cflags="-I$dtc/include" --extra-ldflags="-L$dtc/lib -Wl,-headerpad_max_install_names"
ninja -j"$(sysctl -n hw.ncpu)"
staging=$work/staging
rm -rf "$staging"
pyvenv/bin/meson install --no-rebuild --destdir "$staging"

tree=$output$prefix
mkdir -p "$tree/bin" "$tree/lib" "$tree/share/qemu" "$tree/share/doc/licenses"
cp "$staging$prefix/bin/qemu-system-aarch64" "$staging$prefix/bin/qemu-img" "$tree/bin/"
strip -x "$tree"/bin/* 2>/dev/null
for name in "${shared[@]}"; do
  mkdir -p "$(dirname -- "$tree/share/qemu/$name")"
  cp "$staging$prefix/share/qemu/$name" "$tree/share/qemu/$name"
done

system() {
  case $1 in
    /usr/lib/* | /System/*) return 0 ;;
  esac
  return 1
}

dependencies() {
  otool -L "$1" | awk -v self="$(otool -D "$1" | tail -n +2)" 'NR > 1 && $1 != self { print $1 }'
}

rpaths() {
  otool -l "$1" | awk '$1 == "cmd" { command = $2 } $1 == "path" && command == "LC_RPATH" { print $2 }'
}

resolve() {
  local dependency=$1 source=$2 path candidate
  case $dependency in
    @loader_path/* | @executable_path/*)
      candidate=$(dirname -- "$source")/${dependency#*/}
      ;;
    @rpath/*)
      candidate=
      for path in $(rpaths "$source") "$brew/lib"; do
        path=${path/@loader_path/$(dirname -- "$source")}
        if [ -e "$path/${dependency#@rpath/}" ]; then
          candidate=$path/${dependency#@rpath/}
          break
        fi
      done
      ;;
    *)
      candidate=$dependency
      ;;
  esac
  [ -e "$candidate" ] || { echo "cannot resolve $dependency for $source" >&2; exit 1; }
  realpath "$candidate"
}

origins=$work/origins
: > "$origins"
origin() {
  awk -F '\t' -v file="$1" '$1 == file { print $2 }' "$origins"
}

queue=()
for binary in "$tree"/bin/*; do
  printf '%s\t%s\n' "$binary" "$work/src/build/$(basename -- "$binary")" >> "$origins"
  queue+=("$binary")
done

index=0
while [ "$index" -lt ${#queue[@]} ]; do
  file=${queue[$index]}
  index=$((index + 1))
  chmod u+w "$file"
  for dependency in $(dependencies "$file"); do
    system "$dependency" && continue
    source=$(resolve "$dependency" "$(origin "$file")")
    name=$(basename -- "$source")
    target=$tree/lib/$name
    if [ ! -e "$target" ]; then
      cp "$source" "$target"
      chmod u+w "$target"
      install_name_tool -id "@rpath/$name" "$target" 2>/dev/null
      printf '%s\t%s\n' "$target" "$source" >> "$origins"
      queue+=("$target")
    fi
    install_name_tool -change "$dependency" "@rpath/$name" "$file" 2>/dev/null
  done
  for path in $(rpaths "$file"); do
    install_name_tool -delete_rpath "$path" "$file" 2>/dev/null
  done
  case $file in
    "$tree"/bin/*) install_name_tool -add_rpath @executable_path/../lib "$file" 2>/dev/null ;;
    *) install_name_tool -add_rpath @loader_path "$file" 2>/dev/null ;;
  esac
done

for library in "$tree"/lib/*.dylib; do
  codesign --force --sign - "$library"
done
codesign --force --sign - "$tree/bin/qemu-img"
codesign --force --sign - --entitlements "$work/src/accel/hvf/entitlements.plist" "$tree/bin/qemu-system-aarch64"

licenses=$tree/share/doc/licenses
mkdir -p "$licenses/qemu"
cp "$work/src/COPYING" "$work/src/COPYING.LIB" "$work/src/LICENSE" "$licenses/qemu/"
cut -f2 "$origins" | while read -r source; do
  case $source in
    "$brew"/Cellar/*) ;;
    *) continue ;;
  esac
  keg=${source#"$brew"/Cellar/}
  formula=${keg%%/*}
  keg=$brew/Cellar/$formula/$(cut -d/ -f2 <<< "$keg")
  mkdir -p "$licenses/$formula"
  find "$keg" -maxdepth 1 -type f \( -iname 'COPYING*' -o -iname 'LICENSE*' -o -iname 'LICENCE*' -o -iname 'COPYRIGHT*' -o -iname 'NOTICE*' -o -iname '*GPL*' \) -exec cp {} "$licenses/$formula/" \;
  license=$(brew info --json=v1 "$formula" | plutil -extract 0.license raw -o - -)
  printf '%s %s (%s)\n' "$formula" "$(basename -- "$keg")" "$license" >> "$licenses/versions"
done
sort -u -o "$licenses/versions" "$licenses/versions"
find "$licenses" -type d -empty -delete

cat > "$tree/NOTICE" <<EOF
QEMU $version for maco

This directory contains QEMU, licensed under the GNU General Public License
version 2 (see share/doc/licenses/qemu), built from the official release:

  $url
  sha256 $(awk '{ print $1 }' "$root/SHA256SUMS")

with the patches in tools/qemu/patches of the maco repository
(https://github.com/ChevalRouting/maco) applied.

The libraries in lib/ are copies of the Homebrew builds listed in
share/doc/licenses/versions, changed only in their install names and code
signatures; their license files are in share/doc/licenses.
EOF
