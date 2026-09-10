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

// Our event state.
type ExceptionEvent struct {
	Number    uint8   `json:"number"`
	ErrorCode *uint32 `json:"error-code"`
}

type InterruptEvent struct {
	Number uint8 `json:"number"`
	Soft   bool  `json:"soft"`
	Shadow bool  `json:"shadow"`
}

type Events struct {
	Exception *ExceptionEvent `json:"exception"`
	Interrupt *InterruptEvent `json:"interrupt"`

	NmiPending bool `json:"nmi-pending"`
	NmiMasked  bool `json:"nmi-masked"`

	SipiVector uint32 `json:"sipi-vector"`
	Flags      uint32 `json:"flags"`
}

func (vcpu *Vcpu) GetEvents() (Events, error) {
	var kvmEvents kvmVcpuEvents
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlGetVcpuEvents),
		uintptr(unsafe.Pointer(&kvmEvents)))
	if e != 0 {
		return Events{}, e
	}

	events := Events{
		NmiPending: kvmEvents.nmi.pending != 0,
		NmiMasked:  kvmEvents.nmi.masked != 0,
		SipiVector: kvmEvents.sipiVector,
		Flags:      kvmEvents.flags,
	}
	if kvmEvents.exception.injected != 0 {
		events.Exception = &ExceptionEvent{
			Number: kvmEvents.exception.nr,
		}
		if kvmEvents.exception.hasErrorCode != 0 {
			errorCode := kvmEvents.exception.errorCode
			events.Exception.ErrorCode = &errorCode
		}
	}
	if kvmEvents.interrupt.injected != 0 {
		events.Interrupt = &InterruptEvent{
			Number: kvmEvents.interrupt.nr,
			Soft:   kvmEvents.interrupt.soft != 0,
			Shadow: kvmEvents.interrupt.shadow != 0,
		}
	}

	return events, nil
}

func (vcpu *Vcpu) SetEvents(events Events) error {
	var kvmEvents kvmVcpuEvents

	if events.NmiPending {
		kvmEvents.nmi.pending = 1
	}
	if events.NmiMasked {
		kvmEvents.nmi.masked = 1
	}

	kvmEvents.sipiVector = events.SipiVector
	kvmEvents.flags = events.Flags

	if events.Exception != nil {
		kvmEvents.exception.injected = 1
		kvmEvents.exception.nr = events.Exception.Number
		if events.Exception.ErrorCode != nil {
			kvmEvents.exception.hasErrorCode = 1
			kvmEvents.exception.errorCode = *events.Exception.ErrorCode
		}
	}
	if events.Interrupt != nil {
		kvmEvents.interrupt.injected = 1
		kvmEvents.interrupt.nr = events.Interrupt.Number
		if events.Interrupt.Soft {
			kvmEvents.interrupt.soft = 1
		}
		if events.Interrupt.Shadow {
			kvmEvents.interrupt.shadow = 1
		}
	}

	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlSetVcpuEvents),
		uintptr(unsafe.Pointer(&kvmEvents)))
	if e != 0 {
		return e
	}

	return nil
}
