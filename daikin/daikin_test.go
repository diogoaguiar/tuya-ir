package daikin

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"

	"github.com/diogoaguiar/tuya-ir/codec"
)

// decodeRefBytes extracts Daikin protocol bytes from a Tuya-encoded IR code.
func decodeRefBytes(tuyaCode string) [][]byte {
	compressed, _ := base64.StdEncoding.DecodeString(tuyaCode)
	raw := codec.Decompress(compressed)
	timings := make([]uint16, len(raw)/2)
	for i := range timings {
		timings[i] = binary.LittleEndian.Uint16(raw[i*2 : i*2+2])
	}

	var frames [][]byte
	var bits []int
	i := 0

	flushBits := func() {
		if len(bits) == 0 {
			return
		}
		var fb []byte
		for bi := 0; bi+7 < len(bits); bi += 8 {
			var v byte
			for j := 0; j < 8; j++ {
				v |= byte(bits[bi+j]) << j
			}
			fb = append(fb, v)
		}
		frames = append(frames, fb)
		bits = nil
	}

	for i < len(timings) {
		if timings[i] > 3000 {
			flushBits()
			i += 2
			continue
		}
		if i+1 < len(timings) {
			mark, space := timings[i], timings[i+1]
			if mark > 200 && mark < 600 {
				if space > 20000 {
					flushBits()
					i += 2
					continue
				}
				if space < 1100 {
					bits = append(bits, 0)
				} else {
					bits = append(bits, 1)
				}
				i += 2
			} else {
				i++
			}
		} else {
			i++
		}
	}
	flushBits()
	return frames
}

// TestEncodeFrame_AllCodes validates the Daikin frame encoder against all
// known-working codes from the SmartIR 1109 device file (Daikin BRC4C158).
func TestEncodeFrame_AllCodes(t *testing.T) {
	refFile := "../testdata/1109_tuya_reference.json"
	if _, err := os.Stat(refFile); os.IsNotExist(err) {
		t.Skip("Reference data not found")
	}

	data, err := os.ReadFile(refFile)
	if err != nil {
		t.Fatal(err)
	}

	var refJSON map[string]interface{}
	if err := json.Unmarshal(data, &refJSON); err != nil {
		t.Fatal(err)
	}

	cmds := refJSON["commands"].(map[string]interface{})
	total, matched := 0, 0

	// Test off
	refFrames := decodeRefBytes(cmds["off"].(string))
	genFrame, err := EncodeFrame(ModeOff, "", 0)
	if err != nil {
		t.Fatalf("Failed to encode off: %v", err)
	}
	total++
	if bytesEqual(refFrames[1], genFrame) {
		matched++
	} else {
		t.Logf("MISMATCH off: ref=%x gen=%x", refFrames[1], genFrame)
	}

	// Test all mode/fan/temp combinations
	for _, mode := range []string{ModeCool, ModeHeat, ModeFan, ModeDry} {
		modeMap, ok := cmds[mode].(map[string]interface{})
		if !ok {
			continue
		}
		for _, fan := range []string{FanLow, FanMedium, FanHigh} {
			fanMap, ok := modeMap[fan].(map[string]interface{})
			if !ok {
				continue
			}
			for tempStr, codeVal := range fanMap {
				code, ok := codeVal.(string)
				if !ok {
					continue
				}

				temp := 0
				for _, c := range tempStr {
					temp = temp*10 + int(c-'0')
				}

				total++
				refFrames := decodeRefBytes(code)
				genFrame, err := EncodeFrame(mode, fan, temp)
				if err != nil {
					t.Errorf("Failed to encode %s/%s/%d: %v", mode, fan, temp, err)
					continue
				}

				if bytesEqual(refFrames[1], genFrame) {
					matched++
				} else {
					t.Logf("MISMATCH %s/%s/%d: ref=%x gen=%x", mode, fan, temp, refFrames[1], genFrame)
				}
			}
		}
	}

	t.Logf("Protocol bytes: %d/%d matched", matched, total)

	// Allow the 2 known mismatches (heat/low/16 aux flag, heat/medium/21 source data error)
	if total-matched > 2 {
		t.Errorf("Too many mismatches: %d (expected at most 2)", total-matched)
	}
}

// TestGenerate_OutputFormat verifies that Generate produces valid Tuya base64 output.
func TestGenerate_OutputFormat(t *testing.T) {
	code, err := Generate(ModeCool, FanLow, 23)
	if err != nil {
		t.Fatal(err)
	}

	// Should be valid base64
	compressed, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		t.Fatalf("Invalid base64: %v", err)
	}

	// Should decompress
	raw := codec.Decompress(compressed)
	if len(raw) == 0 {
		t.Fatal("Decompressed to empty")
	}

	// Should be even number of bytes (uint16 pairs)
	if len(raw)%2 != 0 {
		t.Fatalf("Odd byte count: %d", len(raw))
	}

	t.Logf("Generated cool/low/23: %d compressed bytes, %d raw bytes, %d timings",
		len(compressed), len(raw), len(raw)/2)
}

// TestGenerate_Errors tests invalid parameter handling.
func TestGenerate_Errors(t *testing.T) {
	tests := []struct {
		name string
		mode string
		fan  string
		temp int
	}{
		{"invalid mode", "turbo", "low", 23},
		{"invalid fan", "cool", "turbo", 23},
		{"temp too low", "cool", "low", 10},
		{"temp too high", "cool", "low", 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Generate(tt.mode, tt.fan, tt.temp)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
