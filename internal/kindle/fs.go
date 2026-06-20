package kindle

import (
	"fmt"
	"log"
	"os/exec"
)

// Remount runs `mntroot <mode>` to remount the rootfs (rw or ro).
// Kindle ships with / mounted read-only; toggle to rw to edit anything
// under /usr or /opt, then back to ro to leave the device as found.
func Remount(mode string) error {
	if err := exec.Command("mntroot", mode).Run(); err != nil {
		return fmt.Errorf("mntroot %s: %w", mode, err)
	}
	return nil
}

// WithRW remounts root rw for the duration of fn and restores ro on the way
// out. The ro restore is best-effort and logged on failure — leaving the
// device with rw root is undesirable but shouldn't fail the actual write.
func WithRW(fn func() error) error {
	if err := Remount("rw"); err != nil {
		return err
	}
	defer func() {
		if err := Remount("ro"); err != nil {
			log.Printf("mntroot ro failed: %v", err)
		}
	}()
	return fn()
}
