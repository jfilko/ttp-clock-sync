package hidtransport

import "github.com/bearsh/hid"

// Real is the Backend implementation backed by github.com/bearsh/hid.
type Real struct{}

func (Real) Enumerate(vendorID, productID uint16) []Info {
	devices := hid.Enumerate(vendorID, productID)
	infos := make([]Info, len(devices))
	for i, d := range devices {
		infos[i] = Info{VendorID: d.VendorID, ProductID: d.ProductID, UsagePage: d.UsagePage, Usage: d.Usage, Path: d.Path}
	}
	return infos
}

func (Real) Open(info Info) (Handle, error) {
	return hid.DeviceInfo{Path: info.Path, VendorID: info.VendorID, ProductID: info.ProductID}.Open()
}
