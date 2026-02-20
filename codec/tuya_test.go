package codec

import (
	"bytes"
	"testing"
)

func TestCompressDecompressRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"short", []byte("hello world")},
		{"repeated", []byte("abcabcabcabcabcabcabcabc")},
		{"binary", func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed := Compress(tt.data)
			decompressed := Decompress(compressed)

			if !bytes.Equal(decompressed, tt.data) {
				t.Fatalf("roundtrip mismatch: got %d bytes, want %d bytes", len(decompressed), len(tt.data))
			}
		})
	}
}

func TestEmitDistanceBlockFormat(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		distance int
		want     []byte
	}{
		{"short (len=3, dist=1)", 3, 1, []byte{0x20, 0x00}},
		{"short (len=8, dist=5)", 8, 5, []byte{0xC0, 0x04}},
		{"long (len=9, dist=1)", 9, 1, []byte{0xE0, 0x00, 0x00}},
		{"long (len=20, dist=100)", 20, 100, []byte{0xE0, 0x0B, 0x63}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			emitDistanceBlock(out, tt.length, tt.distance)
			if !bytes.Equal(out.Bytes(), tt.want) {
				t.Errorf("got %v, want %v", out.Bytes(), tt.want)
			}
		})
	}
}
