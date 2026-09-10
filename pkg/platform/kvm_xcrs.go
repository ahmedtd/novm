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

// A single XCR.
type Xcr struct {
	Id    uint32 `json:"xcr"`
	Value uint64 `json:"value"`
}

func (vcpu *Vcpu) GetXcrs() ([]Xcr, error) {
	var kvmXcrs kvmXcrs
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlGetXcrs),
		uintptr(unsafe.Pointer(&kvmXcrs)))
	if e != 0 {
		return nil, e
	}

	xcrs := make([]Xcr, 0, kvmXcrs.nrXcrs)
	for i := 0; i < int(kvmXcrs.nrXcrs); i += 1 {
		xcrs = append(xcrs, Xcr{
			Id:    kvmXcrs.xcrs[i].xcr,
			Value: kvmXcrs.xcrs[i].value,
		})
	}

	return xcrs, nil
}

func (vcpu *Vcpu) SetXcrs(xcrs []Xcr) error {
	var kvmXcrs kvmXcrs
	kvmXcrs.nrXcrs = uint32(len(xcrs))
	for i, xcr := range xcrs {
		if i >= len(kvmXcrs.xcrs) {
			break
		}
		kvmXcrs.xcrs[i].xcr = xcr.Id
		kvmXcrs.xcrs[i].value = xcr.Value
	}

	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlSetXcrs),
		uintptr(unsafe.Pointer(&kvmXcrs)))
	if e != 0 {
		return e
	}

	return nil
}
