package vm

import (
	"fmt"
	"golang.org/x/sys/unix"
)

func processIdentity(pid int) (string, error) {
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d:%d:%d", pid, info.Proc.P_starttime.Sec, info.Proc.P_starttime.Usec), nil
}

func peerPID(fd int) (int, error) {
	return unix.GetsockoptInt(fd, unix.SOL_LOCAL, unix.LOCAL_PEERPID)
}
