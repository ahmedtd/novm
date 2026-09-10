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

package machine

import (
	"syscall"
	"unsafe"
)

//
// The virtIO buffer is a collection of different
// descriptor objects. It exposes simple header
// manipulation primitives as well as scatter-gather
// I/O operations for zero-copy efficiency.
//

type VirtioBuffer struct {
	data     [][]byte
	index    uint16
	length   int
	readonly bool
}

func NewVirtioBuffer(index uint16, readonly bool) *VirtioBuffer {
	buf := new(VirtioBuffer)
	buf.data = make([][]byte, 0, 1)
	buf.index = index
	buf.readonly = readonly
	buf.length = 0
	return buf
}

func (buf *VirtioBuffer) Append(data []byte) {
	buf.data = append(buf.data, data)
	buf.length += len(data)
}

func (buf *VirtioBuffer) Length() int {
	return buf.length
}

func (buf *VirtioBuffer) SetLength(length int) {
	buf.length = length
}

func (buf *VirtioBuffer) GatherIovec(
	offset int,
	length int) []syscall.Iovec {

	iovecs := make([]syscall.Iovec, 0, len(buf.data))

	for _, data := range buf.data {
		if offset >= len(data) {
			offset -= len(data)
		} else if offset > 0 {
			l := len(data) - offset
			if l > length {
				l = length
			}
			iovecs = append(iovecs, syscall.Iovec{
				Base: &data[offset],
				Len:  uint64(l),
			})
			length -= l
			offset = 0
		} else {
			l := len(data)
			if l > length {
				l = length
			}
			iovecs = append(iovecs, syscall.Iovec{
				Base: &data[0],
				Len:  uint64(l),
			})
			length -= l
		}

		if length == 0 {
			break
		}
	}

	return iovecs
}

func (buf *VirtioBuffer) doIO(
	fd int,
	fd_offset int64,
	buf_offset int,
	length int,
	write bool) (int, error) {

	iovecs := buf.GatherIovec(buf_offset, length)
	if len(iovecs) == 0 {
		return 0, nil
	}

	var rval uintptr
	var e syscall.Errno
	if fd_offset != -1 {
		if write {
			rval, _, e = syscall.Syscall6(
				syscall.SYS_PWRITEV,
				uintptr(fd),
				uintptr(unsafe.Pointer(&iovecs[0])),
				uintptr(len(iovecs)),
				uintptr(fd_offset),
				0,
				0)
		} else {
			rval, _, e = syscall.Syscall6(
				syscall.SYS_PREADV,
				uintptr(fd),
				uintptr(unsafe.Pointer(&iovecs[0])),
				uintptr(len(iovecs)),
				uintptr(fd_offset),
				0,
				0)
		}
	} else {
		if write {
			rval, _, e = syscall.Syscall(
				syscall.SYS_WRITEV,
				uintptr(fd),
				uintptr(unsafe.Pointer(&iovecs[0])),
				uintptr(len(iovecs)))
		} else {
			rval, _, e = syscall.Syscall(
				syscall.SYS_READV,
				uintptr(fd),
				uintptr(unsafe.Pointer(&iovecs[0])),
				uintptr(len(iovecs)))
		}
	}

	if e != 0 {
		return 0, e
	}
	return int(rval), nil
}

func (buf *VirtioBuffer) Write(
	fd int,
	buf_offset int,
	length int) (int, error) {

	return buf.doIO(fd, -1, buf_offset, length, true)
}

func (buf *VirtioBuffer) PWrite(
	fd int,
	fd_offset int64,
	buf_offset int,
	length int) (int, error) {

	return buf.doIO(fd, fd_offset, buf_offset, length, true)
}

func (buf *VirtioBuffer) Read(
	fd int,
	buf_offset int,
	length int) (int, error) {

	return buf.doIO(fd, -1, buf_offset, length, false)
}

func (buf *VirtioBuffer) PRead(
	fd int,
	fd_offset int64,
	buf_offset int,
	length int) (int, error) {

	return buf.doIO(fd, fd_offset, buf_offset, length, false)
}

func (buf *VirtioBuffer) Map(
	offset int,
	length int) []byte {

	// Empty read?
	if length == 0 {
		return []byte{}
	}

	for _, data := range buf.data {
		if offset >= len(data) {
			offset -= len(data)
		} else if offset > 0 {
			if length > len(data)-offset {
				return data[offset:len(data)]
			} else {
				return data[offset : offset+length]
			}
		} else {
			if length > len(data) {
				return data
			} else {
				return data[:length]
			}
		}
	}

	// We never found the offset,
	// give back nothing to indicate.
	return nil
}

func (buf *VirtioBuffer) CopyOut(
	offset int,
	output []byte) int {

	copied := 0

	for _, data := range buf.data {
		if offset >= len(data) {
			offset -= len(data)
			continue
		} else if offset > 0 {
			data = data[offset:]
		}

		if len(data) > len(output) {
			copy(output, data[:len(output)])
			copied += len(output)
			break
		} else {
			copy(output, data)
			copied += len(data)
			output = output[len(data):]
		}
	}

	return copied
}
