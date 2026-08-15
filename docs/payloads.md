# Payloads

- **Payload 1**: 2026-08-15 12:59:04 (Saturday)
```text
Continuous:
1c00101011230dd4ffff000000001b0000020002000002300000000021090a02010028000a001007ea080f0c3b040600000000000000000000000000000000000000000000000000000000e2

Hex dump format:
1c 00 10 10 11 23 0d d4 ff ff 00 00 00 00 1b 00
00 02 00 02 00 00 02 30 00 00 00 00 21 09 0a 02
01 00 28 00 0a 00 10 07 ea 08 0f 0c 3b 04 06 00
00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00
00 00 00 00 00 00 00 00 00 00 00 e2
```

- **Payload 2**: 2026-08-15 18:59:50 (Saturday)
```text
Continuous:
1c00b05978170dd4ffff000000001b0000020006000002300000000021090a02010028000a001007ea080f123b320600000000000000000000000000000000000000000000000000000000ae

Hex dump format:
1c 00 b0 59 78 17 0d d4 ff ff 00 00 00 00 1b 00
00 02 00 06 00 00 02 30 00 00 00 00 21 09 0a 02
01 00 28 00 0a 00 10 07 ea 08 0f 12 3b 32 06 00
00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00
00 00 00 00 00 00 00 00 00 00 00 ae
```
- **Payload 3**: 2026-06-15 20:55:03 (Monday)
```text
Continuous:
1c00b0d94d270dd4ffff000000001b0000020006000002300000000021090a02010028000a001007ea060f1437030100000000000000000000000000000000000000000000000000000000e6

Hex dump format:
1c 00 b0 d9 4d 27 0d d4 ff ff 00 00 00 00 1b 00
00 02 00 06 00 00 02 30 00 00 00 00 21 09 0a 02
01 00 28 00 0a 00 10 07 ea 06 0f 14 37 03 01 00
00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00
00 00 00 00 00 00 00 00 00 00 00 e6
```

## USB HID interface

The dock (Compx "RapidSync", VID `0x3554` / PID `0xF523`) is a composite HID
device: on macOS it enumerates as 17 separate logical HID collections sharing
that VID/PID (keyboard, mouse, consumer-control, and several vendor-specific
pages), most under one physical interface whose 293-byte report descriptor
declares 9 distinct Report IDs. The time-sync command channel is the
collection with **Usage Page `0xFF08`, Usage `0x0002`, Report ID `0x0A`**,
whose descriptor fragment is:

```
06 08 ff 09 02 a1 01 85 0a 15 00 26 ff 00 75 08 95 27 09 02 81 00 09 02 91 00 c0
```

This declares an **Output** report (`91 00`, not `B1`/Feature) of 1 Report-ID
byte + 39 data bytes = 40 bytes total, confirming both the Output-report
assumption and the `ReportSize = 40` used throughout this codebase. Opening
any *other* collection on this VID/PID (e.g. the keyboard/mouse ones) either
fails outright or writes to the wrong channel — `internal/dock.Device` and
`internal/daemon`'s enumeration filter on this exact UsagePage/Usage pair to
avoid both.

Here is a Gemini extract from the first payload:

|    Offset     | Field Name           |  Data Type  | Value (Hex) | Decoded Example | Description                                                     |
|:-------------:|:---------------------|:-----------:|:-----------:|:---------------:|:----------------------------------------------------------------|
|    `0x00`     | **Report ID**        |   `uint8`   |   `0x0A`    |       10        | Output Report identifier                                        |
|    `0x01`     | **Command Category** |   `uint8`   |   `0x00`    |        0        | Sub-system / category code                                      |
|    `0x02`     | **Command Opcode**   |   `uint8`   |   `0x10`    |       16        | Time synchronization opcode                                     |
|    `0x03`     | **Year (High Byte)** |   `uint8`   |   `0x07`    |      2026       | Year in Big-Endian (`0x07EA` = `2026`)                          |
|    `0x04`     | **Year (Low Byte)**  |   `uint8`   |   `0xEA`    |                 |                                                                 |
|    `0x05`     | **Month**            |   `uint8`   |   `0x08`    |        8        | Month (`1`–`12`, `0x08` = August)                               |
|    `0x06`     | **Day**              |   `uint8`   |   `0x0F`    |       15        | Day of the month (`1`–`31`, `0x0F` = 15)                        |
|    `0x07`     | **Hour**             |   `uint8`   |   `0x0C`    |       12        | Hour in 24-hour format (`0`–`23`, `0x0C` = 12 PM)               |
|    `0x08`     | **Minute**           |   `uint8`   |   `0x3B`    |       59        | Minute (`0`–`59`, `0x3B` = 59)                                  |
|    `0x09`     | **Second**           |   `uint8`   |   `0x04`    |       04        | Second (`0`–`59`, `0x04` = 4s)                                  |
|    `0x0A`     | **Day of Week**      |   `uint8`   |   `0x06`    |        6        | Day of week (`1` = Monday ... `6` = Saturday, `7`/`0` = Sunday) |
| `0x0B`–`0x26` | **Zero Padding**     | `bytes[28]` |  `0x00...`  |        0        | Reserved / null padding (28 zero bytes)                         |
|    `0x27`     | **Checksum / CRC**   |   `uint8`   |   `0xE2`    |       226       | Packet validation checksum / footer                             |
