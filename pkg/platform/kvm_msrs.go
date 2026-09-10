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

type Msr struct {
	Index uint32 `json:"index"`
	Value uint64 `json:"value"`
}

func availableMsrs(fd int) ([]uint32, error) {
	// Find our list of MSR indices.
	// First probe the number of MSRs by passing nmsrs = 0.
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 0)

	for {
		_, _, e := syscall.Syscall(
			syscall.SYS_IOCTL,
			uintptr(fd),
			uintptr(IoctlGetMsrIndexList),
			uintptr(unsafe.Pointer(&buf[0])))
		if e == syscall.E2BIG {
			nmsrs := binary.LittleEndian.Uint32(buf)
			buf = make([]byte, 4+nmsrs*4)
			binary.LittleEndian.PutUint32(buf, nmsrs)
			continue
		} else if e != 0 {
			return nil, e
		}
		break
	}

	nmsrs := binary.LittleEndian.Uint32(buf[:4])
	msrs := make([]uint32, nmsrs)
	for i := uint32(0); i < nmsrs; i++ {
		msrs[i] = binary.LittleEndian.Uint32(buf[4+i*4 : 8+i*4])
	}

	return msrs, nil
}

func (vcpu *Vcpu) GetMsr(index uint32) (uint64, error) {
	var msrs kvmMsrs
	msrs.nmsrs = 1
	msrs.entries[0].index = index

	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlGetMsrs),
		uintptr(unsafe.Pointer(&msrs)))
	if e != 0 {
		return 0, e
	}

	return msrs.entries[0].data, nil
}

func (vcpu *Vcpu) SetMsr(index uint32, value uint64) error {
	var msrs kvmMsrs
	msrs.nmsrs = 1
	msrs.entries[0].index = index
	msrs.entries[0].data = value

	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlSetMsrs),
		uintptr(unsafe.Pointer(&msrs)))
	if e != 0 {
		return e
	}

	return nil
}

func (vcpu *Vcpu) GetMsrs() ([]Msr, error) {

	// Extract each msr individually.
	msrs := make([]Msr, 0, len(vcpu.msrs))

	for _, index := range vcpu.msrs {

		// Get this MSR.
		value, err := vcpu.GetMsr(index)
		if err != nil {
			return msrs, err
		}

		// Got one.
		msrs = append(msrs, Msr{uint32(index), uint64(value)})
	}

	// Finish it off.
	return msrs, nil
}

func (vcpu *Vcpu) SetMsrs(msrs []Msr) error {

	for _, msr := range msrs {
		// Set our msrs.
		err := vcpu.SetMsr(msr.Index, msr.Value)
		if err != nil {
			return err
		}
	}

	return nil
}
