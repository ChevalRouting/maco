#!/bin/bash
set -euo pipefail

host=${1:?usage: deploy.sh user@host binary [install arguments...]}
binary=${2:?binary path required}
shift 2

case "$host" in
  -*|*[!a-zA-Z0-9_.@:\[\]-]*)
    echo 'Use an SSH hostname, user@host, or SSH config alias for HOST.' >&2
    exit 1
    ;;
esac

if [ ! -f "$binary" ]; then
  echo "Binary not found: $binary" >&2
  exit 1
fi

script_dir=$(cd -- "$(dirname -- "$0")" && pwd)
remote_dir=$(ssh -- "$host" 'mktemp -d /tmp/maco-deploy.XXXXXX')
if [[ ! "$remote_dir" =~ ^/tmp/maco-deploy\.[a-zA-Z0-9]+$ ]]; then
  echo "Unexpected remote staging directory: $remote_dir" >&2
  exit 1
fi

printf 'Uploading to %s:%s\n' "$host" "$remote_dir"
scp -- "$binary" "$host:$remote_dir/maco"
scp -- "$script_dir/deploy-install.sh" "$host:$remote_dir/install.sh"
printf -v remote_command '%q ' /bin/bash "$remote_dir/install.sh" "$@"
if ! ssh -t -- "$host" "$remote_command"; then
  printf 'Deployment failed; staged files remain at %s:%s\n' "$host" "$remote_dir" >&2
  exit 1
fi

ssh -- "$host" "rm -rf -- '$remote_dir'"
printf 'Maco installed on %s.\n' "$host"
