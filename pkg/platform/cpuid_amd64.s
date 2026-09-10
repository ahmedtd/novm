// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

// func nativeCpuidAsm(funcNum uint32) (eax, ebx, ecx, edx uint32)
TEXT ·nativeCpuidAsm(SB), NOSPLIT, $0-24
	MOVL funcNum+0(FP), AX
	MOVL $0, CX
	CPUID
	MOVL AX, eax+8(FP)
	MOVL BX, ebx+12(FP)
	MOVL CX, ecx+16(FP)
	MOVL DX, edx+20(FP)
	RET
