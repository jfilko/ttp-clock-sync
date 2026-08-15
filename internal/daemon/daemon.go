package daemon

import (
	"log/slog"
	"time"

	"trs-clock-sync/internal/dock"
	"trs-clock-sync/internal/hidtransport"
)

const (
	ConnectivityCheckInterval = 5 * time.Second
	PushInterval              = 1 * time.Minute
)

type Daemon struct {
	Backend hidtransport.Backend
	Devices []dock.Device
	Logger  *slog.Logger
	Now     func() time.Time // overridable in tests; defaults to time.Now

	connected bool
	lastPush  time.Time
}

// New constructs a Daemon wired to the real HID backend, the supported
// device list, and time.Now.
func New(logger *slog.Logger) *Daemon {
	return &Daemon{
		Backend: hidtransport.Real{},
		Devices: dock.SupportedDevices,
		Logger:  logger,
		Now:     time.Now,
	}
}

// Run pushes an initial time-sync (if a device is already connected), then
// loops forever on ConnectivityCheckInterval. Run never returns under normal
// operation.
func (d *Daemon) Run() {
	d.tick()
	ticker := time.NewTicker(ConnectivityCheckInterval)
	for range ticker.C {
		d.tick()
	}
}

// tick performs one connectivity check, logs state transitions, and pushes a
// time-sync report when due. It is the pure, directly-testable core of the
// loop.
func (d *Daemon) tick() {
	now := d.Now()
	infos := d.enumerate()
	nowConnected := len(infos) > 0

	switch {
	case nowConnected && !d.connected:
		d.Logger.Info("device detected", "count", len(infos))
	case !nowConnected && d.connected:
		d.Logger.Info("device disconnected")
	}
	justConnected := nowConnected && !d.connected
	d.connected = nowConnected

	if !nowConnected {
		return
	}
	due := justConnected || d.lastPush.IsZero() || now.Sub(d.lastPush) >= PushInterval
	if !due {
		return
	}
	d.push(infos, now)
	d.lastPush = now
}

func (d *Daemon) enumerate() []hidtransport.Info {
	var infos []hidtransport.Info
	for _, dv := range d.Devices {
		for _, info := range d.Backend.Enumerate(dv.VendorID, dv.ProductID) {
			// A single physical device enumerates as multiple Infos (one per
			// HID usage collection); only the one matching dv's UsagePage/
			// Usage is the dock's actual command channel.
			if info.UsagePage == dv.UsagePage && info.Usage == dv.Usage {
				infos = append(infos, info)
			}
		}
	}
	return infos
}

func (d *Daemon) push(infos []hidtransport.Info, now time.Time) {
	report := dock.BuildTimeSyncReport(now)
	for _, info := range infos {
		handle, err := d.Backend.Open(info)
		if err != nil {
			d.Logger.Error("open device failed", "vendorID", info.VendorID, "productID", info.ProductID, "err", err)
			continue
		}
		if _, err := handle.Write(report[:]); err != nil {
			d.Logger.Error("write report failed", "vendorID", info.VendorID, "productID", info.ProductID, "err", err)
		} else {
			d.Logger.Info("pushing data", "vendorID", info.VendorID, "productID", info.ProductID)
		}
		_ = handle.Close()
	}
}
