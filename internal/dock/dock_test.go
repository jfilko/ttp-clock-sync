package dock

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestBuildTimeSyncReport_SamplePayloads(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		hex  string // full continuous hex dump from docs/payloads.md
	}{
		{
			name: "payload 1: 2026-08-15 12:59:04 Saturday",
			time: time.Date(2026, time.August, 15, 12, 59, 4, 0, time.UTC),
			hex:  "1c00101011230dd4ffff000000001b0000020002000002300000000021090a02010028000a001007ea080f0c3b040600000000000000000000000000000000000000000000000000000000e2",
		},
		{
			name: "payload 2: 2026-08-15 18:59:50 Saturday",
			time: time.Date(2026, time.August, 15, 18, 59, 50, 0, time.UTC),
			hex:  "1c00b05978170dd4ffff000000001b0000020006000002300000000021090a02010028000a001007ea080f123b320600000000000000000000000000000000000000000000000000000000ae",
		},
		{
			name: "payload 3: 2026-06-15 20:55:03 Monday",
			time: time.Date(2026, time.June, 15, 20, 55, 3, 0, time.UTC),
			hex:  "1c00b0d94d270dd4ffff000000001b0000020006000002300000000021090a02010028000a001007ea060f1437030100000000000000000000000000000000000000000000000000000000e6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			full, err := hex.DecodeString(tt.hex)
			if err != nil {
				t.Fatalf("invalid hex fixture: %v", err)
			}
			if len(full) < ReportSize {
				t.Fatalf("fixture shorter than ReportSize: got %d bytes", len(full))
			}
			want := full[len(full)-ReportSize:]

			got := BuildTimeSyncReport(tt.time)

			if hex.EncodeToString(got[:]) != hex.EncodeToString(want) {
				t.Errorf("BuildTimeSyncReport(%v) =\n  %x\nwant\n  %x", tt.time, got, want)
			}
		})
	}
}

func TestBuildTimeSyncReport_SundayEncodedAsZero(t *testing.T) {
	sunday := time.Date(2026, time.August, 16, 0, 0, 0, 0, time.UTC)
	if sunday.Weekday() != time.Sunday {
		t.Fatalf("test fixture bug: %v is not a Sunday", sunday)
	}

	report := BuildTimeSyncReport(sunday)

	if got := report[0x0A]; got != 0 {
		t.Errorf("day-of-week byte for Sunday = %d, want 0", got)
	}
}

func TestBuildTimeSyncReport_ChecksumHolds(t *testing.T) {
	// Zero-time edge case, beyond the 3 recorded samples, to catch off-by-one
	// errors in the checksum loop bounds.
	report := BuildTimeSyncReport(time.Time{})

	var sum byte
	for _, b := range report {
		sum += b
	}
	if sum != checksumTarget {
		t.Errorf("sum(report) mod 256 = %d, want %d", sum, checksumTarget)
	}
}
