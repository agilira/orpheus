// security_access_unix.go: current-user access checks on Unix systems.
//
// Copyright (c) 2026 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

//go:build unix

package orpheus

import "syscall"

// access(2) mode bits, fixed by POSIX.
const (
	accessRead    = 0x4
	accessWrite   = 0x2
	accessExecute = 0x1
)

// checkCurrentUserAccess replaces the owner-bit guesses with what access(2)
// reports for the current user: group and other bits, root, ACLs and
// read-only mounts included.
//
// WHY the shortcuts: each access(2) is a system call. Read and write are
// asked together first, which settles the common readable-and-writable file
// in one call; a regular file with no execute bit at all is executable by
// nobody, root included, so that call is skipped.
func (fp *FilePermission) checkCurrentUserAccess() {
	if syscall.Access(fp.Path, accessRead|accessWrite) == nil {
		fp.IsReadable, fp.IsWritable = true, true
	} else {
		fp.IsReadable = syscall.Access(fp.Path, accessRead) == nil
		fp.IsWritable = syscall.Access(fp.Path, accessWrite) == nil
	}

	if !fp.IsDirectory && fp.Mode.Perm()&0o111 == 0 {
		fp.IsExecutable = false
		return
	}
	fp.IsExecutable = syscall.Access(fp.Path, accessExecute) == nil
}

// physicalWorkingDir reports that workingDir returns a path without links.
const physicalWorkingDir = true

// workingDir returns the kernel's working directory.
// WHY syscall.Getwd, not os.Getwd: os.Getwd may return $PWD, which can run
// through symlinks; the physical path is one system call and needs no
// symlink walk.
func workingDir() (string, error) {
	return syscall.Getwd()
}
