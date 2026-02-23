# tuya-ir

Toolkit for working with Tuya-format IR codes used by Zigbee2MQTT IR blasters (ZS06, UFO-R11).

## Features

- **Convert** SmartIR device code files from Broadlink (Base64) to Tuya (Raw) format
- **Generate** Daikin AC IR codes dynamically from mode/fan/temperature parameters

## Install

```bash
go install github.com/diogoaguiar/tuya-ir/cmd/tuya-ir@latest
```

## Usage

### Convert SmartIR files

```bash
# Convert a SmartIR Broadlink JSON to Tuya format
tuya-ir convert 1109.json 1109_tuya.json

# Write to stdout
tuya-ir convert 1109.json
```

### Generate Daikin IR codes

```bash
# Generate a cool mode code
tuya-ir generate daikin --mode cool --fan low --temp 23

# Turn off
tuya-ir generate daikin --mode off

# Mode-specific off (for multi-split systems)
tuya-ir generate daikin --mode off_cool

# Pipe directly to MQTT
tuya-ir generate daikin --mode cool --fan high --temp 21 | \
  mosquitto_pub -t 'zigbee2mqtt/IR Blaster/set' \
    -m "{\"ir_code_to_send\": \"$(cat)\"}"
```

### Supported Daikin parameters

| Parameter | Values |
|-----------|--------|
| `--mode` | `off`, `off_cool`, `off_heat`, `off_fan_only`, `off_dry`, `cool`, `heat`, `fan_only`, `dry` |
| `--fan` | `low`, `medium`, `high` (default: `low`) |
| `--temp` | `16`-`32` (default: `23`, ignored for off/dry/fan_only) |

## Development

```bash
make build    # Build to bin/tuya-ir
make test     # Run all tests
make install  # Install to $GOPATH/bin
```

## Project structure

```
codec/    - Tuya compression/encoding, Broadlink decoding
convert/  - Broadlink-to-Tuya IR code conversion
daikin/   - Daikin AC protocol encoder
smartir/  - SmartIR JSON file handling
```
