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

// Our FPU state.
type Fpu struct {
	FPR  [8][16]uint8
	FCW  uint16
	FSW  uint16
	FTWX uint8

	LastOpcode uint16 `json:"last-opcode"`
	LastIp     uint64 `json:"last-ip"`
	LastDp     uint64 `json:"last-dp"`

	XMM   [16][16]uint8
	MXCSR uint32
}

func (vcpu *Vcpu) GetFpuState() (Fpu, error) {
	var kvmFpu kvmFpu
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlGetFpu),
		uintptr(unsafe.Pointer(&kvmFpu)))
	if e != 0 {
		return Fpu{}, e
	}

	state := Fpu{}
	state.FPR = kvmFpu.fpr
	state.FCW = kvmFpu.fcw
	state.FSW = kvmFpu.fsw
	state.FTWX = kvmFpu.ftwx
	state.LastOpcode = kvmFpu.lastOpcode
	state.LastIp = kvmFpu.lastIP
	state.LastDp = kvmFpu.lastDP
	state.XMM = kvmFpu.xmm
	state.MXCSR = kvmFpu.mxcsr

	return state, nil
}

func (vcpu *Vcpu) SetFpuState(state Fpu) error {
	var kvmFpu kvmFpu
	kvmFpu.fpr = state.FPR
	kvmFpu.fcw = state.FCW
	kvmFpu.fsw = state.FSW
	kvmFpu.ftwx = state.FTWX
	kvmFpu.lastOpcode = state.LastOpcode
	kvmFpu.lastIP = state.LastIp
	kvmFpu.lastDP = state.LastDp
	kvmFpu.xmm = state.XMM
	kvmFpu.mxcsr = state.MXCSR

	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlSetFpu),
		uintptr(unsafe.Pointer(&kvmFpu)))
	if e != 0 {
		return e
	}

	return nil
}
