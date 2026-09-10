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

package loader

import (
	"bytes"
	"debug/elf"
	"syscall"

	"github.com/ahmedtd/novm/pkg/machine"
	"github.com/ahmedtd/novm/pkg/platform"
)

func ElfLoad(
	data []byte,
	model *machine.Model) (uint64, bool, error) {

	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return 0, false, err
	}
	defer f.Close()

	is64bit := f.Class == elf.ELFCLASS64
	if f.Class != elf.ELFCLASS32 && f.Class != elf.ELFCLASS64 {
		return 0, false, syscall.EINVAL
	}

	for _, prog := range f.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}
		if prog.Filesz > prog.Memsz || prog.Filesz == 0 {
			return 0, false, syscall.EINVAL
		}
		newLength := platform.Align(prog.Filesz, platform.PageSize, true)
		dest, err := model.Map(
			machine.MemoryTypeUser,
			platform.Paddr(prog.Paddr),
			newLength,
			true)
		if err != nil {
			return 0, false, err
		}
		if int64(prog.Off+prog.Filesz) > int64(len(data)) {
			return 0, false, syscall.EINVAL
		}
		copy(dest, data[prog.Off:prog.Off+prog.Filesz])
	}

	return f.Entry, is64bit, nil
}
