/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package device

import (
	"strings"

	"github.com/mysteriumnetwork/node/ci/util/shell"
	"github.com/pkg/errors"
)

// AttachLoop attaches system image to the first available loop device and returns its name
func AttachLoop(imageFilename string) (loopDevice string, err error) {
	output, err := shell.NewCmdf("sudo /sbin/losetup --show --find --partscan %s", imageFilename).Output()
	if err != nil {
		return "", errors.Wrapf(err, "could not attach image to loop loopDevice, image: %s", imageFilename)
	}
	devices := strings.Split(output, "\n")
	if len(devices) > 0 {
		loopDevice = devices[0]
	}
	if loopDevice == "" {
		return "", errors.New("loop loopDevice not found")
	}
	return loopDevice, nil
}

// DetachLoop detaches given loop device
func DetachLoop(loopDevice string) error {
	err := shell.NewCmdf("sudo /sbin/losetup -d %s", loopDevice).Run()
	return errors.Wrapf(err, "could not detach image from loop loopDevice %s", loopDevice)
}
