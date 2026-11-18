#!/bin/bash
set -euo pipefail

DSC=ArmVirtPkg/ArmVirtQemu.dsc
INC=ArmVirtPkg/ArmVirt.dsc.inc
FDF=ArmVirtPkg/ArmVirtQemuFvMain.fdf.inc

for marker in \
    "$DSC:NetworkPkg/NetworkComponents.dsc.inc" \
    "$DSC:NetworkPkg/UefiPxeBcDxe/UefiPxeBcDxe.inf" \
    "$DSC:OvmfPkg/VirtioNetDxe/VirtioNet.inf" \
    "$INC:TftpDynamicCommand/TftpDynamicCommand.inf" \
    "$INC:HttpDynamicCommand/HttpDynamicCommand.inf" \
    "$FDF:OvmfPkg/VirtioNetDxe/VirtioNet.inf" \
    "$FDF:NetworkPkg/Network.fdf.inc"; do
    file=${marker%%:*}
    pattern=${marker#*:}
    grep -qF "$pattern" "$file" || { echo "strip-network: expected marker missing, edk2 layout changed: $marker" >&2; exit 1; }
done

sed -i \
    -e '\#OvmfPkg/VirtioNetDxe/VirtioNet.inf#d' \
    -e '\#NetworkPkg/NetworkComponents.dsc.inc#d' \
    -e '\#NetworkPkg/UefiPxeBcDxe/UefiPxeBcDxe.inf {#,/^  }/d' \
    "$DSC"

sed -i \
    -e '\#TftpDynamicCommand/TftpDynamicCommand.inf {#,/^  }/d' \
    -e '\#HttpDynamicCommand/HttpDynamicCommand.inf {#,/^  }/d' \
    -e '\#UefiShellNetwork1CommandsLib#d' \
    "$INC"

sed -i \
    -e '\#DynamicCommand/TftpDynamicCommand#d' \
    -e '\#DynamicCommand/HttpDynamicCommand#d' \
    -e '\#OvmfPkg/VirtioNetDxe/VirtioNet.inf#d' \
    -e '\#NetworkPkg/Network.fdf.inc#d' \
    "$FDF"

for leftover in \
    "$DSC:NetworkPkg/NetworkComponents.dsc.inc" \
    "$DSC:NetworkPkg/UefiPxeBcDxe/UefiPxeBcDxe.inf" \
    "$FDF:NetworkPkg/Network.fdf.inc"; do
    file=${leftover%%:*}
    pattern=${leftover#*:}
    grep -qF "$pattern" "$file" && { echo "strip-network: failed to remove $leftover" >&2; exit 1; }
done

echo "strip-network: removed UEFI network stack from ArmVirtQemu"
