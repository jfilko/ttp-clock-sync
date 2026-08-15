# Payloads

## [Payload 1](payloads/payload_1.png)

This payload has been captured with Wireshark by running Teevolution app, changing time in Windows and waiting for the
dock to change to the correct date. Between updating time and dongle showing correct time, there has been only 1
outgoing payload:

```text
1c 00 10 10 11 23 0d d4 ff ff 00 00 00 00 1b 00
00 02 00 02 00 00 02 30 00 00 00 00 21 09 0a 02
01 00 28 00 0a 00 10 07 ea 08 0f 0c 3b 04 06 00
00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00
00 00 00 00 00 00 00 00 00 00 00 e2
```

Gemini extracted this info from it:

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
