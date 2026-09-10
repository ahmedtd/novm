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

//go:build linux

package platform

import (
	"syscall"
)

// Capabilities (extensions).
const (
	CapUserMem               = 3
	CapSetIdentityMapAddr    = 37
	CapIrqChip               = 0
	CapIoFd                  = 36
	CapIrqFd                 = 32
	CapPit2                  = 33
	CapPitState2             = 35
	CapCpuid                 = 7
	CapSignalMsi             = 77
	CapVcpuEvents            = 41
	CapAdjustClock           = 39
	CapXSave                 = 55
	CapXcrs                  = 56
)

type kvmCapability struct {
	name   string
	number uintptr
}

func (capability *kvmCapability) Error() string {
	return "Missing capability: " + capability.name
}

//
// Our required capabilities.
//
// Many of these are actually optional, but none
// of the plumbing has been done to gracefully fail
// when they are not available. For the time being
// development is focused on legacy-free environments,
// so we can split this out when it's necessary later.
//
var requiredCapabilities = []kvmCapability{
	kvmCapability{"User Memory", uintptr(CapUserMem)},
	kvmCapability{"Identity Map", uintptr(CapSetIdentityMapAddr)},
	kvmCapability{"IRQ Chip", uintptr(CapIrqChip)},
	kvmCapability{"IO Event FD", uintptr(CapIoFd)},
	kvmCapability{"IRQ Event FD", uintptr(CapIrqFd)},
	kvmCapability{"PIT2", uintptr(CapPit2)},
	kvmCapability{"PITSTATE2", uintptr(CapPitState2)},
	kvmCapability{"Clock", uintptr(CapAdjustClock)},
	kvmCapability{"CPUID", uintptr(CapCpuid)},
	kvmCapability{"MSI", uintptr(CapSignalMsi)},
	kvmCapability{"VCPU Events", uintptr(CapVcpuEvents)},
	kvmCapability{"XSAVE", uintptr(CapXSave)},
	kvmCapability{"XCRS", uintptr(CapXcrs)},
}

func checkCapability(
	fd int,
	capability kvmCapability) error {

	r, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(IoctlCheckExtension),
		capability.number)
	if r != 1 || e != 0 {
		return &capability
	}

	return nil
}

func checkCapabilities(fd int) error {
	// Check our extensions.
	for _, capSpec := range requiredCapabilities {
		err := checkCapability(fd, capSpec)
		if err != nil {
			return err
		}
	}

	return nil
}
