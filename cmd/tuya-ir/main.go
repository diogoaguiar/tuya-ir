package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/diogoaguiar/tuya-ir/daikin"
	"github.com/diogoaguiar/tuya-ir/smartir"
)

const usage = `Usage: tuya-ir <command> [args]

Commands:
  convert   Convert a SmartIR file from Broadlink to Tuya format
  generate  Generate an IR code from parameters

Run 'tuya-ir <command> -h' for command-specific help.`

const convertUsage = `Usage: tuya-ir convert <input.json> [output.json]

  Converts a SmartIR device code file from Broadlink (Base64) format to
  Tuya (Raw/MQTT) format for use with Zigbee2MQTT IR blasters.

  If output.json is omitted, writes to stdout.`

const generateUsage = `Usage: tuya-ir generate daikin --mode <mode> [--fan <fan>] [--temp <temp>]

  Generates a Tuya-encoded IR code for a Daikin AC.

  Modes: off, off_cool, off_heat, off_fan_only, off_dry, cool, heat, fan_only, dry
  Fan:   low, medium, high (default: low)
  Temp:  16-32 (default: 23, ignored for off/dry/fan_only)

  off_cool/off_heat: mode-specific off commands that preserve the cool/heat
  mode group flag (byte5=0x53) for Daikin multi-split systems. Prevents the
  outdoor unit from dropping to fan mode when the master unit turns off.

Examples:
  tuya-ir generate daikin --mode cool --fan low --temp 23
  tuya-ir generate daikin --mode off
  tuya-ir generate daikin --mode off_cool`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "convert":
		cmdConvert(os.Args[2:])
	case "generate":
		cmdGenerate(os.Args[2:])
	case "-h", "--help":
		fmt.Println(usage)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s\n", os.Args[1], usage)
		os.Exit(1)
	}
}

func cmdConvert(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, convertUsage)
		if len(args) > 0 {
			os.Exit(0)
		}
		os.Exit(1)
	}

	inputPath := args[0]
	var outputPath string
	if len(args) >= 2 {
		outputPath = args[1]
	}

	f, err := smartir.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	if f.IsRaw() {
		fmt.Fprintln(os.Stderr, "Warning: file is already in Raw/Tuya format, no conversion needed.")
		os.Exit(0)
	}

	if !f.IsBroadlink() {
		fmt.Fprintf(os.Stderr, "Error: unsupported commandsEncoding %q (expected \"Base64\")\n", f.CommandsEncoding())
		os.Exit(1)
	}

	if err := f.ConvertToTuya(); err != nil {
		fmt.Fprintf(os.Stderr, "Error converting: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" {
		if err := f.WriteJSON(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Converted %s -> %s\n", inputPath, outputPath)
	} else {
		data, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	}
}

func cmdGenerate(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, generateUsage)
		if len(args) > 0 {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if args[0] != "daikin" {
		fmt.Fprintf(os.Stderr, "Unknown device: %s (supported: daikin)\n", args[0])
		os.Exit(1)
	}

	// Parse flags
	mode := ""
	fan := "low"
	temp := 23

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--mode":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --mode requires a value")
				os.Exit(1)
			}
			mode = args[i]
		case "--fan":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --fan requires a value")
				os.Exit(1)
			}
			fan = args[i]
		case "--temp":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --temp requires a value")
				os.Exit(1)
			}
			var err error
			temp, err = strconv.Atoi(args[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid temperature: %s\n", args[i])
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n\n%s\n", args[i], generateUsage)
			os.Exit(1)
		}
	}

	if mode == "" {
		fmt.Fprintln(os.Stderr, "Error: --mode is required")
		fmt.Fprintln(os.Stderr, generateUsage)
		os.Exit(1)
	}

	code, err := daikin.Generate(mode, fan, temp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(code)
}
