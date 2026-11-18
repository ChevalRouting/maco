import argparse
import os
import re
import socket
import subprocess
import sys
import tempfile
import time
from pathlib import Path

BUFFER = 1 << 20
OFFLOADS = 'csum=off,guest_csum=off,gso=off,guest_tso4=off,guest_tso6=off,guest_ecn=off'
TESTS = {
    '-t 10': 'one stream, a to b',
    '-t 10 -R': 'one stream, b to a',
    '-t 10 -P 4': 'four streams, a to b',
    '-t 10 -P 4 -R': 'four streams, b to a',
}


def arguments():
    parser = argparse.ArgumentParser(description='measure vm-to-vm throughput between two qemu guests')
    parser.add_argument('qemu', help='qemu-system-aarch64 to measure')
    parser.add_argument('--image', required=True, help='directory written by image.sh')
    parser.add_argument('--label', default='', help='name printed in front of each result')
    parser.add_argument('--mtu', type=int, default=1500)
    parser.add_argument('--switch', help='maco-net-helper to put between the guests instead of a direct socket pair')
    return parser.parse_args()


def tune(sock):
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_SNDBUF, BUFFER)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, BUFFER)
    return sock


def wait_for(path, process):
    deadline = time.monotonic() + 10
    while not path.exists():
        if process.poll() is not None or time.monotonic() > deadline:
            sys.exit(f'switch did not start: {path}')

        time.sleep(0.05)


def connect(path):
    sock = socket.socket(socket.AF_UNIX)
    sock.connect(str(path))
    return sock


def start_switch(helper, work):
    path = work / 'switch.sock'
    ready = work / 'switch.ready'
    process = subprocess.Popen([helper, '-', str(path), str(os.getuid()), str(os.getgid()), str(ready)], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    wait_for(ready, process)
    return process, connect(path), connect(path)


def guest(args, role, address, peer, sock, log):
    command = [
        args.qemu, '-machine', 'virt,gic-version=3', '-accel', 'hvf', '-cpu', 'host', '-smp', '2', '-m', '512',
        '-nodefaults', '-display', 'none', '-serial', f'file:{log}',
        '-kernel', str(args.image / 'vmlinuz-virt'), '-initrd', str(args.image / 'initramfs.gz'),
        '-append', f'console=ttyAMA0 quiet rdinit=/init bench.role={role} bench.ip={address} bench.peer={peer} bench.mtu={args.mtu}',
        '-netdev', f'stream,id=n0,server=off,addr.type=fd,addr.str={sock.fileno()}',
        '-device', f'virtio-net-pci,netdev=n0,mac=02:00:00:00:00:{1 if role == "server" else 2:02x},disable-legacy=on,{OFFLOADS}',
    ]
    return subprocess.Popen(command, pass_fds=[sock.fileno()], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def results(log):
    output = log.read_text(errors='replace')
    rows = []
    for block in output.split('BENCH RUN ')[1:]:
        test = block.splitlines()[0].strip()
        rates = re.findall(r'([\d.]+) Gbits/sec.*receiver', block)
        if not rates:
            sys.exit(f'benchmark failed:\n{output[-3000:]}')

        rows.append((TESTS.get(test, test), rates[-1]))

    if len(rows) != len(TESTS):
        sys.exit(f'benchmark failed:\n{output[-3000:]}')

    return rows


def run(args, work):
    switch = None
    if args.switch:
        switch, a, b = start_switch(args.switch, work)
    else:
        a, b = socket.socketpair(socket.AF_UNIX, socket.SOCK_STREAM)

    server = guest(args, 'server', '10.9.0.1', '10.9.0.2', tune(a), work / 'server.log')
    client = guest(args, 'client', '10.9.0.2', '10.9.0.1', tune(b), work / 'client.log')
    a.close()
    b.close()
    try:
        client.wait(timeout=180)
    finally:
        for process in (client, server, switch):
            if process:
                process.kill()
                process.wait()

    return results(work / 'client.log')


def main():
    args = arguments()
    args.image = Path(args.image).resolve()
    with tempfile.TemporaryDirectory(prefix='maco-bench-', dir='/tmp') as work:
        for test, rate in run(args, Path(work)):
            print(f'{args.label:24s} {test:22s} {rate:>6} Gbit/s')


if __name__ == '__main__':
    main()
