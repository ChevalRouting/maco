//go:build cgo

package metrics

/*
#include <mach/mach.h>
#include <mach/mach_host.h>
#include <mach/mach_time.h>
#include <libproc.h>
#include <sys/resource.h>
#include <sys/sysctl.h>
#include <stdlib.h>
#include <net/route.h>
#include <net/if.h>
#include <net/if_mib.h>
#include <net/if_dl.h>

static int host_sample(uint64_t *busy, uint64_t *total, uint64_t *used, uint64_t *memory) {
    mach_port_t host = mach_host_self();
    host_cpu_load_info_data_t cpu;
    mach_msg_type_number_t count = HOST_CPU_LOAD_INFO_COUNT;
    kern_return_t result = host_statistics(host, HOST_CPU_LOAD_INFO, (host_info_t)&cpu, &count);
    if (result != KERN_SUCCESS) { mach_port_deallocate(mach_task_self(), host); return -1; }
    *busy = (uint64_t)cpu.cpu_ticks[CPU_STATE_USER] + cpu.cpu_ticks[CPU_STATE_SYSTEM] + cpu.cpu_ticks[CPU_STATE_NICE];
    *total = *busy + cpu.cpu_ticks[CPU_STATE_IDLE];
    vm_statistics64_data_t vm;
    count = HOST_VM_INFO64_COUNT;
    result = host_statistics64(host, HOST_VM_INFO64, (host_info64_t)&vm, &count);
    mach_port_deallocate(mach_task_self(), host);
    if (result != KERN_SUCCESS) return -1;
    size_t size = sizeof(*memory);
    if (sysctlbyname("hw.memsize", memory, &size, NULL, 0) != 0) return -1;
    *used = ((uint64_t)vm.active_count + vm.wire_count + vm.compressor_page_count) * vm_page_size;
    return 0;
}
static void interface_sample(struct ifmibdata *data, uint64_t *rx, uint64_t *tx) {
    *rx = data->ifmd_data.ifi_ibytes;
    *tx = data->ifmd_data.ifi_obytes;
}
static int process_sample(int pid, uint64_t *cpu, uint64_t *memory, uint64_t *identity) {
    struct rusage_info_v2 info;
    if (proc_pid_rusage(pid, RUSAGE_INFO_V2, (rusage_info_t *)&info) != 0) return -1;
    static mach_timebase_info_data_t timebase;
    if (timebase.denom == 0 && mach_timebase_info(&timebase) != KERN_SUCCESS) return -1;
    *cpu = (info.ri_user_time + info.ri_system_time) * timebase.numer / timebase.denom;
    *memory = info.ri_resident_size;
    *identity = info.ri_proc_start_abstime;
    return 0;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func ReadHost() (Counters, error) {
	var busy, total, used, memory C.uint64_t
	if C.host_sample(&busy, &total, &used, &memory) != 0 {
		return Counters{}, fmt.Errorf("read Mach host statistics")
	}

	return Counters{Busy: uint64(busy), Total: uint64(total), Memory: uint64(used), Capacity: uint64(memory)}, nil
}

func ReadProcess(pid int) (Counters, error) {
	var cpu, memory, identity C.uint64_t
	if C.process_sample(C.int(pid), &cpu, &memory, &identity) != 0 {
		return Counters{}, fmt.Errorf("read process statistics for PID %d", pid)
	}

	return Counters{Busy: uint64(cpu), Memory: uint64(memory), Identity: uint64(identity)}, nil
}

func ReadNetwork() (map[string]Counters, error) {
	mib := [6]C.int{C.CTL_NET, C.PF_LINK, C.NETLINK_GENERIC, C.IFMIB_IFALLDATA, 0, C.IFDATA_GENERAL}
	var size C.size_t
	if C.sysctl(&mib[0], 6, nil, &size, nil, 0) != 0 {
		return nil, fmt.Errorf("size interface statistics")
	}

	buffer := C.malloc(size)
	if buffer == nil {
		return nil, fmt.Errorf("allocate interface statistics")
	}

	defer C.free(buffer)
	if C.sysctl(&mib[0], 6, buffer, &size, nil, 0) != 0 {
		return nil, fmt.Errorf("read interface statistics")
	}

	result := make(map[string]Counters)
	if uintptr(size)%C.sizeof_struct_ifmibdata != 0 {
		return nil, fmt.Errorf("invalid interface statistics size")
	}

	for offset := uintptr(0); offset < uintptr(size); offset += C.sizeof_struct_ifmibdata {
		entry := (*C.struct_ifmibdata)(unsafe.Add(buffer, offset))
		if entry.ifmd_flags&C.IFF_LOOPBACK != 0 {
			continue
		}

		var rx, tx C.uint64_t
		C.interface_sample(entry, &rx, &tx)
		result[C.GoString(&entry.ifmd_name[0])] = Counters{RX: uint64(rx), TX: uint64(tx)}
	}

	return result, nil
}
