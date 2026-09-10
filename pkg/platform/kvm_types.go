// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

// KVM IOCTL numbers (Linux x86_64 ABI).
const (
	IoctlGetApiVersion         = 0xae00
	IoctlCreateVm              = 0xae01
	IoctlGetMsrIndexList       = 0xc004ae02
	IoctlCheckExtension        = 0xae03
	IoctlGetVcpuMmapSize       = 0xae04
	IoctlGetSupportedCpuid     = 0xc008ae05
	IoctlCreateVcpu            = 0xae41
	IoctlSetTssAddr            = 0xae47
	IoctlSetIdentityMapAddr    = 0x4008ae48
	IoctlSetUserMemoryRegion   = 0x4020ae46
	IoctlCreateIrqchip         = 0xae60
	IoctlCreateIrqChip         = 0xae60
	IoctlGetIrqChip            = 0xc208ae62
	IoctlSetIrqChip            = 0x8208ae63
	IoctlIrqLine               = 0x4008ae61
	IoctlSignalMsi             = 0x4020aea5
	ApicSize                   = 1024
	IoctlIrqfd                 = 0x4020ae76
	IoctlCreatePit2            = 0x4040ae77
	IoctlSetClock              = 0x4030ae7b
	IoctlGetClock              = 0x8030ae7c
	IoctlIoeventfd             = 0x4040ae79
	IoctlIoEventFd             = 0x4040ae79
	IoctlIoEventFdFlagDatamatch = 0x1
	IoctlIoEventFdFlagPio       = 0x2
	IoctlIoEventFdFlagDeassign  = 0x4
	IoctlRun                   = 0xae80
	IoctlGetRegs               = 0x8090ae81
	IoctlSetRegs               = 0x4090ae82
	IoctlGetSregs              = 0x8138ae83
	IoctlSetSregs              = 0x4138ae84
	IoctlGetMsrs               = 0xc008ae88
	IoctlSetMsrs               = 0x4008ae89
	IoctlSetSignalMask         = 0x4004ae8b
	IoctlTranslate             = 0xc018ae85
	IoctlGetFpu                = 0x81a0ae8c
	IoctlSetFpu                = 0x41a0ae8d
	IoctlGetLapic              = 0x8400ae8e
	IoctlSetLapic              = 0x4400ae8f
	IoctlSetCpuid              = 0x4008ae90
	IoctlGetCpuid              = 0xc008ae91
	IoctlSetGuestDebug         = 0x4048ae9b
	IoctlGuestDebugEnable      = 0x3
	IoctlGetMpState            = 0x8004ae98
	IoctlSetMpState            = 0x4004ae99
	IoctlGetVcpuEvents         = 0x8040ae9f
	IoctlSetVcpuEvents         = 0x4040aea0
	IoctlGetPit2               = 0x8070ae9f
	IoctlSetPit2               = 0x4070aea0
	IoctlGetXsave              = 0x9000aea4
	IoctlSetXsave              = 0x5000aea5
	IoctlGetXcrs               = 0x8188aea6
	IoctlSetXcrs               = 0x4188aea7
)

// Exit reasons.
const (
	ExitReasonUnknown       = 0
	ExitReasonException     = 1
	ExitReasonIo            = 2
	ExitReasonHypercall     = 3
	ExitReasonDebug         = 4
	ExitReasonHlt           = 5
	ExitReasonMmio          = 6
	ExitReasonIrqWindowOpen = 7
	ExitReasonShutdown      = 8
	ExitReasonFailEntry     = 9
	ExitReasonIntr          = 10
	ExitReasonSetTpr        = 11
	ExitReasonTprAccess     = 12
	ExitReasonS390Sieic     = 13
	ExitReasonS390Reset     = 14
	ExitReasonDcr           = 15
	ExitReasonNmi           = 16
	ExitReasonInternalError = 17
	ExitReasonOsi           = 18
	ExitReasonPaprHcall     = 19
	ExitReasonS390Ucontrol  = 20
	ExitReasonWatchdog      = 21
	ExitReasonS390Tsch      = 22
	ExitReasonEpr           = 23
	ExitReasonSystemEvent   = 24
)

// KVM structures matching Linux x86 ABI.

type kvmUserspaceMemoryRegion struct {
	slot          uint32
	flags         uint32
	guestPhysAddr uint64
	memorySize    uint64
	userspaceAddr uint64
}

type kvmTranslation struct {
	linearAddress   uint64
	physicalAddress uint64
	valid           uint8
	writeable       uint8
	usermode        uint8
	pad             [5]uint8
}

type kvmRegs struct {
	rax    uint64
	rbx    uint64
	rcx    uint64
	rdx    uint64
	rsi    uint64
	rdi    uint64
	rsp    uint64
	rbp    uint64
	r8     uint64
	r9     uint64
	r10    uint64
	r11    uint64
	r12    uint64
	r13    uint64
	r14    uint64
	r15    uint64
	rip    uint64
	rflags uint64
}

type kvmSegment struct {
	base     uint64
	limit    uint32
	selector uint16
	_type    uint8
	present  uint8
	dpl      uint8
	db       uint8
	s        uint8
	l        uint8
	g        uint8
	avl      uint8
	unusable uint8
	padding  uint8
}

type kvmDtable struct {
	base    uint64
	limit   uint16
	padding [3]uint16
}

type kvmSregs struct {
	cs               kvmSegment
	ds               kvmSegment
	es               kvmSegment
	fs               kvmSegment
	gs               kvmSegment
	ss               kvmSegment
	tr               kvmSegment
	ldt              kvmSegment
	gdt              kvmDtable
	idt              kvmDtable
	cr0              uint64
	cr2              uint64
	cr3              uint64
	cr4              uint64
	cr8              uint64
	efer             uint64
	apic_base        uint64
	interrupt_bitmap [4]uint64
}

type kvmFpu struct {
	fpr        [8][16]byte
	fcw        uint16
	fsw        uint16
	ftwx       uint8
	pad1       uint8
	lastOpcode uint16
	lastIP     uint64
	lastDP     uint64
	xmm        [16][16]byte
	mxcsr      uint32
	pad2       uint32
}

type kvmMsrEntry struct {
	index    uint32
	reserved uint32
	data     uint64
}

type kvmMsrs struct {
	nmsrs   uint32
	pad     uint32
	entries [1]kvmMsrEntry
}

type kvmMsrList struct {
	nmsrs   uint32
	indices [1]uint32
}

type kvmCpuidEntry2 struct {
	function uint32
	index    uint32
	flags    uint32
	eax      uint32
	ebx      uint32
	ecx      uint32
	edx      uint32
	padding  [3]uint32
}

type kvmCpuid2 struct {
	nent    uint32
	padding uint32
	entries [1]kvmCpuidEntry2
}

type kvmPitChannelState struct {
	count          uint32
	latchedCount   uint16
	countLatched   uint8
	statusLatched  uint8
	status         uint8
	readState      uint8
	writeState     uint8
	writeLatch     uint8
	rwMode         uint8
	mode           uint8
	bcd            uint8
	gate           uint8
	countLoadTime  int64
}

type kvmPitState2 struct {
	channels [3]kvmPitChannelState
	flags    uint32
	reserved [9]uint32
}

type kvmPitConfig struct {
	flags uint32
	pad   [15]uint32
}

type kvmGuestDebug struct {
	control uint32
	pad     uint32
	arch    [64]byte
}

type kvmClockData struct {
	clock uint64
	flags uint32
	pad   [9]uint32
}

type kvmVcpuEvents struct {
	exception struct {
		injected     uint8
		nr           uint8
		hasErrorCode uint8
		pending      uint8
		errorCode    uint32
	}
	interrupt struct {
		injected uint8
		nr       uint8
		soft     uint8
		shadow   uint8
	}
	nmi struct {
		injected uint8
		pending  uint8
		masked   uint8
		pad      uint8
	}
	sipiVector uint32
	flags      uint32
	smi        struct {
		smm          uint8
		pending      uint8
		smmInsideNmi uint8
		latchedInit  uint8
	}
	tripleFault struct {
		pending uint8
	}
	reserved            [26]uint8
	exceptionHasPayload uint8
	exceptionPayload    uint64
}

type kvmMpState struct {
	mpState uint32
}

type kvmLapicState struct {
	regs [1024]byte
}

type kvmXcr struct {
	xcr      uint32
	reserved uint32
	value    uint64
}

type kvmXcrs struct {
	nrXcrs  uint32
	flags   uint32
	xcrs    [16]kvmXcr
	padding [16]uint64
}

type kvmXsave struct {
	region [1024]uint32
}

type kvmIrqLevel struct {
	irq   uint32
	level uint32
}

type kvmMsi struct {
	addressLo uint32
	addressHi uint32
	data      uint32
	flags     uint32
	pad       [16]uint8
}

type kvmIoeventfd struct {
	datamatch uint64
	addr      uint64
	len       uint32
	fd        int32
	flags     uint32
	pad       [36]uint8
}

type kvmIrqfd struct {
	fd          uint32
	gsi         uint32
	flags       uint32
	resamplefd  uint32
	pad         [16]uint8
}

type kvmSignalMask struct {
	len    uint32
	sigset [8]byte
}

// kvmRun matches the Linux struct kvm_run layout.
type kvmRun struct {
	requestInterruptWindow uint8
	immediateExit          uint8
	padding1               [6]uint8

	exitReason uint32 // offset 8
	sendSig    uint8
	padding2   [3]uint8

	cr8   uint64
	flags uint64

	// offset 32: union of exit information
	data [2320]byte
}
