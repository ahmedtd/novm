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
	"encoding/binary"
	"syscall"
	"unsafe"
)

type Cpuid struct {
	Function uint32 `json:"function"`
	Index    uint32 `json:"index"`
	Flags    uint32 `json:"flags"`

	EAX uint32
	EBX uint32
	ECX uint32
	EDX uint32
}

func supportedCpuid(fd int) ([]Cpuid, error) {
	nent := uint32((PageSize - 8) / 40)
	buf := make([]byte, 8+nent*40)
	binary.LittleEndian.PutUint32(buf[0:4], nent)

	for {
		_, _, e := syscall.Syscall(
			syscall.SYS_IOCTL,
			uintptr(fd),
			uintptr(IoctlGetSupportedCpuid),
			uintptr(unsafe.Pointer(&buf[0])))

		if e == syscall.E2BIG || e == syscall.ENOMEM {
			nent = binary.LittleEndian.Uint32(buf[0:4])
			buf = make([]byte, 8+nent*40)
			binary.LittleEndian.PutUint32(buf[0:4], nent)
			continue
		} else if e != 0 {
			return nil, e
		}

		break
	}

	nent = binary.LittleEndian.Uint32(buf[0:4])
	cpuids := make([]Cpuid, 0, nent)
	for i := uint32(0); i < nent; i++ {
		off := 8 + i*40
		cpuids = append(cpuids, Cpuid{
			Function: binary.LittleEndian.Uint32(buf[off : off+4]),
			Index:    binary.LittleEndian.Uint32(buf[off+4 : off+8]),
			Flags:    binary.LittleEndian.Uint32(buf[off+8 : off+12]),
			EAX:      binary.LittleEndian.Uint32(buf[off+12 : off+16]),
			EBX:      binary.LittleEndian.Uint32(buf[off+16 : off+20]),
			ECX:      binary.LittleEndian.Uint32(buf[off+20 : off+24]),
			EDX:      binary.LittleEndian.Uint32(buf[off+24 : off+28]),
		})
	}

	return cpuids, nil
}

func nativeCpuid(function uint32) Cpuid {
	eax, ebx, ecx, edx := nativeCpuidAsm(function)
	return Cpuid{
		Function: function,
		EAX:      eax,
		EBX:      ebx,
		ECX:      ecx,
		EDX:      edx,
	}
}

func defaultCpuid(fd int) ([]Cpuid, error) {

	// Get the supported cpuids.
	cpuids, err := supportedCpuid(fd)
	if err != nil {
		return nil, err
	}

	// Change the vendor & feature bits.
	result := make([]Cpuid, 0, len(cpuids))
	for _, cpuid := range cpuids {

		if cpuid.Function == 0 {
			// Tweak our vendor.
			native_cpuid := nativeCpuid(cpuid.Function)
			cpuid.EBX = native_cpuid.EBX
			cpuid.ECX = native_cpuid.ECX
			cpuid.EDX = native_cpuid.EDX

		} else if cpuid.Function == 1 {
			// Tweak our model & APIC status.
			native_cpuid := nativeCpuid(cpuid.Function)
			cpuid.EAX = native_cpuid.EAX
			cpuid.EDX |= (1 << 9)

		} else if cpuid.Function == 0x80000001 {
			// Mask our NX support.
			// FIXME: This seems to cause the system
			// to freeze up during boot. I'm not sure
			// why NX support would do that, but it's
			// a mystery that should be solved soon.
			cpuid.EDX &= ^uint32(1 << 20)
		}

		result = append(result, cpuid)
	}

	return result, nil
}

func (vcpu *Vcpu) SetCpuid(cpuids []Cpuid) error {
	buf := make([]byte, 8+len(cpuids)*40)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(cpuids)))
	for i, cpuid := range cpuids {
		off := 8 + i*40
		binary.LittleEndian.PutUint32(buf[off:off+4], cpuid.Function)
		binary.LittleEndian.PutUint32(buf[off+4:off+8], cpuid.Index)
		binary.LittleEndian.PutUint32(buf[off+8:off+12], cpuid.Flags)
		binary.LittleEndian.PutUint32(buf[off+12:off+16], cpuid.EAX)
		binary.LittleEndian.PutUint32(buf[off+16:off+20], cpuid.EBX)
		binary.LittleEndian.PutUint32(buf[off+20:off+24], cpuid.ECX)
		binary.LittleEndian.PutUint32(buf[off+24:off+28], cpuid.EDX)
	}

	// Set our vcpuid.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlSetCpuid),
		uintptr(unsafe.Pointer(&buf[0])))
	if e != 0 {
		return e
	}

	// We're good.
	vcpu.cpuid = cpuids
	return nil
}

func (vcpu *Vcpu) GetCpuid() ([]Cpuid, error) {
	// This is super annoying. If we are querying
	// capabilities, then it expects us to give the
	// size of the buffer we pass, and it will say ENOMEM
	// if have too many entries. On the other hand, if
	// we are calling GET_CPUID2, then it expects us to
	// pass zero and will only adjust nent after it gives
	// us E2BIG as a result. How dumb is that?
	// Anyways, all this lead to just caching the thing.
	return vcpu.cpuid, nil
}
