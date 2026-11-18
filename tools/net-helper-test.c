#include <assert.h>
#pragma push_macro("main")
#undef main
#define main maco_native_helper_main
#include "../pkg/net/datapath/native/helper.c"
#undef main
#pragma pop_macro("main")

static const unsigned char MAC_A[6] = {2, 0, 0, 0, 0, 0xa};
static const unsigned char MAC_B[6] = {2, 0, 0, 0, 0, 0xb};
static const unsigned char MAC_HOST[6] = {2, 0, 0, 0, 0, 0xc};
static const unsigned char BROADCAST[6] = {0xff, 0xff, 0xff, 0xff, 0xff, 0xff};

static size_t frame(unsigned char *out, const unsigned char *dst, const unsigned char *src, size_t length, unsigned char fill) {
    memset(out, fill, length);
    memcpy(out, dst, 6);
    memcpy(out + 6, src, 6);
    out[12] = 0x88;
    out[13] = 0xb5;
    return length;
}

static void send_stream(int fd, const unsigned char *data, size_t length) {
    uint32_t network_length = htonl((uint32_t)length);
    assert(write(fd, &network_length, 4) == 4);
    assert(write(fd, data, length) == (ssize_t)length);
}

static size_t drain(Switch *sw, int fd, unsigned char *out, size_t capacity) {
    size_t used = 0;
    for (int idle = 0; idle < 3;) {
        switch_step(sw, -1, 20);
        ssize_t count = recv(fd, out + used, capacity - used, MSG_DONTWAIT);
        if (count > 0) {
            used += (size_t)count;
            idle = 0;
        } else {
            idle++;
        }
    }
    return used;
}

static void expect_frames(const unsigned char *stream, size_t used, const unsigned char *expected, size_t length, int frames) {
    assert(used == (size_t)frames * (length + 4));
    for (int i = 0; i < frames; i++) {
        uint32_t network_length;
        memcpy(&network_length, stream + i * (length + 4), 4);
        assert(ntohl(network_length) == length);
        assert(!memcmp(stream + i * (length + 4) + 4, expected, length));
    }
}

static int injected_frames(const unsigned char *batch, size_t used, const unsigned char *expected, size_t length) {
    int frames = 0;
    for (size_t offset = 0; offset < used; frames++) {
        struct bpf_hdr header;
        memcpy(&header, batch + offset, sizeof(header));
        assert(header.bh_hdrlen == sizeof(struct bpf_hdr) && header.bh_caplen == length && header.bh_datalen == length);
        assert(!memcmp(batch + offset + header.bh_hdrlen, expected, length));
        offset += BPF_WORDALIGN(header.bh_hdrlen + header.bh_caplen);
    }
    return frames;
}

int main(void) {
    static Switch sw;
    static unsigned char out[4 * 1024 * 1024];
    unsigned char data[1514], other[1514];
    int a[2], b[2], u[2];
    assert(!socketpair(AF_UNIX, SOCK_STREAM, 0, a) && !socketpair(AF_UNIX, SOCK_STREAM, 0, b) && !socketpair(AF_UNIX, SOCK_STREAM, 0, u));

    switch_init(&sw);
    sw.uplink.fd = u[0];
    sw.uplink.size = 65536;
    sw.uplink.buffer = malloc(sw.uplink.size);
    sw.uplink.batch_enabled = 1;
    int port_a = switch_attach(&sw, a[0]), port_b = switch_attach(&sw, b[0]);
    assert(port_a == 0 && port_b == 1);

    size_t length = frame(data, BROADCAST, MAC_A, 128, 1);
    send_stream(a[1], data, length);
    expect_frames(out, drain(&sw, b[1], out, sizeof(out)), data, length, 1);
    assert(injected_frames(out, drain(&sw, u[1], out, sizeof(out)), data, length) == 1);
    assert(drain(&sw, a[1], out, sizeof(out)) == 0);

    length = frame(data, MAC_A, MAC_B, 200, 2);
    uint32_t network_length = htonl((uint32_t)length);
    assert(write(b[1], &network_length, 3) == 3);
    switch_step(&sw, -1, 20);
    assert(write(b[1], (unsigned char *)&network_length + 3, 1) == 1);
    assert(write(b[1], data, 7) == 7);
    switch_step(&sw, -1, 20);
    assert(write(b[1], data + 7, length - 7) == (ssize_t)(length - 7));
    expect_frames(out, drain(&sw, a[1], out, sizeof(out)), data, length, 1);
    assert(drain(&sw, u[1], out, sizeof(out)) == 0);

    length = frame(data, MAC_HOST, MAC_A, 300, 3);
    send_stream(a[1], data, length);
    expect_frames(out, drain(&sw, b[1], out, sizeof(out)), data, length, 1);
    assert(injected_frames(out, drain(&sw, u[1], out, sizeof(out)), data, length) == 1);

    length = frame(data, MAC_B, MAC_HOST, 400, 4);
    unsigned char capture[18 + 400 + 2];
    struct bpf_hdr header = {.bh_hdrlen = 18, .bh_caplen = (bpf_u_int32)length, .bh_datalen = (bpf_u_int32)length};
    memcpy(capture, &header, 18);
    memcpy(capture + 18, data, length);
    assert(write(u[1], capture, sizeof(capture)) == (ssize_t)sizeof(capture));
    expect_frames(out, drain(&sw, b[1], out, sizeof(out)), data, length, 1);
    assert(drain(&sw, a[1], out, sizeof(out)) == 0 && sw.uplink.captured == 1);

    length = frame(data, MAC_HOST, MAC_A, 500, 5);
    send_stream(a[1], data, length);
    assert(injected_frames(out, drain(&sw, u[1], out, sizeof(out)), data, length) == 1);
    assert(drain(&sw, b[1], out, sizeof(out)) == 0);

    int large = 1024 * 1024, small = 4096;
    assert(!setsockopt(a[1], SOL_SOCKET, SO_SNDBUF, &large, sizeof(large)));
    assert(!setsockopt(a[0], SOL_SOCKET, SO_RCVBUF, &large, sizeof(large)));
    assert(!setsockopt(b[0], SOL_SOCKET, SO_SNDBUF, &small, sizeof(small)));
    assert(!setsockopt(b[1], SOL_SOCKET, SO_RCVBUF, &small, sizeof(small)));
    length = frame(data, MAC_B, MAC_A, 1514, 6);
    for (int i = 0; i < 200; i++) send_stream(a[1], data, length);
    expect_frames(out, drain(&sw, b[1], out, sizeof(out)), data, length, 200);

    close(b[1]);
    for (int i = 0; i < 3 && sw.ports[port_b].fd >= 0; i++) switch_step(&sw, -1, 20);
    assert(sw.ports[port_b].fd < 0 && station_find(&sw, MAC_B) < 0 && station_find(&sw, MAC_A) >= 0);
    length = frame(other, MAC_B, MAC_A, 128, 7);
    send_stream(a[1], other, length);
    assert(injected_frames(out, drain(&sw, u[1], out, sizeof(out)), other, length) == 1);

    int raw[2];
    assert(!pipe(raw));
    Uplink fallback = {.fd = raw[1]};
    length = frame(data, MAC_HOST, MAC_A, 128, 8);
    uplink_queue(&fallback, data, length);
    uplink_queue(&fallback, data, length);
    uplink_flush(&fallback);
    assert(fallback.injected == 2 && read(raw[0], out, sizeof(out)) == (ssize_t)(2 * length));
    assert(!memcmp(out, data, length) && !memcmp(out + length, data, length));

    puts("PASS: flooding, MAC learning, unicast forwarding, uplink capture and batched injection, fragmented frames, partial writes, port disconnect, per-frame fallback");
    return 0;
}
