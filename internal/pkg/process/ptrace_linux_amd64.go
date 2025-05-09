// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"encoding/binary"
	"syscall"

	"github.com/pkg/errors"
)

const syscallInstrSize = 2

func getIP(regs *syscall.PtraceRegs) uintptr {
	return uintptr(regs.Rip)
}

func getRegs(pid int, regsout *syscall.PtraceRegs) error {
	err := syscall.PtraceGetRegs(pid, regsout)
	if err != nil {
		return errors.Wrapf(err, "get registers of process %d", pid)
	}

	return nil
}

func setRegs(pid int, regs *syscall.PtraceRegs) error {
	err := syscall.PtraceSetRegs(pid, regs)
	if err != nil {
		return errors.Wrapf(err, "set registers of process %d", pid)
	}

	return nil
}

// Syscall runs a syscall at main thread of process.
func (p *tracedProgram) Syscall(number uint64, args ...uint64) (uint64, error) {
	// save the original registers and the current instructions
	err := p.Protect()
	if err != nil {
		return 0, err
	}

	var regs syscall.PtraceRegs

	err = getRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}
	// set the registers according to the syscall convention. Learn more about
	// it in `man 2 syscall`. In x86_64 the syscall nr is stored in rax
	// register, and the arguments are stored in rdi, rsi, rdx, r10, r8, r9 in
	// order
	regs.Rax = number
	for index, arg := range args {
		// All these registers are hard coded for x86 platform
		switch index {
		case 0:
			regs.Rdi = arg
		case 1:
			regs.Rsi = arg
		case 2:
			regs.Rdx = arg
		case 3:
			regs.R10 = arg
		case 4:
			regs.R8 = arg
		case 5:
			regs.R9 = arg
		default:
			return 0, errors.New("too many arguments for a syscall")
		}
	}
	err = setRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}

	instruction := make([]byte, syscallInstrSize)
	ip := getIP(p.backupRegs)

	// set the current instruction (the ip register points to) to the `syscall`
	// instruction. In x86_64, the `syscall` instruction is 0x050f.
	binary.LittleEndian.PutUint16(instruction, 0x050f)
	_, err = syscall.PtracePokeData(p.pid, ip, instruction)
	if err != nil {
		return 0, errors.Wrapf(err, "writing data %v to %x", instruction, ip)
	}

	// run one instruction, and stop
	err = p.Step()
	if err != nil {
		return 0, err
	}

	// read registers, the return value of syscall is stored inside rax register
	err = getRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}

	// restore the state saved at beginning.
	return regs.Rax, p.Restore()
}
