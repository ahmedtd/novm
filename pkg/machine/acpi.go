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

package machine

import (
	"encoding/binary"

	"github.com/ahmedtd/novm/pkg/platform"
)

func acpiChecksum(data []byte) byte {
	var total byte
	for _, b := range data {
		total += b
	}
	return 0xff - total + 1
}

func buildRsdp(buf []byte, rsdtAddr uint32, xsdtAddr uint64) int {
	copy(buf[0:8], "RSD PTR ")
	copy(buf[9:15], "PERVIR")
	buf[15] = 2
	binary.LittleEndian.PutUint32(buf[16:20], rsdtAddr)
	binary.LittleEndian.PutUint32(buf[20:24], 36)
	binary.LittleEndian.PutUint64(buf[24:32], xsdtAddr)
	buf[33] = 0
	buf[34] = 0
	buf[35] = 0

	buf[8] = acpiChecksum(buf[:20])
	buf[32] = acpiChecksum(buf[:36])
	return 36
}

func buildAcpiHeader(buf []byte, sig string, length uint32, oemTableID string) {
	copy(buf[0:4], sig)
	binary.LittleEndian.PutUint32(buf[4:8], length)
	buf[8] = 1 // revision
	buf[9] = 0 // checksum placeholder
	copy(buf[10:16], "PERVIR")
	for i := 16; i < 24; i++ {
		buf[i] = 0
	}
	copy(buf[16:24], oemTableID)
	binary.LittleEndian.PutUint32(buf[24:28], 0) // oem_revision
	copy(buf[28:32], "NOVM")                     // asl_compiler_id
	binary.LittleEndian.PutUint32(buf[32:36], 0) // asl_compiler_rev
}

func buildRsdt(buf []byte, madtAddr uint32) int {
	const length = 40
	buildAcpiHeader(buf, "RSDT", length, "RSDT")
	binary.LittleEndian.PutUint32(buf[36:40], madtAddr)
	buf[9] = acpiChecksum(buf[:length])
	return length
}

func buildXsdt(buf []byte, madtAddr uint64) int {
	const length = 44
	buildAcpiHeader(buf, "XSDT", length, "XSDT")
	binary.LittleEndian.PutUint64(buf[36:44], madtAddr)
	buf[9] = acpiChecksum(buf[:length])
	return length
}

func buildDsdt(buf []byte) int {
	const length = 36
	buildAcpiHeader(buf, "DSDT", length, "DSDT")
	buf[9] = acpiChecksum(buf[:length])
	return length
}

func buildMadt(buf []byte, lapicAddr uint32, vcpus int, ioapicAddr uint32, ioapicInterrupt uint32) int {
	offset := 44
	for vcpu := 0; vcpu < vcpus; vcpu++ {
		lapic := buf[offset : offset+8]
		lapic[0] = 0 // type = 0
		lapic[1] = 8 // length = 8
		lapic[2] = byte(vcpu)
		lapic[3] = byte(vcpu)
		binary.LittleEndian.PutUint32(lapic[4:8], 1) // flags = Enabled
		offset += 8
	}

	ioapic := buf[offset : offset+12]
	ioapic[0] = 1  // type = 1
	ioapic[1] = 12 // length = 12
	ioapic[2] = 0  // ioapic_id
	ioapic[3] = 0  // reserved
	binary.LittleEndian.PutUint32(ioapic[4:8], ioapicAddr)
	binary.LittleEndian.PutUint32(ioapic[8:12], ioapicInterrupt)
	offset += 12

	totalLen := uint32(offset)
	buildAcpiHeader(buf, "ACPI", totalLen, "MADT")
	binary.LittleEndian.PutUint32(buf[36:40], lapicAddr)
	binary.LittleEndian.PutUint32(buf[40:44], 0) // flags
	buf[9] = acpiChecksum(buf[:totalLen])
	return int(totalLen)
}

type Acpi struct {
	BaseDevice

	Addr platform.Paddr `json:"address"`
	Data []byte         `json:"data"`
}

func NewAcpi(info *DeviceInfo) (Device, error) {
	acpi := new(Acpi)
	acpi.Addr = platform.Paddr(0xf0000)
	return acpi, acpi.init(info)
}

func (acpi *Acpi) Attach(vm *platform.Vm, model *Model) error {

	// Do we already have data?
	rebuild := true
	if acpi.Data == nil {
		// Create our data.
		acpi.Data = make([]byte, platform.PageSize, platform.PageSize)
	} else {
		// Align our data.
		// This is necessary because we map this in
		// directly. It's possible that the data was
		// decoded and refers to the middle of some
		// larger array somewhere, and isn't aligned.
		acpi.Data = platform.AlignBytes(acpi.Data)
		rebuild = false
	}

	// Allocate our memory block.
	err := model.Reserve(
		vm,
		acpi,
		MemoryTypeAcpi,
		acpi.Addr,
		platform.PageSize,
		acpi.Data)
	if err != nil {
		return err
	}

	// Already done.
	if !rebuild {
		return nil
	}

	// Find our APIC information.
	// This will find the APIC device if it
	// is attached, otherwise the MADT table
	// will unfortunately have be a bit invalid.
	var IOApic platform.Paddr
	var LApic platform.Paddr
	for _, device := range model.Devices() {
		apic, ok := device.(*Apic)
		if ok {
			IOApic = apic.IOApic
			LApic = apic.LApic
			break
		}
	}

	// Load the MADT.
	madt_bytes := buildMadt(
		acpi.Data,
		uint32(LApic),
		len(vm.Vcpus()),
		uint32(IOApic),
		0, // I/O APIC interrupt?
	)
	acpi.Debug("MADT %x @ %x", madt_bytes, acpi.Addr)

	// Align offset.
	offset := madt_bytes
	if offset%64 != 0 {
		offset += 64 - (offset % 64)
	}

	// Load the DSDT.
	dsdt_address := uint64(acpi.Addr) + uint64(offset)
	dsdt_bytes := buildDsdt(acpi.Data[int(offset):])
	acpi.Debug("DSDT %x @ %x", dsdt_bytes, dsdt_address)

	// Align offset.
	offset += dsdt_bytes
	if offset%64 != 0 {
		offset += 64 - (offset % 64)
	}

	// Load the XSDT.
	xsdt_address := uint64(acpi.Addr) + uint64(offset)
	xsdt_bytes := buildXsdt(acpi.Data[int(offset):], uint64(acpi.Addr)) // MADT address.
	acpi.Debug("XSDT %x @ %x", xsdt_bytes, xsdt_address)

	// Align offset.
	offset += xsdt_bytes
	if offset%64 != 0 {
		offset += 64 - (offset % 64)
	}

	// Load the RSDT.
	rsdt_address := uint64(acpi.Addr) + uint64(offset)
	rsdt_bytes := buildRsdt(acpi.Data[int(offset):], uint32(acpi.Addr)) // MADT address.
	acpi.Debug("RSDT %x @ %x", rsdt_bytes, rsdt_address)

	// Align offset.
	offset += rsdt_bytes
	if offset%64 != 0 {
		offset += 64 - (offset % 64)
	}

	// Load the RSDP.
	rsdp_address := uint64(acpi.Addr) + uint64(offset)
	rsdp_bytes := buildRsdp(
		acpi.Data[int(offset):],
		uint32(rsdt_address), // RSDT address.
		uint64(xsdt_address), // XSDT address.
	)
	acpi.Debug("RSDP %x @ %x", rsdp_bytes, rsdp_address)

	// Everything went okay.
	return nil
}
