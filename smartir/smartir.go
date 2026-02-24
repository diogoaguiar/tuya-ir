// Package smartir handles SmartIR JSON device code files.
package smartir

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/diogoaguiar/irx/format"
)

// File represents a SmartIR JSON device code file.
// Uses a raw map internally to preserve all fields, including unknown ones.
type File struct {
	raw map[string]interface{}
}

// ReadFile reads and parses a SmartIR JSON file.
func ReadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &File{raw: raw}, nil
}

// CommandsEncoding returns the commandsEncoding field value.
func (f *File) CommandsEncoding() string {
	if v, ok := f.raw["commandsEncoding"].(string); ok {
		return v
	}
	return ""
}

// IsBroadlink returns true if the file uses Broadlink encoding.
func (f *File) IsBroadlink() bool {
	return f.CommandsEncoding() == "Base64"
}

// IsRaw returns true if the file already uses Raw/Tuya encoding.
func (f *File) IsRaw() bool {
	return f.CommandsEncoding() == "Raw"
}

// Convert converts all IR codes in the file from one format to another.
func (f *File) Convert(dec format.Decoder, enc format.Encoder) error {
	commands, ok := f.raw["commands"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid 'commands' field")
	}

	converted, err := convertCommands(commands, dec, enc)
	if err != nil {
		return fmt.Errorf("failed to convert commands: %w", err)
	}

	f.raw["commands"] = converted
	return nil
}

// ConvertToTuya converts all Broadlink IR codes to Tuya format and updates metadata.
func (f *File) ConvertToTuya(dec format.Decoder, enc format.Encoder) error {
	if !f.IsBroadlink() {
		return fmt.Errorf("file is not in Broadlink format (commandsEncoding: %s)", f.CommandsEncoding())
	}

	if err := f.Convert(dec, enc); err != nil {
		return err
	}

	f.raw["commandsEncoding"] = "Raw"
	f.raw["supportedController"] = "MQTT"
	return nil
}

// WriteJSON writes the SmartIR file as pretty-printed JSON to the given path.
func (f *File) WriteJSON(path string) error {
	data, err := json.MarshalIndent(f.raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// MarshalJSON returns the JSON representation of the file.
func (f *File) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.raw)
}

// convertCommands recursively converts all string IR codes in a nested map.
func convertCommands(commands map[string]interface{}, dec format.Decoder, enc format.Encoder) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range commands {
		switch v := value.(type) {
		case string:
			timings, err := dec.Decode(v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert code for key '%s': %w", key, err)
			}
			encoded, err := enc.Encode(timings)
			if err != nil {
				return nil, fmt.Errorf("failed to encode code for key '%s': %w", key, err)
			}
			result[key] = encoded

		case map[string]interface{}:
			converted, err := convertCommands(v, dec, enc)
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
