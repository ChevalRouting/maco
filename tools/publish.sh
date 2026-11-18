#!/bin/bash
set -euo pipefail

host=${1:?usage: publish.sh user@host version url}
version=${2:?version required}
url=${3:?public url required, like https://maco.chevalrouting.org}

case "$host" in
  -*|*[!a-zA-Z0-9_.@:\[\]-]*)
    echo 'Use an SSH hostname, user@host, or SSH config alias for HOST.' >&2
    exit 1
    ;;
esac

case "$version" in
  *[!0-9A-Za-z.+-]*|'')
    echo "Invalid version: $version" >&2
    exit 1
    ;;
esac

case "$url" in
  https://*) url=${url%/} ;;
  *)
    echo "Use an https:// URL: $url" >&2
    exit 1
    ;;
esac
case "$url" in
  *[!a-zA-Z0-9:/._-]*)
    echo "Invalid URL: $url" >&2
    exit 1
    ;;
esac

root=$(cd -- "$(dirname -- "$0")/.." && pwd)
package="$root/maco-$version.pkg"
binary="$root/maco"
for file in "$package" "$binary"; do
  if [ ! -f "$file" ]; then
    echo "File not found: $file (run task pkg VERSION=$version first)" >&2
    exit 1
  fi
done

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/downloads" "$stage/screenshots"
sed -e "s/@VERSION@/$version/g" -e "s#@URL@#$url#g" "$root/site/index.html" > "$stage/index.html"
cp "$root/web/public/logo.svg" "$stage/logo.svg"
cp "$root"/docs/screenshots/*.png "$stage/screenshots/"
cp "$package" "$stage/downloads/maco-$version.pkg"
COPYFILE_DISABLE=1 tar --no-mac-metadata --no-xattrs -czf "$stage/downloads/maco-$version-darwin-arm64.tar.gz" -C "$root" maco
(cd "$stage/downloads" && shasum -a 256 "maco-$version.pkg" "maco-$version-darwin-arm64.tar.gz" > SHA256SUMS)

printf 'Publishing maco %s to %s\n' "$version" "$host"
COPYFILE_DISABLE=1 tar --no-mac-metadata --no-xattrs -C "$stage" -czf - . | ssh -- "$host" 'tar -xzf - -C /var/lib/maco-static/site'
printf 'Published maco %s.\n' "$version"
