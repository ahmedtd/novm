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

package platform

import (
	"syscall"
	"unsafe"
)

// Our clock state.
type Clock struct {
	Time  uint64 `json:"time"`
	Flags uint32 `json:"flags"`
}

func (vm *Vm) GetClock() (Clock, error) {
	var kvmClockData kvmClockData
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(IoctlGetClock),
		uintptr(unsafe.Pointer(&kvmClockData)))
	if e != 0 {
		return Clock{}, e
	}

	return Clock{
		Time:  kvmClockData.clock,
		Flags: kvmClockData.flags,
	}, nil
}

func (vm *Vm) SetClock(clock Clock) error {
	var kvmClockData kvmClockData
	kvmClockData.clock = clock.Time
	kvmClockData.flags = clock.Flags
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(IoctlSetClock),
		uintptr(unsafe.Pointer(&kvmClockData)))
	if e != 0 {
		return e
	}

	return nil
}
