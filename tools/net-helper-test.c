#include <string.h>
#include <sys/uio.h>
#include <arpa/inet.h>
#include <assert.h>
static unsigned char output[2048];
static size_t output_used;
static ssize_t short_writev(int fd, const struct iovec *vectors, int count) {
    (void)fd;
    size_t copied = 0;
    for (int i = 0; i < count && copied < 7; i++) {
        size_t size = vectors[i].iov_len < 7 - copied ? vectors[i].iov_len : 7 - copied;
        memcpy(output + output_used, vectors[i].iov_base, size);
        output_used += size; copied += size;
    }
    return (ssize_t)copied;
}
#define writev short_writev
#pragma push_macro("main")
#undef main
#define main maco_native_helper_main
#include "../pkg/net/datapath/native/helper.c"
#undef main
#pragma pop_macro("main")
#undef writev
static void check_event_loop(const char *choice, const unsigned char *stream, const unsigned char *capture) {
    int client[2], bpf[2];
    assert(!socketpair(AF_UNIX, SOCK_STREAM, 0, client));
    assert(!socketpair(AF_UNIX, SOCK_STREAM, 0, bpf));
    assert(write(client[1], stream, 264) == 264);
    assert(write(bpf[1], capture, 296) == 296);
    assert(!shutdown(client[1], SHUT_WR));
    Port port = {.fd = bpf[0], .size = 1024, .buffer = malloc(1024)};
    Pump pump = {.batch_enabled = 1};
    output_used = 0;
    assert(!setenv("MACO_L2_EVENTS", choice, 1));
    assert(!run_loop(&port, client[0], &pump));
    assert(pump.injected == 2 && pump.captured == 2);
    assert(output_used == 264 && !memcmp(output, stream, 264));
    free(port.buffer);
    close(client[0]); close(client[1]); close(bpf[0]); close(bpf[1]);
}

int main(void) {
    int stream_pipe[2], injection_pipe[2];
    assert(!pipe(stream_pipe) && !pipe(injection_pipe));
    Port port = {.fd = injection_pipe[1]};
    Pump io = {.batch_enabled = 1};
    unsigned char frame[128];
    for (size_t i=0; i<sizeof(frame); i++) frame[i]=(unsigned char)i;
    unsigned char stream[264];
    uint32_t length=htonl(128);
    for (int offset=0; offset<264; offset+=132) {
        memcpy(stream+offset,&length,4); memcpy(stream+offset+4,frame,128);
    }
    assert(write(stream_pipe[1],stream,3)==3);
    assert(!pump_stream_to_bpf(&port,stream_pipe[0],&io));
    assert(io.used==3 && !io.injected);
    assert(write(stream_pipe[1],stream+3,261)==261);
    assert(!pump_stream_to_bpf(&port,stream_pipe[0],&io));
    assert(io.used==0 && io.injected==2 && io.bpf_writes==1);
    unsigned char batch[296];
    assert(read(injection_pipe[0],batch,sizeof(batch))==sizeof(batch));
    for (int offset=0; offset<296; offset+=148) {
        struct bpf_hdr header;
        memcpy(&header,batch+offset,sizeof(header));
        assert(header.bh_hdrlen==20 && header.bh_caplen==128 && header.bh_datalen==128);
        assert(!memcmp(batch+offset+20,frame,128));
    }
    close(injection_pipe[1]); close(injection_pipe[0]);
    int capture_pipe[2]; assert(!pipe(capture_pipe));
    for (int offset=0; offset<296; offset+=148) {
        struct bpf_hdr header={.bh_hdrlen=18,.bh_caplen=128,.bh_datalen=128};
        memcpy(batch+offset,&header,18); memcpy(batch+offset+18,frame,128);
    }
    assert(write(capture_pipe[1],batch,sizeof(batch))==sizeof(batch));
    port.fd=capture_pipe[0]; port.size=1024; port.buffer=malloc(1024);
    assert(!pump_bpf_to_stream(&port,99,&io));
    assert(io.captured==2 && output_used==264 && !memcmp(output,stream,264));
    assert(io.socket_writes>1);
    close(stream_pipe[1]);
    assert(pump_stream_to_bpf(&port,stream_pipe[0],&io)==1);
    int fallback_pipe[2]; assert(!pipe(fallback_pipe));
    port.fd=fallback_pipe[1]; io.batch_enabled=0;
    struct bpf_hdr fallback_header={.bh_hdrlen=20,.bh_caplen=128,.bh_datalen=128};
    memcpy(io.injection,&fallback_header,20); memcpy(io.injection+20,frame,128);
    bpf_inject(&port,&io,148,1);
    unsigned char received[128]; assert(read(fallback_pipe[0],received,128)==128);
    assert(!memcmp(received,frame,128));
    close(fallback_pipe[0]); close(fallback_pipe[1]);
    check_event_loop("poll", stream, batch);
    check_event_loop("kqueue", stream, batch);
    assert(!unsetenv("MACO_L2_EVENTS"));
    puts("PASS: poll/kqueue event loops, fragmented/coalesced QEMU frames, aligned BPF batch encoding, 18-byte capture headers, partial vectored writes, clean EOF");
    free(port.buffer);
    close(capture_pipe[0]);close(capture_pipe[1]);close(stream_pipe[0]);
    return 0;
}
