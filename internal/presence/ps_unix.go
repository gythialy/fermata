//go:build unix

package presence

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunningProcesses reports whether any of the needles appears in a running
// process's name or arguments.
func RunningProcesses(needles []string) (bool, error) {
	if len(needles) == 0 {
		return false, nil
	}
	out, err := exec.Command("ps", "-axo", "comm=,args=").Output()
	if err != nil {
		return false, fmt.Errorf("list processes: %w", err)
	}
	lowered := make([]string, 0, len(needles))
	for _, n := range needles {
		if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
			lowered = append(lowered, n)
		}
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.ToLower(line)
		for _, needle := range lowered {
			if strings.Contains(line, needle) {
				return true, nil
			}
		}
	}
	return false, nil
}
