package diskutil

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-cmd/cmd"
	"howett.net/plist"
)

type DiskUtil struct {
}

func New() (*DiskUtil, error) {
	d := &DiskUtil{}
	err := d.check()
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DiskUtil) Resize(device string, size int64) (err error) {
	if size == 0 {
		size, err = d.GetResizeLimit(device)
		if err != nil {
			return
		}
	}
	_, err = d.runSync("resizeVolume", device, strconv.FormatInt(size, 10))
	return
}

func (d *DiskUtil) GetResizeLimit(device string) (int64, error) {
	out, err := d.runSync("resizeVolume", device, "limits", "-plist")
	if err != nil {
		return 0, err
	}
	var r limitsResult
	_, err = plist.Unmarshal([]byte(out), &r)
	if err != nil {
		return 0, err
	}
	return r.MinimumSizeNoGuard, nil
}

func (d *DiskUtil) check() error {
	_, err := d.runSync("list")
	return err
}

func (d *DiskUtil) runSync(args ...string) (string, error) {
	c := cmd.NewCmd("diskutil", args...)
	s := <-c.Start()
	if s.Error == nil && len(s.Stderr) != 0 {
		return "", errors.New(strings.Join(s.Stderr, "\n"))
	}
	return strings.Join(s.Stdout, "\n"), s.Error
}

type limitsResult struct {
	MinimumSizeNoGuard int64 `plist:"MinimumSizeNoGuard"`
}
