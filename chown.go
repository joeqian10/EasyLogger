// +build !linux

package EasyLogger

import (
	"os"
)

func chown(_ string, _ os.FileInfo) error {
	return nil
}
