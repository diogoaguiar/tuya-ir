package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/diogoaguiar/tuya-ir/codec"
)

func main() {
	for _, code := range os.Args[1:] {
		compressed, _ := base64.StdEncoding.DecodeString(code)
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

		for fi, frame := range frames {
			fmt.Printf("Frame %d: ", fi)
			for _, b := range frame {
				fmt.Printf("%02x ", b)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}
