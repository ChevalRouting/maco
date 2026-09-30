#!/usr/bin/env bash
set -euo pipefail

# macos-server-setup.sh
# Turn a Mac (mini) into a headless server: never sleep, restart after power
# loss, and stop the background daemons that steal CPU from your workloads.
#
# Usage:
#   sudo ./macos-server-setup.sh                 # power + de-thief (safe, no SIP)
#   sudo ./macos-server-setup.sh --gatekeeper    # also disable Gatekeeper
#   sudo ./macos-server-setup.sh --xprotect      # also disable XProtect scans (needs SIP off)
#   sudo ./macos-server-setup.sh --all           # everything
#   sudo ./macos-server-setup.sh --dry-run       # print actions only

WITH_GATEKEEPER=0
WITH_XPROTECT=0
DRY_RUN=0

for arg in "$@"; do
	case "$arg" in
		--gatekeeper) WITH_GATEKEEPER=1 ;;
		--xprotect)   WITH_XPROTECT=1 ;;
		--all)        WITH_GATEKEEPER=1; WITH_XPROTECT=1 ;;
		--dry-run)    DRY_RUN=1 ;;
		-h|--help)    grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
		*) echo "unknown flag: $arg" >&2; exit 2 ;;
	esac
done

if [[ $EUID -ne 0 && $DRY_RUN -eq 0 ]]; then
	echo "must run as root: sudo $0 $*" >&2
	exit 1
fi

CONSOLE_USER="$(stat -f%Su /dev/console)"
CONSOLE_UID="$(id -u "$CONSOLE_USER")"

log()  { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!!!\033[0m %s\n' "$*"; }
run()  {
	if [[ $DRY_RUN -eq 1 ]]; then
		printf '   [dry-run] %s\n' "$*"
	else
		printf '   %s\n' "$*"
		"$@" || warn "failed (continuing): $*"
	fi
}
# run a command in the console user's GUI launchd domain
urun() {
	if [[ $DRY_RUN -eq 1 ]]; then
		printf '   [dry-run] (as %s) %s\n' "$CONSOLE_USER" "$*"
	else
		printf '   (as %s) %s\n' "$CONSOLE_USER" "$*"
		launchctl asuser "$CONSOLE_UID" sudo -u "$CONSOLE_USER" "$@" || warn "failed (continuing): $*"
	fi
}

log "Console user: $CONSOLE_USER (uid $CONSOLE_UID)"

log "Tier 1: power management (never sleep, survive power loss)"
run pmset -a sleep 0
run pmset -a disksleep 0
run pmset -a displaysleep 0
run pmset -a powernap 0
run pmset -a hibernatemode 0
run pmset -a womp 1
run pmset -a autorestart 1
run pmset -a lowpowermode 0

log "Tier 2: stop the background CPU thieves (no SIP needed)"

log "  Spotlight indexing off (mds / mds_stores)"
run mdutil -a -i off

log "  Photo/media content analysis off"
for label in com.apple.photoanalysisd com.apple.mediaanalysisd com.apple.mediaanalysisd.fullanalysis; do
	run launchctl disable "gui/$CONSOLE_UID/$label"
	run launchctl bootout  "gui/$CONSOLE_UID/$label"
done

log "  Automatic software update checks/downloads off"
run softwareupdate --schedule off
run defaults write /Library/Preferences/com.apple.SoftwareUpdate AutomaticCheckEnabled -bool false
run defaults write /Library/Preferences/com.apple.SoftwareUpdate AutomaticDownload -bool false
run defaults write /Library/Preferences/com.apple.commerce AutoUpdate -bool false

log "  App Nap off (console user)"
urun defaults write NSGlobalDomain NSAppSleepDisabled -bool true

if [[ $WITH_GATEKEEPER -eq 1 ]]; then
	log "Tier 3: Gatekeeper disable (allow unsigned binaries)"
	run spctl --master-disable
else
	warn "Gatekeeper left enabled (pass --gatekeeper to disable)"
fi

SIP_STATUS="$(csrutil status 2>/dev/null || true)"
if [[ $WITH_XPROTECT -eq 1 ]]; then
	if echo "$SIP_STATUS" | grep -qi disabled; then
		log "Tier 3: XProtect / malware scan daemons off (SIP is disabled, OK)"
		for label in \
			com.apple.XProtect.daemon.scan \
			com.apple.XProtect.daemon.scan.startup \
			com.apple.XprotectFramework.PluginService \
			com.apple.MRTa; do
			run launchctl disable "system/$label"
			run launchctl bootout  "system/$label"
		done
	else
		warn "XProtect NOT disabled: SIP is enabled."
		warn "  Boot into Recovery, run 'csrutil disable', reboot, then re-run with --xprotect."
	fi
else
	warn "XProtect left enabled (pass --xprotect; requires SIP off)"
fi

log "Done."
echo
log "Verify:"
echo "   pmset -g custom | grep -E 'sleep|autorestart|womp'"
echo "   mdutil -a -s"
echo "   spctl --status"
echo "   csrutil status"
warn "Reboot (or log out/in) so the disabled user agents fully stop."
