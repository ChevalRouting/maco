# Build the ARM64 UEFI firmware with the maco splash compiled into LogoDxe,
# rather than relying on the BIOS-only QEMU -boot splash options. The final
# stage exports only the distributable assets.
FROM ubuntu:24.04 AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
        acpica-tools \
        build-essential \
        bzip2 \
        ca-certificates \
        gcc-aarch64-linux-gnu \
        git \
        nasm \
        python-is-python3 \
        python3 \
        python3-cairosvg \
        python3-pil \
        uuid-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /edk2
RUN git clone --depth 1 --branch edk2-stable202411 https://github.com/tianocore/edk2.git . \
    && git submodule update --init --depth 1
RUN make -C BaseTools -j4

COPY tools/firmware/strip-network.sh /strip-network.sh
RUN /strip-network.sh

COPY web/public/logo.svg /logo.svg
COPY tools/firmware/logo.py /logo.py
RUN python3 /logo.py /logo.svg MdeModulePkg/Logo/Logo.bmp

ENV GCC5_AARCH64_PREFIX=aarch64-linux-gnu-
SHELL ["/bin/bash", "-c"]
RUN source edksetup.sh && build -a AARCH64 -t GCC5 -b RELEASE \
        -p ArmVirtPkg/ArmVirtQemu.dsc -n 4 \
        -D NETWORK_ENABLE=FALSE \
        -D TPM2_ENABLE=FALSE \
        -D TPM2_CONFIG_ENABLE=FALSE \
        -D SECURE_BOOT_ENABLE=FALSE \
        -D DEBUG_PRINT_ERROR_LEVEL=0x80000000 \
        --pcd gEfiMdePkgTokenSpaceGuid.PcdPlatformBootTimeOut=0 \
        --pcd PcdDxeNxMemoryProtectionPolicy=0xC000000000007FD1 \
        --pcd PcdUninstallMemAttrProtocol=TRUE

RUN mkdir /out \
    && cp Build/ArmVirtQemu-AARCH64/RELEASE_GCC5/FV/QEMU_EFI.fd /out/maco-aarch64-code.fd \
    && truncate -s 64M /out/maco-aarch64-code.fd \
    && bzip2 -9 /out/maco-aarch64-code.fd \
    && cp License.txt /out/LICENSE.edk2.txt \
    && cp MdeModulePkg/Logo/Logo.bmp /out/logo.bmp \
    && git rev-parse HEAD > /out/edk2-revision.txt \
    && find . -name 'LICENSE*' -o -name 'COPYING*' | sort | while read path; do \
           echo "===== $path ====="; cat "$path"; \
       done > /out/THIRD-PARTY-NOTICES.txt

FROM scratch
COPY --from=build /out/ /
