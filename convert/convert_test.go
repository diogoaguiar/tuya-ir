package convert

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/diogoaguiar/tuya-ir/codec"
)

func TestBroadlinkToTuya_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorText   string
	}{
		{"Empty string", "", true, "empty"},
		{"Invalid base64", "Not!Valid@Base64", true, "invalid base64"},
		{"Too short", "JgA=", true, "too short"},
		{"Whitespace handling", "  JgBsAaVGDDoMFw4WDBcPOAwXDhYOFQ4WDTkNFww7DDoNFw06DDoNOg06DToMFw45DRcNFg4VDhYOFQ0XDTkNOg0XDRYOFQ4WDhUOOQ0WDRcOFQ4WDxQPFQwXDhUOFg4VDhYNFg4VDRcMOw05DToNOg0WDxUPFA4AA8SmRQ06DBcPFQ0WDjkMFw4WDBcOFgw6DRcNOgw6DRcOOQw6DToNOg06DRYPOA0WDRcNFg4WDBcNFw44DToNFw0WDhUNFw8UDxUNFg0WDxUOFQ4WDDoNOg0XDhUOFg0WDToNFg4WDRYPFA4WDRYNFw4VDxQOFg4VDhYMFw4VDhYOFQ4WDBgNFg4VDhUOFgwXDhYMFw4VDxUOFQ4WDRYNFg4WDhUNFw0WDhUPFQw7DBcNFwwXDhUOOQ45DRYPOA0XDRYOFQ4WDhUOFg0WDhYNFg4VDxUNFg4VDhYOFQ4WDToMFw4VDjkNOg0WDRcOFQ45DRYOOQ0ADQUAAAAAAAAAAAAA  ", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BroadlinkToTuya(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorText)
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorText)) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if result == "" {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

// TestGoldenReference validates Go converter output against Python reference.
func TestGoldenReference(t *testing.T) {
	testFiles := []struct {
		broadlink string
		reference string
	}{
		{"../testdata/1109.json", "../testdata/1109_tuya_reference.json"},
		{"../testdata/1116.json", "../testdata/1116_tuya_reference.json"},
	}

	for _, tf := range testFiles {
		t.Run(tf.broadlink, func(t *testing.T) {
			if _, err := os.Stat(tf.broadlink); os.IsNotExist(err) {
				t.Skipf("Test data not found: %s", tf.broadlink)
			}
			if _, err := os.Stat(tf.reference); os.IsNotExist(err) {
				t.Skipf("Reference data not found: %s", tf.reference)
			}

			srcData, _ := os.ReadFile(tf.broadlink)
			refData, _ := os.ReadFile(tf.reference)

			var srcJSON, refJSON map[string]interface{}
			json.Unmarshal(srcData, &srcJSON)
			json.Unmarshal(refData, &refJSON)

			srcCommands := srcJSON["commands"].(map[string]interface{})
			goConverted, err := Commands(srcCommands)
			if err != nil {
				t.Fatalf("Go conversion failed: %v", err)
			}

			refCommands := refJSON["commands"].(map[string]interface{})

			var compared, matched int
			var compare func(goMap, refMap map[string]interface{}, path string)
			compare = func(goMap, refMap map[string]interface{}, path string) {
				for key, goVal := range goMap {
					refVal, ok := refMap[key]
					if !ok {
						continue
					}
					switch gv := goVal.(type) {
					case string:
						rv := refVal.(string)
						compared++
						goRaw := codec.Decompress(mustDecodeB64(gv))
						refRaw := codec.Decompress(mustDecodeB64(rv))
						if bytes.Equal(goRaw, refRaw) {
							matched++
						} else {
							t.Errorf("Raw IR mismatch at %s/%s", path, key)
						}
					case map[string]interface{}:
						compare(gv, refVal.(map[string]interface{}), path+"/"+key)
					}
				}
			}
			compare(goConverted, refCommands, "")
			t.Logf("Compared %d codes, %d matched", compared, matched)

			if matched != compared {
				t.Errorf("%d/%d codes mismatched", compared-matched, compared)
			}
		})
	}
}

func mustDecodeB64(s string) []byte {
	b, _ := base64.StdEncoding.DecodeString(s)
	return b
}
