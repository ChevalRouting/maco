//go:build linux

package vm

import (
	"fmt"
	"os"
	"strings"
)

func processIdentity(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", err
	}

	end := strings.LastIndex(string(data), ")")
	if end < 0 {
		return "", fmt.Errorf("invalid process stat")
	}

	fields := strings.Fields(string(data)[end+1:])
	if len(fields) < 20 {
		return "", fmt.Errorf("invalid process stat")
	}

	return fmt.Sprintf("%d:%s", pid, fields[19]), nil
}
