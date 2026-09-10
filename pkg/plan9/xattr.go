// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plan9

import (
	"path"
	"syscall"
	"unsafe"
)

func rawGetxattr(path, name string, dest []byte) (int, error) {
	p0, err := syscall.BytePtrFromString(path)
	if err != nil {
		return 0, err
	}
	p1, err := syscall.BytePtrFromString(name)
	if err != nil {
		return 0, err
	}
	var dPtr uintptr
	if len(dest) > 0 {
		dPtr = uintptr(unsafe.Pointer(&dest[0]))
	}
	r0, _, e1 := syscall.Syscall6(syscall.SYS_GETXATTR, uintptr(unsafe.Pointer(p0)), uintptr(unsafe.Pointer(p1)), dPtr, uintptr(len(dest)), 0, 0)
	if e1 != 0 {
		return int(r0), e1
	}
	return int(r0), nil
}

func rawSetxattr(path, name string, data []byte, flags int) error {
	p0, err := syscall.BytePtrFromString(path)
	if err != nil {
		return err
	}
	p1, err := syscall.BytePtrFromString(name)
	if err != nil {
		return err
	}
	var dPtr uintptr
	if len(data) > 0 {
		dPtr = uintptr(unsafe.Pointer(&data[0]))
	}
	_, _, e1 := syscall.Syscall6(syscall.SYS_SETXATTR, uintptr(unsafe.Pointer(p0)), uintptr(unsafe.Pointer(p1)), dPtr, uintptr(len(data)), uintptr(flags), 0)
	if e1 != 0 {
		return e1
	}
	return nil
}

func readdelattr(filepath string) (bool, error) {
	var val [1]byte
	n, err := rawGetxattr(filepath, "novm-deleted", val[:])
	if err != nil || n != 1 {
		var stat syscall.Stat_t
		err := syscall.Stat(
			path.Join(filepath, ".deleted"),
			&stat)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return val[0] == 1, nil
}

func setdelattr(filepath string) error {
	val := []byte{1}
	err := rawSetxattr(filepath, "novm-deleted", val, 0)
	if err != nil {
		fd, err := syscall.Open(
			path.Join(filepath, ".deleted"),
			syscall.O_RDWR|syscall.O_CREAT,
			syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR)
		if err == nil {
			syscall.Close(fd)
		}
		return err
	}
	return nil
}

func cleardelattr(filepath string) error {
	val := []byte{0}
	err := rawSetxattr(filepath, "novm-deleted", val, 0)
	if err != nil {
		return syscall.Unlink(path.Join(filepath, ".deleted"))
	}
	return nil
}
