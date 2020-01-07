/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package composite

import (
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/utils"
)

type finderComposite struct {
	finders []discovery.ProposalFinder
}

// NewFinder creates an instance of composite finder
func NewFinder(finders ...discovery.ProposalFinder) *finderComposite {
	return &finderComposite{finders: finders}
}

// AddRegistry adds registry to set of registries
func (fc *finderComposite) AddFinder(finder discovery.ProposalFinder) {
	fc.finders = append(fc.finders, finder)
}

// Start begins proposals synchronization to storage
func (fc *finderComposite) Start() error {
	var err utils.ErrorCollection
	for _, finder := range fc.finders {
		err.Add(finder.Start())
	}
	return err.Error()
}

// Stop ends proposals synchronization to storage
func (fc *finderComposite) Stop() {
	for _, finder := range fc.finders {
		finder.Stop()
	}
}
