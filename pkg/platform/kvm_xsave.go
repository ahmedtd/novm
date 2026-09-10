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

// Our xsave state.
type XSave struct {
	Region [1024]uint32 `json:"region"`
}

func (vcpu *Vcpu) GetXSave() (XSave, error) {
	var kvmXsave kvmXsave
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlGetXsave),
		uintptr(unsafe.Pointer(&kvmXsave)))
	if e != 0 {
		return XSave{}, e
	}

	state := XSave{}
	state.Region = kvmXsave.region
	return state, nil
}

func (vcpu *Vcpu) SetXSave(state XSave) error {
	var kvmXsave kvmXsave
	kvmXsave.region = state.Region
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlSetXsave),
		uintptr(unsafe.Pointer(&kvmXsave)))
	if e != 0 {
		return e
	}

	return nil
}
