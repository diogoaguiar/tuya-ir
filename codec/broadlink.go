package codec

import (
	"encoding/base64"
	"fmt"
	"math"
	"strings"
)

const (
	// BroadlinkUnit is the time unit used by Broadlink devices (~32.84 microseconds).
	BroadlinkUnit = 269.0 / 8192.0
)

// DecodeBroadlink converts a Broadlink base64 IR code to microsecond timings.
func DecodeBroadlink(broadlinkCode string) ([]uint16, error) {
	broadlinkCode = strings.TrimSpace(broadlinkCode)
	if broadlinkCode == "" {
		return nil, fmt.Errorf("empty Broadlink code")
	}

	decoded, err := base64.StdEncoding.DecodeString(broadlinkCode)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	hexString := fmt.Sprintf("%x", decoded)

	durations, err := parseBroadlinkDurations(hexString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Broadlink format: %w", err)
	}

	if len(durations) == 0 {
		return nil, fmt.Errorf("no IR durations found in Broadlink code")
	}

	result := make([]uint16, 0, len(durations))
	for _, duration := range durations {
		microseconds := int(math.Ceil(float64(duration) / BroadlinkUnit))
		if microseconds < 65535 {
			result = append(result, uint16(microseconds))
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("all durations filtered out (too large for uint16)")
	}

	return result, nil
}

func parseBroadlinkDurations(hexString string) ([]int, error) {
	if len(hexString) < 8 {
		return nil, fmt.Errorf("invalid Broadlink format: too short (min 8 hex chars)")
	}

	lengthHex := hexString[6:8] + hexString[4:6]
	length64, err := parseHexToInt(lengthHex)
	if err != nil {
		return nil, fmt.Errorf("invalid payload length: %w", err)
	}
	length := int(length64)

	var durations []int
	i := 8

	for i < length*2+8 {
		if i+2 > len(hexString) {
			break
		}

		hexValue := hexString[i : i+2]

		if hexValue == "00" {
			if i+6 > len(hexString) {
				return nil, fmt.Errorf("truncated extended duration at position %d", i)
			}
			hexValue = hexString[i+2:i+4] + hexString[i+4:i+6]
			i += 4
		}

		val, err := parseHexToInt(hexValue)
		if err != nil {
			return nil, fmt.Errorf("invalid hex value '%s' at position %d: %w", hexValue, i, err)
		}

		durations = append(durations, int(val))
		i += 2
	}

	return durations, nil
}

func parseHexToInt(hexStr string) (int64, error) {
	var val int64
	_, err := fmt.Sscanf(hexStr, "%x", &val)
	return val, err
}
