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

//go:build 386 || amd64

package loader

import (
	"encoding/binary"

	"github.com/ahmedtd/novm/pkg/machine"
	"github.com/ahmedtd/novm/pkg/platform"
)

// E820 codes.
const (
	E820Ram      = 1
	E820Reserved = 2
	E820Acpi     = 3
)

func SetupLinuxBootParams(
	model *machine.Model,
	boot_params_data []byte,
	orig_boot_params_data []byte,
	cmdline_addr platform.Paddr,
	initrd_addr platform.Paddr,
	initrd_len uint64) error {

	// The setup header.
	// First step is to copy the existing setup_header
	// out of the given kernel image. We copy only the
	// header, and not the rest of the setup page.
	setup_start := 0x01f1
	setup_end := 0x0202 + int(orig_boot_params_data[0x0201])
	if setup_end > platform.PageSize {
		return InvalidSetupHeader
	}
	copy(boot_params_data[setup_start:setup_end], orig_boot_params_data[setup_start:setup_end])

	// Setup our BIOS memory map (E820).
	// e820_entries is at offset 0x1e8.
	boot_params_data[0x1e8] = byte(len(model.MemoryMap))

	// e820_table is at offset 0x2d0. Each entry is 20 bytes:
	//   __u64 addr (8 bytes)
	//   __u64 size (8 bytes)
	//   __u32 type (4 bytes)
	for index, region := range model.MemoryMap {
		var memtype uint32
		switch region.MemoryType {
		case machine.MemoryTypeUser:
			memtype = E820Ram
		case machine.MemoryTypeReserved:
			memtype = E820Reserved
		case machine.MemoryTypeSpecial:
			memtype = E820Reserved
		case machine.MemoryTypeAcpi:
			memtype = E820Acpi
		}

		offset := 0x2d0 + index*20
		binary.LittleEndian.PutUint64(boot_params_data[offset:], uint64(region.Start))
		binary.LittleEndian.PutUint64(boot_params_data[offset+8:], uint64(region.Size))
		binary.LittleEndian.PutUint32(boot_params_data[offset+16:], memtype)
	}

	// Set necessary setup header bits.
	// vid_mode: 0x1fa (2 bytes)
	binary.LittleEndian.PutUint16(boot_params_data[0x1fa:], 0xffff)
	// type_of_loader: 0x210 (1 byte)
	boot_params_data[0x210] = 0xff
	// loadflags: 0x211 (1 byte)
	boot_params_data[0x211] = 0
	// setup_move_size: 0x212 (2 bytes)
	binary.LittleEndian.PutUint16(boot_params_data[0x212:], 0)
	// ramdisk_image: 0x218 (4 bytes)
	binary.LittleEndian.PutUint32(boot_params_data[0x218:], uint32(initrd_addr))
	// ramdisk_size: 0x21c (4 bytes)
	binary.LittleEndian.PutUint32(boot_params_data[0x21c:], uint32(initrd_len))
	// heap_end_ptr: 0x224 (2 bytes)
	binary.LittleEndian.PutUint16(boot_params_data[0x224:], 0)
	// cmd_line_ptr: 0x228 (4 bytes)
	binary.LittleEndian.PutUint32(boot_params_data[0x228:], uint32(cmdline_addr))
	// setup_data: 0x250 (8 bytes)
	binary.LittleEndian.PutUint64(boot_params_data[0x250:], 0)

	// All done!
	return nil
}
