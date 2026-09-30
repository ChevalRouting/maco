#include <arpa/inet.h>
#include <errno.h>
#include <fcntl.h>
#include <grp.h>
#include <net/bpf.h>
#include <net/if.h>
#include <poll.h>
#include <signal.h>
#include <sys/event.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/uio.h>
#include <sys/un.h>
#include <unistd.h>

#define MACO_BIOCSBATCHWRITE _IOW('B', 143, int)
#define MACO_BPF_BUFFER (1024 * 1024)
#define MACO_MAX_FRAME 16384
#define MACO_BATCH_SIZE 65536
#define MACO_VECTOR_FRAMES 64

typedef struct {
    int fd;
    unsigned int size;
    unsigned char *buffer;
} Port;

typedef struct {
    unsigned char stream[MACO_BATCH_SIZE];
    unsigned char injection[MACO_BATCH_SIZE];
    size_t used;
    int batch_enabled;
    unsigned long injected;
    unsigned long captured;
    unsigned long socket_reads;
    unsigned long socket_writes;
    unsigned long bpf_reads;
    unsigned long bpf_writes;
    unsigned long polls;
} Pump;

typedef struct {
    const char *interface;
    const char *socket_path;
    uid_t uid;
    gid_t gid;
    const char *ready_path;
} Args;

__attribute__((noreturn)) static void fail(const char *operation) {
    perror(operation);
    exit(1);
}

static int open_any_bpf(void) {
    char path[64];
    for (int unit = 0; unit < 256; unit++) {
        snprintf(path, sizeof(path), "/dev/bpf%d", unit);
        int fd = open(path, O_RDWR | O_NONBLOCK);
        if (fd >= 0) return fd;
    }
    fail("open BPF");
}

static Port port_open(const char *interface) {
    Port port = {.fd = open_any_bpf()};

    unsigned int requested = MACO_BPF_BUFFER;
    if (ioctl(port.fd, BIOCSBLEN, &requested)) fail("BIOCSBLEN");

    struct ifreq request = {0};
    if (strlen(interface) >= sizeof(request.ifr_name)) {
        fprintf(stderr, "interface name too long: %s\n", interface);
        exit(1);
    }
    strlcpy(request.ifr_name, interface, sizeof(request.ifr_name));
    if (ioctl(port.fd, BIOCSETIF, &request)) fail("BIOCSETIF");

    unsigned int dlt = 0;
    if (ioctl(port.fd, BIOCGDLT, &dlt)) fail("BIOCGDLT");
    if (dlt != DLT_EN10MB) {
        fprintf(stderr, "%s: expected Ethernet DLT=%u, got %u\n", interface, DLT_EN10MB, dlt);
        exit(1);
    }

    unsigned int enabled = 1, disabled = 0;
    if (ioctl(port.fd, BIOCIMMEDIATE, &enabled)) fail("BIOCIMMEDIATE");
    if (ioctl(port.fd, BIOCSHDRCMPLT, &enabled)) fail("BIOCSHDRCMPLT");
    if (ioctl(port.fd, BIOCSSEESENT, &disabled)) fail("BIOCSSEESENT");
    if (ioctl(port.fd, BIOCPROMISC, NULL)) fail("BIOCPROMISC");
    if (ioctl(port.fd, BIOCGBLEN, &port.size)) fail("BIOCGBLEN");

    port.buffer = malloc(port.size);
    if (!port.buffer) fail("malloc");

    printf("BPF ready: interface=%s buffer=%u\n", interface, port.size);
    fflush(stdout);
    return port;
}

static int port_enable_batch(const Port *port) {
    int enabled = 1;
    if (ioctl(port->fd, MACO_BIOCSBATCHWRITE, &enabled) == 0) return 1;
    if (errno != ENOTTY && errno != EINVAL && errno != ENOTSUP) fail("BIOCSBATCHWRITE");
    return 0;
}

static int writev_all(int fd, struct iovec *vectors, int count, Pump *pump) {
    int first = 0;
    while (first < count) {
        ssize_t written = writev(fd, vectors + first, count - first);
        pump->socket_writes++;
        if (written < 0 && errno == EINTR) continue;
        if (written <= 0) return -1;
        size_t remaining = (size_t)written;
        while (first < count && remaining >= vectors[first].iov_len) {
            remaining -= vectors[first].iov_len;
            first++;
        }
        if (first < count && remaining) {
            vectors[first].iov_base = (unsigned char *)vectors[first].iov_base + remaining;
            vectors[first].iov_len -= remaining;
        }
    }
    return 0;
}

static void bpf_inject(Port *port, Pump *pump, size_t length, unsigned int packets) {
    if (!length) return;

    if (pump->batch_enabled) {
        ssize_t written;
        do {
            written = write(port->fd, pump->injection, length);
            pump->bpf_writes++;
        } while (written < 0 && errno == EINTR);
        if (written != (ssize_t)length) fail("batch inject Ethernet");
    } else {
        size_t offset = 0;
        while (offset < length) {
            struct bpf_hdr header;
            memcpy(&header, pump->injection + offset, sizeof(header));
            ssize_t written;
            do {
                written = write(port->fd, pump->injection + offset + header.bh_hdrlen, header.bh_caplen);
                pump->bpf_writes++;
            } while (written < 0 && errno == EINTR);
            if (written != (ssize_t)header.bh_caplen) fail("inject Ethernet");
            offset += BPF_WORDALIGN(header.bh_hdrlen + header.bh_caplen);
        }
    }

    pump->injected += packets;
}

static int pump_stream_to_bpf(Port *port, int client, Pump *pump) {
    ssize_t count = read(client, pump->stream + pump->used, sizeof(pump->stream) - pump->used);
    pump->socket_reads++;
    if (count < 0 && errno == EINTR) return 0;
    if (count < 0) fail("read QEMU stream");
    if (!count) {
        if (pump->used) {
            fprintf(stderr, "QEMU disconnected in the middle of a frame\n");
            return -1;
        }
        return 1;
    }

    pump->used += (size_t)count;

    size_t offset = 0, batch = 0;
    unsigned int packets = 0;
    while (pump->used - offset >= 4) {
        uint32_t network_length;
        memcpy(&network_length, pump->stream + offset, 4);
        size_t length = ntohl(network_length);
        if (length < 14 || length > MACO_MAX_FRAME) {
            fprintf(stderr, "invalid QEMU frame length: %zu\n", length);
            return -1;
        }
        if (pump->used - offset < length + 4) break;

        size_t record = BPF_WORDALIGN(sizeof(struct bpf_hdr) + length);
        if (batch + record > sizeof(pump->injection)) {
            bpf_inject(port, pump, batch, packets);
            batch = 0;
            packets = 0;
        }

        struct bpf_hdr header = {
            .bh_hdrlen = sizeof(struct bpf_hdr),
            .bh_caplen = (bpf_u_int32)length,
            .bh_datalen = (bpf_u_int32)length,
        };
        memcpy(pump->injection + batch, &header, sizeof(header));
        memcpy(pump->injection + batch + sizeof(header), pump->stream + offset + 4, length);
        memset(pump->injection + batch + sizeof(header) + length, 0, record - sizeof(header) - length);
        batch += record;
        packets++;
        offset += length + 4;
    }

    bpf_inject(port, pump, batch, packets);
    pump->used -= offset;
    if (pump->used) memmove(pump->stream, pump->stream + offset, pump->used);
    return 0;
}

static int pump_bpf_to_stream(Port *port, int client, Pump *pump) {
    ssize_t count = read(port->fd, port->buffer, port->size);
    pump->bpf_reads++;
    if (count < 0 && (errno == EINTR || errno == EAGAIN)) return 0;
    if (count <= 0) fail("capture Ethernet");

    struct iovec vectors[MACO_VECTOR_FRAMES * 2];
    uint32_t lengths[MACO_VECTOR_FRAMES];
    unsigned int frames = 0;
    size_t offset = 0;
    size_t minimum = offsetof(struct bpf_hdr, bh_hdrlen) + sizeof(unsigned short);

    while (offset < (size_t)count) {
        if ((size_t)count - offset < minimum) return -1;

        struct bpf_hdr header = {0};
        memcpy(&header, port->buffer + offset, minimum);
        size_t record = (size_t)header.bh_hdrlen + header.bh_caplen;
        if (header.bh_hdrlen < minimum || record > (size_t)count - offset ||
            header.bh_caplen != header.bh_datalen || header.bh_caplen < 14 || header.bh_caplen > MACO_MAX_FRAME) {
            fprintf(stderr, "invalid or truncated Ethernet BPF record\n");
            return -1;
        }

        lengths[frames] = htonl(header.bh_caplen);
        vectors[frames * 2].iov_base = &lengths[frames];
        vectors[frames * 2].iov_len = 4;
        vectors[frames * 2 + 1].iov_base = port->buffer + offset + header.bh_hdrlen;
        vectors[frames * 2 + 1].iov_len = header.bh_caplen;
        frames++;
        offset += BPF_WORDALIGN(record);

        if (frames == MACO_VECTOR_FRAMES || offset >= (size_t)count) {
            if (writev_all(client, vectors, (int)(frames * 2), pump)) return -1;
            pump->captured += frames;
            frames = 0;
        }
    }

    return 0;
}

static Args parse_args(int argc, char **argv) {
    if (argc != 6) {
        fprintf(stderr, "usage: %s interface socket uid gid ready\n", argv[0]);
        exit(2);
    }

    return (Args){
        .interface = argv[1],
        .socket_path = argv[2],
        .uid = (uid_t)strtoul(argv[3], NULL, 10),
        .gid = (gid_t)strtoul(argv[4], NULL, 10),
        .ready_path = argv[5],
    };
}

static int listen_socket(const char *path, uid_t uid, gid_t gid) {
    int server = socket(AF_UNIX, SOCK_STREAM, 0);
    if (server < 0) fail("socket");

    struct sockaddr_un address = {.sun_family = AF_UNIX};
    if (strlen(path) >= sizeof(address.sun_path)) {
        fprintf(stderr, "socket path too long: %s\n", path);
        exit(2);
    }
    strlcpy(address.sun_path, path, sizeof(address.sun_path));

    umask(077);
    if (bind(server, (struct sockaddr *)&address, sizeof(address))) fail("bind");
    if (chown(path, uid, gid)) fail("chown socket");
    if (listen(server, 1)) fail("listen");
    return server;
}

static void drop_privileges(uid_t uid, gid_t gid) {
    if (setgroups(0, NULL) || setgid(gid) || setuid(uid)) fail("drop privileges");
}

static void write_marker(const char *path) {
    int marker = open(path, O_WRONLY | O_CREAT | O_EXCL, 0600);
    if (marker < 0) fail("create readiness marker");
    close(marker);
}

static int accept_client(int server) {
    struct pollfd listener = {.fd = server, .events = POLLIN};
    if (poll(&listener, 1, 60000) <= 0) return -1;

    int client = accept(server, NULL, NULL);
    if (client < 0) fail("accept");
    return client;
}

static int service(Port *port, int client, Pump *pump, int client_readable, int bpf_readable, int failed) {
    if (client_readable) {
        int result = pump_stream_to_bpf(port, client, pump);
        if (result) return result < 0 ? 1 : 0;
    }
    if (bpf_readable) {
        if (pump_bpf_to_stream(port, client, pump)) return 1;
    }
    if (failed) return 1;
    return -1;
}

static int run_loop_poll(Port *port, int client, Pump *pump) {
    for (;;) {
        struct pollfd fds[2] = {
            {.fd = client, .events = POLLIN},
            {.fd = port->fd, .events = POLLIN},
        };

        int ready = poll(fds, 2, -1);
        pump->polls++;
        if (ready < 0 && errno == EINTR) continue;
        if (ready < 0) fail("poll helper");

        int status = service(port, client, pump,
                             fds[0].revents & (POLLIN | POLLHUP),
                             fds[1].revents & POLLIN,
                             (fds[0].revents | fds[1].revents) & (POLLERR | POLLNVAL));
        if (status >= 0) return status;
    }
}

static int run_loop_kqueue(Port *port, int client, Pump *pump) {
    int kq = kqueue();
    if (kq < 0) fail("kqueue");

    struct kevent changes[2];
    EV_SET(&changes[0], client, EVFILT_READ, EV_ADD, 0, 0, NULL);
    EV_SET(&changes[1], port->fd, EVFILT_READ, EV_ADD, 0, 0, NULL);
    if (kevent(kq, changes, 2, NULL, 0, NULL) < 0) fail("kevent register");

    for (;;) {
        struct kevent events[2];
        int ready = kevent(kq, NULL, 0, events, 2, NULL);
        pump->polls++;
        if (ready < 0 && errno == EINTR) continue;
        if (ready < 0) fail("kevent wait");

        int client_readable = 0, bpf_readable = 0, failed = 0;
        for (int i = 0; i < ready; i++) {
            if (events[i].flags & EV_ERROR) failed = 1;
            if ((int)events[i].ident == client) client_readable = 1;
            else if ((int)events[i].ident == port->fd) bpf_readable = 1;
        }

        int status = service(port, client, pump, client_readable, bpf_readable, failed);
        if (status >= 0) {
            close(kq);
            return status;
        }
    }
}

static int run_loop(Port *port, int client, Pump *pump) {
    const char *choice = getenv("MACO_L2_EVENTS");
    int use_kqueue = choice && strcmp(choice, "kqueue") == 0;
    printf("event backend: %s\n", use_kqueue ? "kqueue" : "poll");
    fflush(stdout);
    return use_kqueue ? run_loop_kqueue(port, client, pump) : run_loop_poll(port, client, pump);
}

static void print_stats(Port *port, const Pump *pump) {
    printf("Disconnected: injected=%lu captured=%lu\n", pump->injected, pump->captured);
    printf("I/O calls: polls=%lu socket_reads=%lu socket_writes=%lu bpf_reads=%lu bpf_writes=%lu\n",
           pump->polls, pump->socket_reads, pump->socket_writes, pump->bpf_reads, pump->bpf_writes);

    struct bpf_stat stats = {0};
    if (ioctl(port->fd, BIOCGSTATS, &stats)) fail("BIOCGSTATS");
    printf("BPF stats: received=%u dropped=%u\n", stats.bs_recv, stats.bs_drop);
}

int main(int argc, char **argv) {
    signal(SIGPIPE, SIG_IGN);
    Args args = parse_args(argc, argv);

    Port port = port_open(args.interface);
    Pump pump = {.batch_enabled = port_enable_batch(&port)};
    printf("BPF batch injection: %s\n", pump.batch_enabled ? "enabled" : "unavailable, per-frame fallback");

    int server = listen_socket(args.socket_path, args.uid, args.gid);
    printf("READY %s\n", args.socket_path);
    fflush(stdout);

    drop_privileges(args.uid, args.gid);
    write_marker(args.ready_path);

    int client = accept_client(server);
    if (client < 0) return 1;
    close(server);

    struct timeval timeout = {.tv_sec = 5};
    if (setsockopt(client, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout))) fail("socket timeout");

    int status = run_loop(&port, client, &pump);

    print_stats(&port, &pump);
    close(client);
    close(port.fd);
    free(port.buffer);
    unlink(args.socket_path);
    return status;
}
