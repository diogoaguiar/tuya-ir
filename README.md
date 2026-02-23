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
| `--mode` | `off`, `off_cool`, `off_heat`, `off_fan_only`, `off_dry`, `off_auto`, `cool`, `heat`, `fan_only`, `dry`, `auto` |
| `--fan` | `low`, `medium`, `high` (default: `low`) |
| `--temp` | `16`-`32` (default: `23`, ignored for off/dry/fan_only) |

### Daikin IR protocol (BRC4C160 remote / FXAA63AUV1B indoor unit)

Two frames per transmission. Frame 0 is a 7-byte preamble, Frame 1 is the 15-byte command.

**Frame 1 byte map:**

| Byte | Field | Description |
|------|-------|-------------|
| 0-3 | Header | `11 da 17 18` (fixed) |
| 4 | Extended flag | `0x00` standard, `0x10` for auto and mode-specific off |
| 5 | Mode group | `0x43` fan/off, `0x53` cool/heat, `0x03` dry, `0x73` auto |
| 6 | Aux flag | `0x00` usually, `0x04` for dry and auto |
| 7 | Mode code | Bit 0 = power. `0x01`=fan, `0x11`=heat, `0x21`=cool, `0x31`=auto, `0x71`=dry |
| 10 | Temperature | `(temp - 9) * 2` |
| 11 | Fan speed | `0x16`=low, `0x36`=medium, `0x56`=high |
| 14 | Checksum | `sum(bytes[0:14]) & 0xFF` |

**All modes:**

| Mode | byte4 | byte5 | byte6 | byte7 |
|------|-------|-------|-------|-------|
| off | `0x00` | `0x43` | `0x00` | `0x00` |
| fan_only | `0x00` | `0x43` | `0x00` | `0x01` |
| cool | `0x00` | `0x53` | `0x00` | `0x21` |
| heat | `0x00` | `0x53` | `0x00` | `0x11` |
| dry | `0x00` | `0x03` | `0x04` | `0x71` |
| auto | `0x10` | `0x73` | `0x04` | `0x31` |

### Mode-specific off (multi-split systems)

In Daikin multi-split systems, the outdoor unit determines the operating mode from the indoor units' bus state. A generic off uses byte5=`0x43` (fan/off group), causing the outdoor unit to drop to fan mode — other units can no longer cool or heat.

Mode-specific off commands preserve the mode group. Derived from the base active mode:
- **Byte 4**: set `0x10` (extended flag)
- **Byte 5**: set bit 5 (`0x20`) — off flag
- **Byte 7**: clear bit 0 — power off

| Command | byte4 | byte5 | byte7 | Derived from |
|---------|-------|-------|-------|--------------|
| `off` | `0x00` | `0x43` | `0x00` | — |
| `off_cool` | `0x10` | `0x73` | `0x20` | cool |
| `off_heat` | `0x10` | `0x73` | `0x10` | heat |
| `off_fan_only` | `0x10` | `0x63` | `0x00` | fan |
| `off_dry` | `0x10` | `0x23` | `0x70` | dry |
| `off_auto` | `0x10` | `0x73` | `0x30` | auto |

The preamble (frame 0) byte 4 mirrors the command byte 4: `0x04` normally, `0x14` when extended.

SmartIR maps these to `off_<mode>` command keys and automatically sends the correct one when the user turns off.

## Development

```bash
make build    # Build to bin/tuya-ir
make test     # Run all tests
make install  # Install to $GOPATH/bin
```

## Project structure

```
cmd/tuya-ir/  - Main CLI tool
cmd/decode/   - IR code decoder (Tuya base64 → protocol bytes)
codec/        - Tuya compression/encoding, Broadlink decoding
convert/      - Broadlink-to-Tuya IR code conversion
daikin/       - Daikin AC protocol encoder
smartir/      - SmartIR JSON file handling
```
