//go:build !linux

package consumes

import "errors"

// becomeChildSubreaper keeps the package compiling where there is no child subreaper; the only
// caller is skipped there.
func becomeChildSubreaper() error {
	return errors.New("child subreapers exist on Linux only")
}
