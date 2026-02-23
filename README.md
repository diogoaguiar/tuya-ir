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

### Mode-specific off (multi-split systems)

In Daikin multi-split systems, the outdoor unit determines the operating mode (cool/heat) from the indoor units' bus state. A generic off command uses byte5=`0x43` (fan/off group), which causes the outdoor unit to drop to fan mode — other indoor units can then no longer cool or heat.

Mode-specific off commands (`off_cool`, `off_heat`, etc.) preserve the mode group on the bus so other units can continue operating. The protocol differences from a generic off:

| Byte | Generic off | Mode-specific off |
|------|-------------|-------------------|
| Frame 0, byte 4 | `0x04` | `0x14` (set bit 4) |
| Frame 1, byte 4 | `0x00` | `0x10` (set bit 4) |
| Frame 1, byte 5 | `0x43` | base mode byte5 \| `0x20` (set off flag) |
| Frame 1, byte 7 | `0x00` | base mode byte7 & `0xFE` (clear power bit) |

Byte 7 bit 0 is the power bit. Mode-specific off clears it while preserving the mode code:

| Command | byte5 | byte7 | Derived from |
|---------|-------|-------|--------------|
| `off` | `0x43` | `0x00` | — |
| `off_cool` | `0x73` | `0x20` | cool: `0x53`/`0x21` |
| `off_heat` | `0x73` | `0x10` | heat: `0x53`/`0x11` |
| `off_fan_only` | `0x63` | `0x00` | fan: `0x43`/`0x01` |
| `off_dry` | `0x23` | `0x70` | dry: `0x03`/`0x71` |

These map directly to SmartIR's `off_<mode>` command keys — SmartIR automatically sends the correct off command based on the current mode when the user turns off the AC.

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
