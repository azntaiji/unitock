package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// LockFile is the single-instance lock's name inside the data directory (D-12).
const LockFile = "unitock.lock"

// errLocked means another instance holds the data directory's lock (R-09).
var errLocked = errors.New("another unitock is already running with this data directory")

// acquireLock takes an exclusive advisory flock on the data directory's lock file and
// holds it until release is called. The OS releases it if the process dies (D-12).
func acquireLock(dataDir string) (release func(), err error) {
	path := filepath.Join(dataDir, LockFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%w (lock held on %s)", errLocked, path)
		}
		return nil, fmt.Errorf("lock file %s: %w", path, err)
	}
	return func() { _ = f.Close() }, nil
}
