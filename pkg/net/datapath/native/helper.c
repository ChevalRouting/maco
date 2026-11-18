#include <arpa/inet.h>
#include <errno.h>
#include <fcntl.h>
#include <grp.h>
#include <net/bpf.h>
#include <net/if.h>
#include <poll.h>
#include <signal.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <unistd.h>

#define MACO_BIOCSBATCHWRITE _IOW('B', 143, int)
#define MACO_BPF_BUFFER (1024 * 1024)
#define MACO_SOCKET_BUFFER (1024 * 1024)
#define MACO_MAX_FRAME 16384
#define MACO_BATCH_SIZE 65536
#define MACO_PORTS 64
#define MACO_PORT_INPUT (256 * 1024)
#define MACO_PORT_OUTPUT (1024 * 1024)
#define MACO_MACS 1024
#define MACO_UPLINK (-1)
#define MACO_LISTENER (-2)

typedef struct {
    int fd;
    unsigned int size;
    unsigned char *buffer;
    int batch_enabled;
    unsigned char injection[MACO_BATCH_SIZE];
    size_t batch;
    unsigned int batch_frames;
    unsigned long injected;
    unsigned long captured;
    unsigned long dropped;
} Uplink;

typedef struct {
    int fd;
    unsigned char *input;
    size_t input_used;
    unsigned char *output;
    size_t output_used;
    unsigned long received;
    unsigned long dropped;
} SwitchPort;

typedef struct {
    unsigned char mac[6];
    int port;
} Station;

typedef struct {
    Uplink uplink;
    SwitchPort ports[MACO_PORTS];
    Station stations[MACO_MACS];
    int station_count;
} Switch;

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

static void uplink_open(Uplink *uplink, const char *interface) {
    uplink->fd = open_any_bpf();

    unsigned int requested = MACO_BPF_BUFFER;
    if (ioctl(uplink->fd, BIOCSBLEN, &requested)) fail("BIOCSBLEN");

    struct ifreq request = {0};
    if (strlen(interface) >= sizeof(request.ifr_name)) {
        fprintf(stderr, "interface name too long: %s\n", interface);
        exit(1);
    }
    strlcpy(request.ifr_name, interface, sizeof(request.ifr_name));
    if (ioctl(uplink->fd, BIOCSETIF, &request)) fail("BIOCSETIF");

    unsigned int dlt = 0;
    if (ioctl(uplink->fd, BIOCGDLT, &dlt)) fail("BIOCGDLT");
    if (dlt != DLT_EN10MB) {
        fprintf(stderr, "%s: expected Ethernet DLT=%u, got %u\n", interface, DLT_EN10MB, dlt);
        exit(1);
    }

    unsigned int enabled = 1, disabled = 0;
    if (ioctl(uplink->fd, BIOCIMMEDIATE, &enabled)) fail("BIOCIMMEDIATE");
    if (ioctl(uplink->fd, BIOCSHDRCMPLT, &enabled)) fail("BIOCSHDRCMPLT");
    if (ioctl(uplink->fd, BIOCSSEESENT, &disabled)) fail("BIOCSSEESENT");
    if (ioctl(uplink->fd, BIOCPROMISC, NULL)) fail("BIOCPROMISC");
    if (ioctl(uplink->fd, BIOCGBLEN, &uplink->size)) fail("BIOCGBLEN");

    uplink->buffer = malloc(uplink->size);
    if (!uplink->buffer) fail("malloc");

    int batch = 1;
    if (ioctl(uplink->fd, MACO_BIOCSBATCHWRITE, &batch) == 0) uplink->batch_enabled = 1;
    else if (errno != ENOTTY && errno != EINVAL && errno != ENOTSUP) fail("BIOCSBATCHWRITE");

    printf("BPF ready: interface=%s buffer=%u batch=%s\n", interface, uplink->size, uplink->batch_enabled ? "enabled" : "per-frame");
    fflush(stdout);
}

static void uplink_flush(Uplink *uplink) {
    if (!uplink->batch) return;

    if (uplink->batch_enabled) {
        ssize_t written;
        do {
            written = write(uplink->fd, uplink->injection, uplink->batch);
        } while (written < 0 && errno == EINTR);
        if (written != (ssize_t)uplink->batch) fail("batch inject Ethernet");
    } else {
        size_t offset = 0;
        while (offset < uplink->batch) {
            struct bpf_hdr header;
            memcpy(&header, uplink->injection + offset, sizeof(header));
            ssize_t written;
            do {
                written = write(uplink->fd, uplink->injection + offset + header.bh_hdrlen, header.bh_caplen);
            } while (written < 0 && errno == EINTR);
            if (written != (ssize_t)header.bh_caplen) fail("inject Ethernet");
            offset += BPF_WORDALIGN(header.bh_hdrlen + header.bh_caplen);
        }
    }

    uplink->injected += uplink->batch_frames;
    uplink->batch = 0;
    uplink->batch_frames = 0;
}

static void uplink_queue(Uplink *uplink, const unsigned char *frame, size_t length) {
    if (uplink->fd < 0) return;

    size_t record = BPF_WORDALIGN(sizeof(struct bpf_hdr) + length);
    if (record > sizeof(uplink->injection)) {
        uplink->dropped++;
        return;
    }
    if (uplink->batch + record > sizeof(uplink->injection)) uplink_flush(uplink);

    struct bpf_hdr header = {
        .bh_hdrlen = sizeof(struct bpf_hdr),
        .bh_caplen = (bpf_u_int32)length,
        .bh_datalen = (bpf_u_int32)length,
    };
    memcpy(uplink->injection + uplink->batch, &header, sizeof(header));
    memcpy(uplink->injection + uplink->batch + sizeof(header), frame, length);
    memset(uplink->injection + uplink->batch + sizeof(header) + length, 0, record - sizeof(header) - length);
    uplink->batch += record;
    uplink->batch_frames++;
}

static int station_find(const Switch *sw, const unsigned char *mac) {
    for (int i = 0; i < sw->station_count; i++) {
        if (!memcmp(sw->stations[i].mac, mac, 6)) return i;
    }
    return -1;
}

static void station_learn(Switch *sw, const unsigned char *mac, int port) {
    if (mac[0] & 1) return;
    int index = station_find(sw, mac);
    if (index >= 0) {
        sw->stations[index].port = port;
        return;
    }
    if (sw->station_count == MACO_MACS) return;
    memcpy(sw->stations[sw->station_count].mac, mac, 6);
    sw->stations[sw->station_count++].port = port;
}

static void station_forget(Switch *sw, int port) {
    int kept = 0;
    for (int i = 0; i < sw->station_count; i++) {
        if (sw->stations[i].port != port) sw->stations[kept++] = sw->stations[i];
    }
    sw->station_count = kept;
}

static void port_queue(SwitchPort *port, const unsigned char *frame, size_t length) {
    if (port->output_used + 4 + length > MACO_PORT_OUTPUT) {
        port->dropped++;
        return;
    }
    uint32_t network_length = htonl((uint32_t)length);
    memcpy(port->output + port->output_used, &network_length, 4);
    memcpy(port->output + port->output_used + 4, frame, length);
    port->output_used += 4 + length;
}

static void switch_forward(Switch *sw, int source, const unsigned char *frame, size_t length) {
    station_learn(sw, frame + 6, source);

    int index = frame[0] & 1 ? -1 : station_find(sw, frame);
    if (index >= 0) {
        int target = sw->stations[index].port;
        if (target == source) return;
        if (target == MACO_UPLINK) uplink_queue(&sw->uplink, frame, length);
        else port_queue(&sw->ports[target], frame, length);
        return;
    }

    for (int i = 0; i < MACO_PORTS; i++) {
        if (i != source && sw->ports[i].fd >= 0) port_queue(&sw->ports[i], frame, length);
    }
    if (source != MACO_UPLINK) uplink_queue(&sw->uplink, frame, length);
}

static void switch_init(Switch *sw) {
    *sw = (Switch){.uplink = {.fd = -1}};
    for (int i = 0; i < MACO_PORTS; i++) sw->ports[i].fd = -1;
}

static int switch_attach(Switch *sw, int client) {
    int index = 0;
    while (index < MACO_PORTS && sw->ports[index].fd >= 0) index++;
    if (index == MACO_PORTS || fcntl(client, F_SETFL, O_NONBLOCK)) return -1;

    unsigned char *input = malloc(MACO_PORT_INPUT), *output = malloc(MACO_PORT_OUTPUT);
    if (!input || !output) fail("malloc");
    sw->ports[index] = (SwitchPort){.fd = client, .input = input, .output = output};
    return index;
}

static void switch_detach(Switch *sw, int index) {
    SwitchPort *port = &sw->ports[index];
    printf("port %d disconnected: received=%lu dropped=%lu\n", index, port->received, port->dropped);
    fflush(stdout);
    close(port->fd);
    free(port->input);
    free(port->output);
    sw->ports[index] = (SwitchPort){.fd = -1};
    station_forget(sw, index);
}

static void switch_accept(Switch *sw, int server) {
    int client = accept(server, NULL, NULL);
    if (client < 0) return;

    int buffer = MACO_SOCKET_BUFFER;
    if (setsockopt(client, SOL_SOCKET, SO_SNDBUF, &buffer, sizeof(buffer)) ||
        setsockopt(client, SOL_SOCKET, SO_RCVBUF, &buffer, sizeof(buffer))) {
        close(client);
        return;
    }

    int index = switch_attach(sw, client);
    if (index < 0) {
        close(client);
        return;
    }
    printf("port %d connected\n", index);
    fflush(stdout);
}

static int switch_receive(Switch *sw, int index) {
    SwitchPort *port = &sw->ports[index];
    ssize_t count = read(port->fd, port->input + port->input_used, MACO_PORT_INPUT - port->input_used);
    if (count < 0 && (errno == EINTR || errno == EAGAIN)) return 0;
    if (count <= 0) return -1;
    port->input_used += (size_t)count;

    size_t offset = 0;
    while (port->input_used - offset >= 4) {
        uint32_t network_length;
        memcpy(&network_length, port->input + offset, 4);
        size_t length = ntohl(network_length);
        if (length < 14 || length > MACO_MAX_FRAME) {
            fprintf(stderr, "port %d: invalid QEMU frame length %zu\n", index, length);
            return -1;
        }
        if (port->input_used - offset < length + 4) break;
        port->received++;
        switch_forward(sw, index, port->input + offset + 4, length);
        offset += length + 4;
    }

    port->input_used -= offset;
    if (port->input_used) memmove(port->input, port->input + offset, port->input_used);
    return 0;
}

static int switch_transmit(SwitchPort *port) {
    size_t sent = 0;
    while (sent < port->output_used) {
        ssize_t count = write(port->fd, port->output + sent, port->output_used - sent);
        if (count < 0 && errno == EINTR) continue;
        if (count < 0 && errno == EAGAIN) break;
        if (count <= 0) return -1;
        sent += (size_t)count;
    }

    port->output_used -= sent;
    if (port->output_used) memmove(port->output, port->output + sent, port->output_used);
    return 0;
}

static int switch_capture(Switch *sw) {
    Uplink *uplink = &sw->uplink;
    ssize_t count = read(uplink->fd, uplink->buffer, uplink->size);
    if (count < 0 && (errno == EINTR || errno == EAGAIN)) return 0;
    if (count <= 0) return -1;

    size_t offset = 0;
    size_t minimum = offsetof(struct bpf_hdr, bh_hdrlen) + sizeof(unsigned short);
    while (offset < (size_t)count) {
        if ((size_t)count - offset < minimum) return -1;

        struct bpf_hdr header = {0};
        memcpy(&header, uplink->buffer + offset, minimum);
        size_t record = (size_t)header.bh_hdrlen + header.bh_caplen;
        if (header.bh_hdrlen < minimum || record > (size_t)count - offset) {
            fprintf(stderr, "truncated Ethernet BPF record\n");
            return -1;
        }
        if (header.bh_caplen == header.bh_datalen && header.bh_caplen >= 14 && header.bh_caplen <= MACO_MAX_FRAME) {
            uplink->captured++;
            switch_forward(sw, MACO_UPLINK, uplink->buffer + offset + header.bh_hdrlen, header.bh_caplen);
        }
        offset += BPF_WORDALIGN(record);
    }

    return 0;
}

static int switch_step(Switch *sw, int server, int timeout) {
    struct pollfd fds[MACO_PORTS + 2];
    int owners[MACO_PORTS + 2];
    int count = 0;

    if (server >= 0) {
        fds[count] = (struct pollfd){.fd = server, .events = POLLIN};
        owners[count++] = MACO_LISTENER;
    }
    if (sw->uplink.fd >= 0) {
        fds[count] = (struct pollfd){.fd = sw->uplink.fd, .events = POLLIN};
        owners[count++] = MACO_UPLINK;
    }
    for (int i = 0; i < MACO_PORTS; i++) {
        if (sw->ports[i].fd < 0) continue;
        fds[count] = (struct pollfd){.fd = sw->ports[i].fd, .events = sw->ports[i].output_used ? POLLIN | POLLOUT : POLLIN};
        owners[count++] = i;
    }

    int ready = poll(fds, (nfds_t)count, timeout);
    if (ready < 0 && errno == EINTR) return 0;
    if (ready < 0) fail("poll switch");

    for (int i = 0; i < count; i++) {
        if (!fds[i].revents) continue;
        if (owners[i] == MACO_LISTENER) switch_accept(sw, server);
        else if (owners[i] == MACO_UPLINK) {
            if (fds[i].revents & (POLLERR | POLLNVAL) || switch_capture(sw)) fail("capture Ethernet");
        } else if (sw->ports[owners[i]].fd >= 0 && fds[i].revents & (POLLIN | POLLHUP | POLLERR | POLLNVAL)) {
            if (switch_receive(sw, owners[i])) switch_detach(sw, owners[i]);
        }
    }

    uplink_flush(&sw->uplink);
    for (int i = 0; i < MACO_PORTS; i++) {
        if (sw->ports[i].fd >= 0 && sw->ports[i].output_used && switch_transmit(&sw->ports[i])) switch_detach(sw, i);
    }
    return ready;
}

static Args parse_args(int argc, char **argv) {
    if (argc != 6) {
        fprintf(stderr, "usage: %s interface|- socket uid gid ready\n", argv[0]);
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
    if (listen(server, 16)) fail("listen");
    if (fcntl(server, F_SETFL, O_NONBLOCK)) fail("listener");
    return server;
}

static void drop_privileges(uid_t uid, gid_t gid) {
    if (geteuid() != 0) return;
    if (setgroups(0, NULL) || setgid(gid) || setuid(uid)) fail("drop privileges");
}

static void write_marker(const char *path) {
    int marker = open(path, O_WRONLY | O_CREAT | O_EXCL, 0600);
    if (marker < 0) fail("create readiness marker");
    close(marker);
}

int main(int argc, char **argv) {
    signal(SIGPIPE, SIG_IGN);
    Args args = parse_args(argc, argv);

    static Switch sw;
    switch_init(&sw);
    if (strcmp(args.interface, "-")) uplink_open(&sw.uplink, args.interface);

    int server = listen_socket(args.socket_path, args.uid, args.gid);
    printf("READY %s\n", args.socket_path);
    fflush(stdout);

    drop_privileges(args.uid, args.gid);
    write_marker(args.ready_path);

    for (;;) switch_step(&sw, server, -1);
}
