// Package convert handles Broadlink-to-Tuya IR code conversion.
package convert

import (
	"fmt"

	"github.com/diogoaguiar/tuya-ir/codec"
)

// BroadlinkToTuya converts a single Broadlink IR code to Tuya format.
func BroadlinkToTuya(broadlinkCode string) (string, error) {
	timings, err := codec.DecodeBroadlink(broadlinkCode)
	if err != nil {
		return "", err
	}

	return codec.EncodeTuyaBase64(timings), nil
}

// Commands recursively converts all Broadlink IR codes in a SmartIR commands
// map to Tuya format, preserving the nested structure.
func Commands(commands map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range commands {
		switch v := value.(type) {
		case string:
			converted, err := BroadlinkToTuya(v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert code for key '%s': %w", key, err)
			}
			result[key] = converted

		case map[string]interface{}:
			converted, err := Commands(v)
			if err != nil {
				return nil, err
			}
			result[key] = converted

		default:
			result[key] = v
		}
	}

	return result, nil
}
