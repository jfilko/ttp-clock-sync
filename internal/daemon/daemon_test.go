package daemon

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"trs-clock-sync/internal/dock"
	"trs-clock-sync/internal/hidtransport"
)

// fakeHandle is a hidtransport.Handle test double that records writes and
// can be made to fail on demand.
type fakeHandle struct {
	info     hidtransport.Info
	writeErr error
	onWrite  func(info hidtransport.Info, report []byte)
	onClose  func(info hidtransport.Info)
	closeErr error
}

func (h *fakeHandle) Write(report []byte) (int, error) {
	if h.writeErr != nil {
		return 0, h.writeErr
	}
	if h.onWrite != nil {
		cp := append([]byte(nil), report...)
		h.onWrite(h.info, cp)
	}
	return len(report), nil
}

func (h *fakeHandle) Close() error {
	if h.onClose != nil {
		h.onClose(h.info)
	}
	return h.closeErr
}

// fakeBackend is a hidtransport.Backend test double. connected holds the set
// of currently "plugged in" devices; openErr/writeErr let individual devices
// (keyed by Path) be made to fail Open/Write.
type fakeBackend struct {
	connected []hidtransport.Info
	openErr   map[string]error
	writeErr  map[string]error

	openCalls  []hidtransport.Info
	writes     []hidtransport.Info
	writtenFor map[string][]byte
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		openErr:    map[string]error{},
		writeErr:   map[string]error{},
		writtenFor: map[string][]byte{},
	}
}

func (b *fakeBackend) Enumerate(vendorID, productID uint16) []hidtransport.Info {
	var out []hidtransport.Info
	for _, info := range b.connected {
		if info.VendorID == vendorID && info.ProductID == productID {
			out = append(out, info)
		}
	}
	return out
}

func (b *fakeBackend) Open(info hidtransport.Info) (hidtransport.Handle, error) {
	b.openCalls = append(b.openCalls, info)
	if err, ok := b.openErr[info.Path]; ok {
		return nil, err
	}
	return &fakeHandle{
		info:     info,
		writeErr: b.writeErr[info.Path],
		onWrite: func(info hidtransport.Info, report []byte) {
			b.writes = append(b.writes, info)
			b.writtenFor[info.Path] = report
		},
	}, nil
}

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, nil))
}

func countLines(logs, substr string) int {
	count := 0
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, substr) {
			count++
		}
	}
	return count
}

var testDevice = dock.Device{VendorID: 0x1234, ProductID: 0x5678, UsagePage: 0xFF08, Usage: 0x0002}

// testInfo builds a hidtransport.Info for testDevice's matching HID
// collection, distinguished only by Path (as real connected devices are).
func testInfo(path string) hidtransport.Info {
	return hidtransport.Info{
		VendorID:  testDevice.VendorID,
		ProductID: testDevice.ProductID,
		UsagePage: testDevice.UsagePage,
		Usage:     testDevice.Usage,
		Path:      path,
	}
}

func newTestDaemon(backend *fakeBackend, logger *slog.Logger, now time.Time) *Daemon {
	clock := now
	return &Daemon{
		Backend: backend,
		Devices: []dock.Device{testDevice},
		Logger:  logger,
		Now:     func() time.Time { return clock },
	}
}

func TestTick_DeviceAlreadyConnectedAtStartup_PushesImmediately(t *testing.T) {
	backend := newFakeBackend()
	backend.connected = []hidtransport.Info{testInfo("dev1")}
	var buf bytes.Buffer
	d := newTestDaemon(backend, testLogger(&buf), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	d.tick()

	if len(backend.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(backend.writes))
	}
	logs := buf.String()
	if countLines(logs, "device detected") != 1 {
		t.Errorf("expected exactly one \"device detected\" log, got logs:\n%s", logs)
	}
	if countLines(logs, "pushing data") != 1 {
		t.Errorf("expected exactly one \"pushing data\" log, got logs:\n%s", logs)
	}
}

func TestTick_DeviceNeverConnected_NoOpenOrWrite(t *testing.T) {
	backend := newFakeBackend() // nothing connected
	var buf bytes.Buffer
	d := newTestDaemon(backend, testLogger(&buf), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	for i := 0; i < 10; i++ {
		d.tick()
	}

	if len(backend.openCalls) != 0 {
		t.Errorf("openCalls = %d, want 0", len(backend.openCalls))
	}
	if len(backend.writes) != 0 {
		t.Errorf("writes = %d, want 0", len(backend.writes))
	}
	if buf.Len() != 0 {
		t.Errorf("expected no log output for a device that never connects, got:\n%s", buf.String())
	}
}

func TestTick_SecondTickBeforePushInterval_NoSecondPush(t *testing.T) {
	backend := newFakeBackend()
	backend.connected = []hidtransport.Info{testInfo("dev1")}
	var buf bytes.Buffer
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := newTestDaemon(backend, testLogger(&buf), start)

	d.tick() // startup push
	if len(backend.writes) != 1 {
		t.Fatalf("after first tick, writes = %d, want 1", len(backend.writes))
	}

	d.Now = func() time.Time { return start.Add(30 * time.Second) } // < PushInterval
	d.tick()

	if len(backend.writes) != 1 {
		t.Errorf("after second tick before PushInterval, writes = %d, want 1", len(backend.writes))
	}
}

func TestTick_PushIntervalElapsed_SecondPushOccurs(t *testing.T) {
	backend := newFakeBackend()
	backend.connected = []hidtransport.Info{testInfo("dev1")}
	var buf bytes.Buffer
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := newTestDaemon(backend, testLogger(&buf), start)

	d.tick() // startup push
	d.Now = func() time.Time { return start.Add(PushInterval) }
	d.tick()

	if len(backend.writes) != 2 {
		t.Errorf("writes = %d, want 2", len(backend.writes))
	}
}

func TestTick_MidRunConnect_ImmediatePushRegardlessOfPushInterval(t *testing.T) {
	backend := newFakeBackend() // starts disconnected
	var buf bytes.Buffer
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := newTestDaemon(backend, testLogger(&buf), start)

	d.tick() // not connected, no push
	if len(backend.writes) != 0 {
		t.Fatalf("writes after first tick = %d, want 0", len(backend.writes))
	}

	// Device plugs in shortly after startup, well within PushInterval.
	backend.connected = []hidtransport.Info{testInfo("dev1")}
	d.Now = func() time.Time { return start.Add(1 * time.Second) }
	d.tick()

	if len(backend.writes) != 1 {
		t.Errorf("writes after connect = %d, want 1", len(backend.writes))
	}
	logs := buf.String()
	if countLines(logs, "device detected") != 1 {
		t.Errorf("expected exactly one \"device detected\" log, got logs:\n%s", logs)
	}
}

func TestTick_DisconnectThenSteadyState_NoPushesNoLogSpam(t *testing.T) {
	backend := newFakeBackend()
	backend.connected = []hidtransport.Info{testInfo("dev1")}
	var buf bytes.Buffer
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := newTestDaemon(backend, testLogger(&buf), start)

	d.tick() // connect + push

	backend.connected = nil
	d.tick() // disconnect
	buf.Reset()

	for i := 0; i < 5; i++ {
		d.tick()
	}

	if len(backend.writes) != 1 {
		t.Errorf("writes after disconnect = %d, want 1 (only the initial push)", len(backend.writes))
	}
	if buf.Len() != 0 {
		t.Errorf("expected no log output on repeated not-connected ticks, got:\n%s", buf.String())
	}
}

func TestTick_DisconnectLogsExactlyOnce(t *testing.T) {
	backend := newFakeBackend()
	backend.connected = []hidtransport.Info{testInfo("dev1")}
	var buf bytes.Buffer
	d := newTestDaemon(backend, testLogger(&buf), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	d.tick() // connect + push
	backend.connected = nil
	d.tick() // disconnect

	logs := buf.String()
	if countLines(logs, "device disconnected") != 1 {
		t.Errorf("expected exactly one \"device disconnected\" log, got logs:\n%s", logs)
	}
}

func TestPush_OneDeviceFailsAnotherSucceeds(t *testing.T) {
	backend := newFakeBackend()
	failing := testInfo("bad")
	working := testInfo("good")
	backend.connected = []hidtransport.Info{failing, working}
	backend.openErr["bad"] = errors.New("boom")
	var buf bytes.Buffer
	d := newTestDaemon(backend, testLogger(&buf), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	d.tick()

	if len(backend.writes) != 1 || backend.writes[0].Path != "good" {
		t.Fatalf("writes = %+v, want exactly one write to \"good\"", backend.writes)
	}
	logs := buf.String()
	if countLines(logs, "pushing data") != 1 {
		t.Errorf("expected exactly one \"pushing data\" log, got logs:\n%s", logs)
	}
}

func TestPush_WriteFailureDoesNotStopOtherDevices(t *testing.T) {
	backend := newFakeBackend()
	failing := testInfo("bad")
	working := testInfo("good")
	backend.connected = []hidtransport.Info{failing, working}
	backend.writeErr["bad"] = errors.New("boom")
	var buf bytes.Buffer
	d := newTestDaemon(backend, testLogger(&buf), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	d.tick()

	if len(backend.writes) != 1 || backend.writes[0].Path != "good" {
		t.Fatalf("writes = %+v, want exactly one write to \"good\"", backend.writes)
	}
}

func TestEnumerate_IgnoresOtherUsageCollectionsOnSameVIDPID(t *testing.T) {
	backend := newFakeBackend()
	match := testInfo("command-channel")
	other := match
	other.Path = "keyboard-collection"
	other.UsagePage = 0x0001
	other.Usage = 0x0006
	backend.connected = []hidtransport.Info{other, match}
	var buf bytes.Buffer
	d := newTestDaemon(backend, testLogger(&buf), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	d.tick()

	if len(backend.writes) != 1 || backend.writes[0].Path != "command-channel" {
		t.Fatalf("writes = %+v, want exactly one write to \"command-channel\"", backend.writes)
	}
	if len(backend.openCalls) != 1 {
		t.Errorf("openCalls = %+v, want exactly one Open call, for the matching UsagePage/Usage only", backend.openCalls)
	}
}
