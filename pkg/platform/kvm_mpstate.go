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
	"encoding/json"
	"syscall"
	"unsafe"
)

// Our vcpus state.
type MpState int

const (
	MpStateRunnable      = MpState(0)
	MpStateUninitialized = MpState(1)
	MpStateInitReceived  = MpState(2)
	MpStateHalted        = MpState(3)
	MpStateSipiReceived  = MpState(4)
)

var stateMap = map[MpState]string{
	MpStateRunnable:      "runnable",
	MpStateUninitialized: "uninitialized",
	MpStateInitReceived:  "init-received",
	MpStateHalted:        "halted",
	MpStateSipiReceived:  "sipi-received",
}

var stateRevMap = map[string]MpState{
	"runnable":      MpStateRunnable,
	"uninitialized": MpStateUninitialized,
	"init-received": MpStateInitReceived,
	"halted":        MpStateHalted,
	"sipi-received": MpStateSipiReceived,
}

func (vcpu *Vcpu) GetMpState() (MpState, error) {
	var kvmState kvmMpState
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlGetMpState),
		uintptr(unsafe.Pointer(&kvmState)))
	if e != 0 {
		return MpState(kvmState.mpState), e
	}

	return MpState(kvmState.mpState), nil
}

func (vcpu *Vcpu) SetMpState(state MpState) error {
	var kvmState kvmMpState
	kvmState.mpState = uint32(state)
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(IoctlSetMpState),
		uintptr(unsafe.Pointer(&kvmState)))
	if e != 0 {
		return e
	}

	return nil
}

func (state *MpState) MarshalJSON() ([]byte, error) {

	// Marshal as a string.
	value, ok := stateMap[*state]
	if !ok {
		return nil, UnknownState
	}

	return json.Marshal(value)
}

func (state *MpState) UnmarshalJSON(data []byte) error {

	// Unmarshal as an string.
	var value string
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	// Find the state.
	newstate, ok := stateRevMap[value]
	if !ok {
		return UnknownState
	}

	// That's our state.
	*state = newstate
	return nil
}
