//go:build linux

package consumes

import "syscall"

// becomeChildSubreaper makes the calling process the subreaper of its descendants. 36 is
// PR_SET_CHILD_SUBREAPER, which the syscall package does not define on linux/amd64.
func becomeChildSubreaper() error {
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, 36, 1, 0); errno != 0 {
		return errno
	}
	return nil
}
