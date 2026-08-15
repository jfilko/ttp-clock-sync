package hidtransport

// Info identifies one connected HID device. Composite devices (e.g. ones that
// also expose keyboard/mouse HID collections) enumerate as multiple Infos
// sharing the same VendorID/ProductID but differing UsagePage/Usage.
type Info struct {
	VendorID  uint16
	ProductID uint16
	UsagePage uint16
	Usage     uint16
	Path      string // opaque, backend-specific identifier passed back to Open
}

// Handle is an open HID device that reports can be written to.
type Handle interface {
	Write(report []byte) (int, error)
	Close() error
}

// Backend enumerates and opens HID devices. Real (real.go) wraps
// github.com/bearsh/hid for production use; tests use a fake.
type Backend interface {
	Enumerate(vendorID, productID uint16) []Info
	Open(info Info) (Handle, error)
}
