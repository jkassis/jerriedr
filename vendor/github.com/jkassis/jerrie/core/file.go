package core

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

// FileGet returns a files as []byte or exits if not found
func FileGet(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		Log.Fatal(err)
	}
	return data
}

// DirSize get the size of the directory
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// RLimitNoFileUpgrade attempts to update the system resource limit for open files
func RLimitNoFileUpgrade() error {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return err
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		rLimit.Cur = 49152
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
			return err
		}
	}
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return err
	}

	Log.Warnf("RLimit is %d", rLimit.Cur)
	return nil
}

// RLimitNoFileGet attempts to return the current system resource limit for open files
func RLimitNoFileGet() int32 {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}
	return int32(rLimit.Cur)
}
