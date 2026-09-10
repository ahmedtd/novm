// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64

package platform

func nativeCpuidAsm(funcNum uint32) (eax, ebx, ecx, edx uint32)
