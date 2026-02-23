// Package daikin generates Daikin AC IR codes from parameters.
//
// Supports the Daikin protocol used by BRC4C158 and compatible remotes.
// The protocol sends two frames: a 7-byte preamble and a 15-byte command
// frame encoding power, mode, temperature, and fan speed.
package daikin

import (
	"fmt"

	"github.com/diogoaguiar/tuya-ir/codec"
)

// IR timing constants (microseconds).
const (
	HeaderMark  = 5025
	HeaderSpace = 2132
	BitMark     = 427
	ZeroSpace   = 670
	OneSpace    = 1736
	FrameGap    = 29358
)

// Operating modes.
const (
	ModeOff      = "off"
	ModeOffCool  = "off_cool"
	ModeOffHeat  = "off_heat"
	ModeOffFan   = "off_fan_only"
	ModeOffDry   = "off_dry"
	ModeOffAuto  = "off_auto"
	ModeCool     = "cool"
	ModeHeat     = "heat"
	ModeFan      = "fan_only"
	ModeDry      = "dry"
	ModeAuto     = "auto"
)

// Fan speeds.
const (
	FanLow    = "low"
	FanMedium = "medium"
	FanHigh   = "high"
)

type modeParams struct {
	byte4 byte // extended mode flag
	byte5 byte // power/mode flags
	byte6 byte // aux flag
	byte7 byte // mode code (bit 0 = power on)
}

var modes = map[string]modeParams{
	ModeOff:  {0x00, 0x43, 0x00, 0x00},
	ModeCool: {0x00, 0x53, 0x00, 0x21},
	ModeHeat: {0x00, 0x53, 0x00, 0x11},
	ModeFan:  {0x00, 0x43, 0x00, 0x01},
	ModeDry:  {0x00, 0x03, 0x04, 0x71},
	ModeAuto: {0x10, 0x73, 0x04, 0x31},
}

// offModeBase maps mode-specific off modes to their base active mode.
// Mode-specific off commands preserve the mode group on the Daikin bus,
// which is required for multi-split systems where the outdoor unit uses
// the master unit's mode to determine the system operating mode.
//
// Protocol: byte 7 bit 0 is the power bit. Mode-specific off clears it
// while preserving the mode code. Byte 5 gets the off flag (bit 5) set,
// and byte 4 gets the extended flag (0x10) set.
var offModeBase = map[string]string{
	ModeOffCool: ModeCool,
	ModeOffHeat: ModeHeat,
	ModeOffFan:  ModeFan,
	ModeOffDry:  ModeDry,
	ModeOffAuto: ModeAuto,
}

var fans = map[string]byte{
	FanLow:    0x16,
	FanMedium: 0x36,
	FanHigh:   0x56,
}

// Generate produces a Tuya-encoded IR code for the given AC parameters.
// Returns a base64 string ready for Zigbee2MQTT ir_code_to_send.
func Generate(mode, fan string, temp int) (string, error) {
	frame, err := EncodeFrame(mode, fan, temp)
	if err != nil {
		return "", err
	}

	// Build preamble. Byte 4 mirrors frame byte 4 (0x10 for extended modes).
	preamble := []byte{0x11, 0xda, 0x17, 0x18, 0x04, 0x00, 0x1e}
	if frame[4] != 0x00 {
		preamble[4] = 0x04 | frame[4]
		// Recalculate preamble checksum
		var sum byte
		for _, b := range preamble[:6] {
			sum += b
		}
		preamble[6] = sum
	}

	timings := FrameToTimings(preamble, frame)
	return codec.EncodeTuyaBase64(timings), nil
}

// EncodeFrame builds the 15-byte Daikin command frame.
func EncodeFrame(mode, fan string, temp int) ([]byte, error) {
	// Mode-specific off: derive from the base active mode.
	baseMode, isSpecificOff := offModeBase[mode]

	var mp modeParams
	if isSpecificOff {
		base, ok := modes[baseMode]
		if !ok {
			return nil, fmt.Errorf("unknown base mode for %q", mode)
		}
		mp = modeParams{
			byte4: base.byte4 | 0x10,        // set extended flag
			byte5: base.byte5 | 0x20,        // set off flag (bit 5)
			byte6: base.byte6,               // preserve aux
			byte7: base.byte7 &^ 0x01,       // clear power bit (bit 0)
		}
	} else {
		var ok bool
		mp, ok = modes[mode]
		if !ok {
			return nil, fmt.Errorf("unknown mode: %q", mode)
		}
	}

	isOff := mode == ModeOff || isSpecificOff

	fanByte, ok := fans[fan]
	if !ok && !isOff && mode != ModeDry {
		return nil, fmt.Errorf("unknown fan speed: %q", fan)
	}

	// Determine temperature and fan bytes based on mode.
	var tempByte byte
	switch {
	case isOff || mode == ModeFan:
		tempByte = 0x10 // fixed
	case mode == ModeDry:
		tempByte = 0x10 // dry ignores temp
		fanByte = 0x56  // dry uses fixed fan
	default:
		if temp < 16 || temp > 32 {
			return nil, fmt.Errorf("temperature %d out of range (16-32)", temp)
		}
		tempByte = byte((temp - 9) * 2)
	}

	if isOff {
		fanByte = 0x16 // fixed for off
	}

	frame := []byte{
		0x11, 0xda, 0x17, 0x18, // header
		mp.byte4, // byte 4: extended mode flag
		mp.byte5, // byte 5: power/mode flags
		mp.byte6, // byte 6: aux flag
		mp.byte7, // byte 7: mode code (bit 0 = power)
		0x00, 0x00, // bytes 8-9
		tempByte, // byte 10: temperature
		fanByte,  // byte 11: fan speed
		0x00, 0x20, // bytes 12-13
		0x00, // byte 14: checksum (computed below)
	}

	// Checksum: sum of bytes 0-13.
	var sum byte
	for _, b := range frame[:14] {
		sum += b
	}
	frame[14] = sum

	return frame, nil
}

// FrameToTimings converts protocol frames into IR mark/space timings (microseconds).
func FrameToTimings(frames ...[]byte) []uint16 {
	var timings []uint16

	for i, frame := range frames {
		// Header
		timings = append(timings, HeaderMark, HeaderSpace)

		// Data bits (LSB first per byte)
		for _, b := range frame {
			for bit := 0; bit < 8; bit++ {
				timings = append(timings, BitMark)
				if b&(1<<bit) != 0 {
					timings = append(timings, OneSpace)
				} else {
					timings = append(timings, ZeroSpace)
				}
			}
		}

		// Trail mark
		timings = append(timings, BitMark)

		// Frame gap (except after last frame)
		if i < len(frames)-1 {
			timings = append(timings, FrameGap)
		}
	}

	return timings
}
