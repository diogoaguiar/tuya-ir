// Package codec provides Tuya IR compression/encoding and Broadlink decoding.
package codec

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

const (
	// TuyaWindowSize is the sliding window size for LZ-style compression (8KB).
	TuyaWindowSize = 1 << 13

	// TuyaMaxMatchLength is the maximum length of a matched sequence.
	TuyaMaxMatchLength = 256 + 9
)

// PackTimings converts microsecond durations to a little-endian uint16 byte stream.
func PackTimings(timings []uint16) []byte {
	buf := new(bytes.Buffer)
	for _, v := range timings {
		binary.Write(buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

// Compress applies Tuya LZ-style compression to raw data.
func Compress(data []byte) []byte {
	out := new(bytes.Buffer)

	blockStart := 0
	pos := 0

	for pos < len(data) {
		bestLength, bestDistance := findBestMatch(data, pos)

		if bestLength >= 3 {
			emitLiteralBlocks(out, data[blockStart:pos])
			emitDistanceBlock(out, bestLength, bestDistance)
			pos += bestLength
			blockStart = pos
		} else {
			pos++
		}
	}

	emitLiteralBlocks(out, data[blockStart:pos])
	return out.Bytes()
}

// Decompress reverses Tuya LZ-style compression.
func Decompress(data []byte) []byte {
	var out []byte
	pos := 0

	for pos < len(data) {
		b := data[pos]
		lengthBits := (b >> 5) & 0x07

		if lengthBits == 0 {
			litLen := int(b&0x1f) + 1
			pos++
			out = append(out, data[pos:pos+litLen]...)
			pos += litLen
		} else {
			distHigh := int(b & 0x1f)
			pos++

			length := int(lengthBits) + 2
			if lengthBits == 7 {
				extra := int(data[pos])
				pos++
				length = 9 + extra
			}

			distLow := int(data[pos])
			pos++
			distance := (distHigh<<8 | distLow) + 1

			for i := 0; i < length; i++ {
				out = append(out, out[len(out)-distance])
			}
		}
	}

	return out
}

// EncodeTuyaBase64 compresses raw IR timing data and returns a base64 string
// ready for Tuya IR blasters (via Zigbee2MQTT).
func EncodeTuyaBase64(timings []uint16) string {
	raw := PackTimings(timings)
	compressed := Compress(raw)
	return base64.StdEncoding.EncodeToString(compressed)
}

func findBestMatch(data []byte, pos int) (int, int) {
	bestLength := 0
	bestDistance := 0

	windowStart := pos - TuyaWindowSize
	if windowStart < 0 {
		windowStart = 0
	}

	for distance := 1; distance <= pos-windowStart; distance++ {
		comparePos := pos - distance
		length := 0
		maxLength := TuyaMaxMatchLength
		if pos+maxLength > len(data) {
			maxLength = len(data) - pos
		}

		for length < maxLength && data[pos+length] == data[comparePos+length] {
			length++
		}

		if length > bestLength {
			bestLength = length
			bestDistance = distance
		}
	}

	return bestLength, bestDistance
}

func emitLiteralBlocks(out *bytes.Buffer, data []byte) {
	for i := 0; i < len(data); i += 32 {
		end := i + 32
		if end > len(data) {
			end = len(data)
		}
		emitLiteralBlock(out, data[i:end])
	}
}

func emitLiteralBlock(out *bytes.Buffer, data []byte) {
	length := len(data) - 1
	if length < 0 || length >= (1<<5) {
		panic(fmt.Sprintf("invalid literal block length: %d", len(data)))
	}
	out.WriteByte(byte(length))
	out.Write(data)
}

func emitDistanceBlock(out *bytes.Buffer, length int, distance int) {
	distance--
	length -= 2

	var block []byte

	if length >= 7 {
		block = []byte{
			byte(7<<5 | distance>>8),
			byte(length - 7),
			byte(distance & 0xFF),
		}
	} else {
		block = []byte{
			byte(length<<5 | distance>>8),
			byte(distance & 0xFF),
		}
	}

	out.Write(block)
}
