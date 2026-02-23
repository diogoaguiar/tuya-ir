// Package daikin generates Daikin AC IR codes from parameters.
//
// Supports the Daikin protocol used by BRC4C158 and compatible remotes.
// The protocol sends two frames: a fixed preamble and a 15-byte command
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
	ModeOff    = "off"
	ModeOffCool = "off_cool"
	ModeOffHeat = "off_heat"
	ModeOffFan  = "off_fan_only"
	ModeOffDry  = "off_dry"
	ModeCool    = "cool"
	ModeHeat    = "heat"
	ModeFan     = "fan_only"
	ModeDry     = "dry"
)

// Fan speeds.
const (
	FanLow    = "low"
	FanMedium = "medium"
	FanHigh   = "high"
)

type modeParams struct {
	byte5 byte // power/mode flags
	byte6 byte // aux flag
	byte7 byte // mode code
}

var modes = map[string]modeParams{
	ModeOff:     {0x43, 0x00, 0x00},
	ModeOffCool: {0x53, 0x00, 0x00},
	ModeOffHeat: {0x53, 0x00, 0x00},
	ModeOffFan:  {0x43, 0x00, 0x00},
	ModeOffDry:  {0x03, 0x04, 0x00},
	ModeCool:    {0x53, 0x00, 0x21},
	ModeHeat:    {0x53, 0x00, 0x11},
	ModeFan:     {0x43, 0x00, 0x01},
	ModeDry:     {0x03, 0x04, 0x71},
}

var fans = map[string]byte{
	FanLow:    0x16,
	FanMedium: 0x36,
	FanHigh:   0x56,
}

// preamble is the fixed first frame sent before every command.
var preamble = []byte{0x11, 0xda, 0x17, 0x18, 0x04, 0x00, 0x1e}

// Generate produces a Tuya-encoded IR code for the given AC parameters.
// Returns a base64 string ready for Zigbee2MQTT ir_code_to_send.
func Generate(mode, fan string, temp int) (string, error) {
	frame, err := EncodeFrame(mode, fan, temp)
	if err != nil {
		return "", err
	}

	timings := FrameToTimings(preamble, frame)
	return codec.EncodeTuyaBase64(timings), nil
}

// EncodeFrame builds the 15-byte Daikin command frame.
func EncodeFrame(mode, fan string, temp int) ([]byte, error) {
	mp, ok := modes[mode]
	if !ok {
		return nil, fmt.Errorf("unknown mode: %q", mode)
	}

	fanByte, ok := fans[fan]
	if !ok && mode != ModeOff && mode != ModeDry {
		return nil, fmt.Errorf("unknown fan speed: %q", fan)
	}

	// Determine temperature and fan bytes based on mode
	var tempByte byte
	switch mode {
	case ModeOff, ModeOffCool, ModeOffHeat, ModeOffFan, ModeOffDry, ModeFan:
		tempByte = 0x10 // fixed
	case ModeDry:
		tempByte = 0x10 // dry ignores temp
		fanByte = 0x56  // dry ignores fan
	default:
		if temp < 16 || temp > 32 {
			return nil, fmt.Errorf("temperature %d out of range (16-32)", temp)
		}
		tempByte = byte((temp - 9) * 2)
	}

	switch mode {
	case ModeOff, ModeOffCool, ModeOffHeat, ModeOffFan, ModeOffDry:
		fanByte = 0x16 // fixed for off
	}

	frame := []byte{
		0x11, 0xda, 0x17, 0x18, // header
		0x00,      // byte 4
		mp.byte5,  // byte 5: power/mode flags
		mp.byte6,  // byte 6: aux flag
		mp.byte7,  // byte 7: mode code
		0x00, 0x00, // bytes 8-9
		tempByte, // byte 10: temperature
		fanByte,  // byte 11: fan speed
		0x00, 0x20, // bytes 12-13
		0x00, // byte 14: checksum (computed below)
	}

	// Checksum: sum of bytes 0-13
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
