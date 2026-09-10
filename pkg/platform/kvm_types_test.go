// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package platform

import (
	"testing"
	"unsafe"
)

func TestKvmStructSizes(t *testing.T) {
	tests := []struct {
		name     string
		got      uintptr
		expected uintptr
	}{
		{"kvmUserspaceMemoryRegion", unsafe.Sizeof(kvmUserspaceMemoryRegion{}), 32},
		{"kvmRegs", unsafe.Sizeof(kvmRegs{}), 144},
		{"kvmSegment", unsafe.Sizeof(kvmSegment{}), 24},
		{"kvmDtable", unsafe.Sizeof(kvmDtable{}), 16},
		{"kvmSregs", unsafe.Sizeof(kvmSregs{}), 312},
		{"kvmFpu", unsafe.Sizeof(kvmFpu{}), 416},
		{"kvmPitChannelState", unsafe.Sizeof(kvmPitChannelState{}), 24},
		{"kvmPitState2", unsafe.Sizeof(kvmPitState2{}), 112},
		{"kvmPitConfig", unsafe.Sizeof(kvmPitConfig{}), 64},
		{"kvmClockData", unsafe.Sizeof(kvmClockData{}), 48},
		{"kvmVcpuEvents", unsafe.Sizeof(kvmVcpuEvents{}), 64},
		{"kvmMpState", unsafe.Sizeof(kvmMpState{}), 4},
		{"kvmLapicState", unsafe.Sizeof(kvmLapicState{}), 1024},
		{"kvmXsave", unsafe.Sizeof(kvmXsave{}), 4096},
		{"kvmXcr", unsafe.Sizeof(kvmXcr{}), 16},
		{"kvmIrqLevel", unsafe.Sizeof(kvmIrqLevel{}), 8},
		{"kvmMsi", unsafe.Sizeof(kvmMsi{}), 32},
		{"kvmIoeventfd", unsafe.Sizeof(kvmIoeventfd{}), 64},
		{"kvmIrqfd", unsafe.Sizeof(kvmIrqfd{}), 32},
		{"kvmGuestDebug", unsafe.Sizeof(kvmGuestDebug{}), 72},
		{"kvmTranslation", unsafe.Sizeof(kvmTranslation{}), 24},
		{"kvmMsrEntry", unsafe.Sizeof(kvmMsrEntry{}), 16},
		{"kvmCpuidEntry2", unsafe.Sizeof(kvmCpuidEntry2{}), 40},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("sizeof(%s) = %d, expected %d", tt.name, tt.got, tt.expected)
		}
	}
}

func TestKvmRunOffsets(t *testing.T) {
	var run kvmRun
	base := uintptr(unsafe.Pointer(&run))

	if off := uintptr(unsafe.Pointer(&run.requestInterruptWindow)) - base; off != 0 {
		t.Errorf("run.requestInterruptWindow offset = %d, want 0", off)
	}
	if off := uintptr(unsafe.Pointer(&run.immediateExit)) - base; off != 1 {
		t.Errorf("run.immediateExit offset = %d, want 1", off)
	}
	if off := uintptr(unsafe.Pointer(&run.exitReason)) - base; off != 8 {
		t.Errorf("run.exitReason offset = %d, want 8", off)
	}
	if off := uintptr(unsafe.Pointer(&run.cr8)) - base; off != 16 {
		t.Errorf("run.cr8 offset = %d, want 16", off)
	}
	if off := uintptr(unsafe.Pointer(&run.flags)) - base; off != 24 {
		t.Errorf("run.flags offset = %d, want 24", off)
	}
	if off := uintptr(unsafe.Pointer(&run.data)) - base; off != 32 {
		t.Errorf("run.data offset = %d, want 32", off)
	}
}

func TestKvmVcpuEventsOffsets(t *testing.T) {
	var ev kvmVcpuEvents
	base := uintptr(unsafe.Pointer(&ev))

	if off := uintptr(unsafe.Pointer(&ev.exception)) - base; off != 0 {
		t.Errorf("ev.exception offset = %d, want 0", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.interrupt)) - base; off != 8 {
		t.Errorf("ev.interrupt offset = %d, want 8", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.nmi)) - base; off != 12 {
		t.Errorf("ev.nmi offset = %d, want 12", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.sipiVector)) - base; off != 16 {
		t.Errorf("ev.sipiVector offset = %d, want 16", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.flags)) - base; off != 20 {
		t.Errorf("ev.flags offset = %d, want 20", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.smi)) - base; off != 24 {
		t.Errorf("ev.smi offset = %d, want 24", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.tripleFault)) - base; off != 28 {
		t.Errorf("ev.tripleFault offset = %d, want 28", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.reserved)) - base; off != 29 {
		t.Errorf("ev.reserved offset = %d, want 29", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.exceptionHasPayload)) - base; off != 55 {
		t.Errorf("ev.exceptionHasPayload offset = %d, want 55", off)
	}
	if off := uintptr(unsafe.Pointer(&ev.exceptionPayload)) - base; off != 56 {
		t.Errorf("ev.exceptionPayload offset = %d, want 56", off)
	}
}

func TestKvmPitStateOffsets(t *testing.T) {
	var pit kvmPitState2
	base := uintptr(unsafe.Pointer(&pit))

	if off := uintptr(unsafe.Pointer(&pit.channels)) - base; off != 0 {
		t.Errorf("pit.channels offset = %d, want 0", off)
	}
	if off := uintptr(unsafe.Pointer(&pit.flags)) - base; off != 72 {
		t.Errorf("pit.flags offset = %d, want 72", off)
	}
	if off := uintptr(unsafe.Pointer(&pit.reserved)) - base; off != 76 {
		t.Errorf("pit.reserved offset = %d, want 76", off)
	}
}
