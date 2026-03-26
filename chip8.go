package main

import (
	"fmt"
	"os"
)

// chip 8 programs start at location 0x200 (512)
const (
	startAddress = 0x200
	ScreenWidth  = 64 // x ScreenWidth
	ScreenHeight = 32 // y ScreenHeight
)

type Chip8 struct {
	memory [4096]byte                       // 4kb memory
	V      [16]byte                         // registers V0 to VF , VF is flag register and should not be used by any program
	I      uint16                           // 16 bit memory address register
	PC     uint16                           // 16 bit pseudo register "program counter"
	inst   uint16                           // each instruction is 2 bytes long
	stack  [16]uint16                       // array of 16, 16bit values, upto 16 levels of nested subroutines allowed
	SP     uint8                            // points to topmost level of stack
	gfx    [ScreenWidth * ScreenHeight]byte // 64x32 display
	DT     byte                             // delay timer register
	ST     byte                             // sound timer register
	keys   [16]byte                         // keypad has 16 keys
}

var chip8Font = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

func NewChip() (chip *Chip8) {
	chip = &Chip8{}
	chip.PC = startAddress
	// load fonts
	copy(chip.memory[0x050:0x09F], chip8Font[:])
	return
}

// loads the rom into memory 0x200 to 0xFFF
// returns error
func (chip *Chip8) LoadRom(rom string) error {
	data, err := os.ReadFile(rom)
	if err != nil {
		return err
	}

	nbytes := len(data)

	if nbytes <= 0 {
		return fmt.Errorf("Empty ROM %v bytes\n", nbytes)
	} else if nbytes > 4096-startAddress {
		return fmt.Errorf("ROM size %v bytes exceeds maximum allowed %v bytes\n", nbytes, (4096 - startAddress))
	} else {
		// 0x200 is the start of chip8 programs
		copy(chip.memory[startAddress:], data)
	}

	return nil
}

func (chip *Chip8) Cycle() {
	chip.fetch()
	chip.decodeAndExecute()
}

// read and copy two bytes from memory and store as instruction
// immediately increment PC by 2 bytes
func (chip *Chip8) fetch() {
	chip.inst = uint16(chip.memory[chip.PC])<<8 | uint16(chip.memory[chip.PC+1])
	chip.PC += 0x2
}

func (chip *Chip8) decodeAndExecute() {
	// DECODE

	// 0xAXYN instruction format
	// A,X,Y,N are 4 nibbles making up the 16 bits
	// X: second nibble, used to lookup one of the registers VX from V0 to VF
	// Y: third nibble, used to lookup one of the registers VY from V0 to VF
	// N: fourth nibble, a 4 bit number
	// NN: second byte, third and fourth nibbles, an immediate number
	// NNN: second, third and fourth nibble, a 12bit immediate memory address

	inst := chip.inst

	// extract second nibble
	X := (inst & 0x0F00) >> 8
	// extract third nibble
	Y := (inst & 0x00F0) >> 4
	// extract foruth nibble
	N := inst & 0x000F

	// extract second byte
	NN := inst & 0x00FF
	// extract second nibble and byte
	NNN := inst & 0x0FFF

	// EXECUTE
	switch inst & 0xF000 {

	case 0x0000: // 0x00E0, 0x00EE

		switch NN {
		case 0xE0:
			// CLS
			chip.cls()
		case 0xEE:
			// RET
			chip.ret()
		}

	case 0x1000: // 1NNN
		// JP addr
		chip.jp(NNN)

	case 0x2000: // 2NNN
		// CALL addr
		chip.call(NNN)

	case 0x3000: // 3XNN
		// SE VX, byte
		chip.skipIfEqual(X, NN)

	case 0x4000: // 4XNN
		// SNE VX, byte
		chip.skipIfNotEqual(X, NN)

	case 0x5000: // 5XY0
		// SE VX, VY
		chip.skipIfEqualReg(X, Y)

	case 0x6000: // 6XNN
		// LD VX, byte
		chip.ldByte(X, NN)

	case 0x7000: // 7XNN
		// ADD VX, byte
		chip.addByte(X, NN)

	case 0x8000: // 8XY0, 8XY1, 8XY2, 8XY3

		switch N {
		case 0x0:
			// LD VX, VY
			chip.ldReg(X, Y)
		case 0x1:
			// OR VX, VY
			chip.orReg(X, Y)
		case 0x2:
			// AND VX, VY
			chip.andReg(X, Y)
		case 0x3:
			// XOR VX, VY
			chip.xorReg(X, Y)
		}	
	// 8XY4
	// 8XY5
	// 8XY6
	// 8XY7
	// 8XYE
	// 9XY0
	//
	case 0xA000: // ANNN
		// LD I, addr
		chip.ldAddr(NNN)

	// TODO: BNNN
	// CXNN
	//
	case 0xD000: // DXYN
		// DRW VX, VY, nibble
		chip.drw(X, Y, N)
	}
}

// CLS clear the display
// zero gfx array
func (chip *Chip8) cls() {
	for i := range chip.gfx {
		chip.gfx[i] = 0
	}
}

// RET return from a subroutine
// sets PC to address at top of stack, subtracts 1 from SP
func (chip *Chip8) ret() {
	chip.PC = chip.stack[chip.SP]
	chip.SP--
}

// JP addr
// sets PC to NNN
func (chip *Chip8) jp(NNN uint16) {
	chip.PC = NNN
}

// CALL addr
// call subroutine at nnn
func (chip *Chip8) call(NNN uint16) {
	chip.SP++
	chip.stack[chip.SP] = chip.PC
	chip.PC = NNN
}

// SE VX, byte
// if VX = NN then increment PC
func (chip *Chip8) skipIfEqual(X, NN uint16) {
	if chip.V[X] == byte(NN) {
		chip.PC += 2
	}
}

// SNE VX, byte
// if VX != NN then increment PC
func (chip *Chip8) skipIfNotEqual(X, NN uint16) {
	if chip.V[X] != byte(NN) {
		chip.PC += 2
	}
}

// SE Vx, Vy
// if Vx = Vy then increment PC
func (chip *Chip8) skipIfEqualReg(X, Y uint16) {
	if chip.V[X] == chip.V[Y] {
		chip.PC += 2
	}
}

// LD VX, byte
// set VX = byte
func (chip *Chip8) ldByte(X, NN uint16) {
	chip.V[X] = byte(NN)
}

// ADD VX, byte
// set VX = VX + byte
func (chip *Chip8) addByte(X, NN uint16) {
	chip.V[X] += byte(NN)
}

// LD VX, VY
// set VX = VY
func (chip *Chip8) ldReg(X, Y uint16) {
	chip.V[X] = chip.V[Y]
}

// OR VX, VY
// set VX = VX OR VY
func (chip *Chip8) orReg(X, Y uint16) {
	chip.V[X] |= chip.V[Y]
}

// AND VX, VY
// set VX = VX AND VY
func (chip *Chip8) andReg(X, Y uint16) {
	chip.V[X] &= chip.V[Y]
}

// XOR VX, VY
// set VX = VX XOR VY
func (chip *Chip8) xorReg(X, Y uint16) {
	chip.V[X] ^= chip.V[Y]
}

// LD I, addr
// set I = NNN
func (chip *Chip8) ldAddr(NNN uint16) {
	chip.I = NNN
}

// TODO...

// DRW VX, VY, nibble
// display nbyte sprite starting at memory location I at (VX, VY)
// set VF = collision (1 or 0)
func (chip *Chip8) drw(X, Y, N uint16) {
	// coordinates are modulo the size of display
	// however the actual drawing of the sprite should not wrap
	x := int(chip.V[X] & 63) // set x to VX modulo 64
	y := int(chip.V[Y] & 31) // set y to VY modulo 31

	chip.V[0xF] = 0

	for i := uint16(0); i < N; i++ {
		spriteNbyte := chip.memory[chip.I+i]

		// 8 bit sprite
		for j := 0; j < 8; j++ {
			// bit extraction
			bit := (spriteNbyte >> (7 - j)) & 1

			if bit == 1 {
				screenX := int(x) + j
				screenY := int(y) + int(i)

				// reached edge of screen, stop
				if screenX >= 64 || screenY >= 32 {
					continue // do not warp pixels
				}

				index := screenY*64 + screenX

				// set VF to 1, pixel is being turned off
				if chip.gfx[index] == 1 {
					chip.V[0xF] = 1
				}

				// xor sprite into the screen
				chip.gfx[index] ^= 1
			}
		}
	}
}
