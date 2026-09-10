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
	"encoding/binary"
	"unsafe"
)

func (vcpu *Vcpu) GetExitError() error {
	switch vcpu.kvm.exitReason {
	case ExitReasonMmio:
		physAddr := binary.LittleEndian.Uint64(vcpu.kvm.data[0:8])
		dataPtr := (*uint64)(unsafe.Pointer(&vcpu.kvm.data[8]))
		length := binary.LittleEndian.Uint32(vcpu.kvm.data[16:20])
		isWrite := vcpu.kvm.data[20] != 0
		return &ExitMmio{
			addr:   Paddr(physAddr),
			data:   dataPtr,
			length: length,
			write:  isWrite,
		}

	case ExitReasonIo:
		direction := vcpu.kvm.data[0]
		size := vcpu.kvm.data[1]
		port := binary.LittleEndian.Uint16(vcpu.kvm.data[2:4])
		dataOffset := binary.LittleEndian.Uint64(vcpu.kvm.data[8:16])
		dataPtr := (*uint64)(unsafe.Pointer(uintptr(unsafe.Pointer(vcpu.kvm)) + uintptr(dataOffset)))
		return &ExitPio{
			port: Paddr(port),
			size: size,
			data: dataPtr,
			out:  direction == 1, // KVM_EXIT_IO_OUT
		}

	case ExitReasonInternalError:
		suberror := binary.LittleEndian.Uint32(vcpu.kvm.data[0:4])
		return &ExitInternalError{
			code: suberror,
		}

	case ExitReasonException:
		exception := binary.LittleEndian.Uint32(vcpu.kvm.data[0:4])
		errorCode := binary.LittleEndian.Uint32(vcpu.kvm.data[4:8])
		return &ExitException{
			exception: exception,
			errorCode: errorCode,
		}

	case ExitReasonDebug:
		return &ExitDebug{}

	case ExitReasonShutdown:
		return &ExitShutdown{}

	default:
		return &ExitUnknown{
			code: vcpu.kvm.exitReason,
		}
	}
}
