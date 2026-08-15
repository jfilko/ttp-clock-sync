package dock

import "time"

const ReportSize = 40

const (
	reportID        = 0x0A
	commandCategory = 0x00
	timeSyncOpcode  = 0x10
	checksumTarget  = 0x55 // sum of all ReportSize bytes must equal this, mod 256
)

// Device identifies one supported dock model by its USB HID VendorID/ProductID
// and, since these docks expose many logical HID interfaces (keyboard, mouse,
// and several vendor-specific report collections) under that same VID/PID,
// the specific UsagePage/Usage pair whose report descriptor declares the
// time-sync report's Report ID (0x0A) as an Output report.
type Device struct {
	VendorID  uint16
	ProductID uint16
	UsagePage uint16
	Usage     uint16
}

// SupportedDevices lists every dock model this daemon knows how to talk to.
// Add more entries here to support additional models.
var SupportedDevices = []Device{
	{VendorID: 13652, ProductID: 62755, UsagePage: 0xFF08, Usage: 0x0002}, // Teevolution RapidSync 8k (0x3554 / 0xF523)
}

// BuildTimeSyncReport encodes t's calendar fields (as t.Year/Month/.../Weekday
// report them — t's own Location, not necessarily forced to Local) into a
// 40-byte time-sync HID report, including the trailing checksum byte.
func BuildTimeSyncReport(t time.Time) [ReportSize]byte {
	var report [ReportSize]byte
	report[0x00] = reportID
	report[0x01] = commandCategory
	report[0x02] = timeSyncOpcode

	year := uint16(t.Year())
	report[0x03] = byte(year >> 8)
	report[0x04] = byte(year)
	report[0x05] = byte(t.Month())
	report[0x06] = byte(t.Day())
	report[0x07] = byte(t.Hour())
	report[0x08] = byte(t.Minute())
	report[0x09] = byte(t.Second())
	report[0x0A] = dayOfWeek(t.Weekday())
	// 0x0B..0x26 left as zero padding.
	report[0x27] = checksum(report)
	return report
}

// dayOfWeek converts Go's time.Weekday (0=Sunday..6=Saturday) to the dock's
// encoding (1=Monday..6=Saturday, 7=Sunday).
func dayOfWeek(w time.Weekday) byte {
	if w == time.Sunday {
		return 7
	}
	return byte(w)
}

// checksum computes byte 0x27 such that summing all ReportSize bytes, mod
// 256, equals checksumTarget — derived from and verified against 3 sample
// payloads in docs/payloads.md. Go's uint8 subtraction wraps mod 256, so this
// is equivalent to (checksumTarget - sum) & 0xFF.
func checksum(report [ReportSize]byte) byte {
	var sum byte
	for i := 0; i < ReportSize-1; i++ {
		sum += report[i]
	}
	return checksumTarget - sum
}
