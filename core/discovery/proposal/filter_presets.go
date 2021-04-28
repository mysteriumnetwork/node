/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package proposal

import (
	"sync"

	"github.com/pkg/errors"
)

var errMsgBoltNotFound = "not found"

type persistentStorage interface {
	Store(bucket string, data interface{}) error
	GetAllFrom(bucket string, data interface{}) error
	GetLast(bucket string, to interface{}) error
	Delete(bucket string, data interface{}) error
}

const (
	bucketName = "proposal-filter-presets"
	startingID = 100
)

// FilterPresetStorage filter preset storage
type FilterPresetStorage struct {
	lock    sync.Mutex
	storage persistentStorage
}

// NewFilterPresetStorage constructor for FilterPresetStorage
func NewFilterPresetStorage(storage persistentStorage) *FilterPresetStorage {
	return &FilterPresetStorage{
		storage: storage,
	}
}

// List list all filter presets
// system preset are identified by preset.ID < startingID
func (fps *FilterPresetStorage) List() (*FilterPresets, error) {
	fps.lock.Lock()
	defer fps.lock.Unlock()

	entries, err := fps.ls()
	return filterPresets(entries).prependDefault(), err
}

func (fps *FilterPresetStorage) ls() ([]FilterPreset, error) {
	var entries []FilterPreset
	err := fps.storage.GetAllFrom(bucketName, &entries)
	return entries, err
}

// Save created or updates existing
// to update existing: preset.ID > startingID
func (fps *FilterPresetStorage) Save(preset FilterPreset) error {
	fps.lock.Lock()
	defer fps.lock.Unlock()

	if preset.ID != 0 {
		return fps.storage.Store(bucketName, &preset)
	}

	nextID, err := fps.nextID()
	if err != nil {
		return err
	}

	preset.ID = nextID
	err = fps.storage.Store(bucketName, &preset)
	if err != nil {
		return err
	}

	return nil
}

// Delete delete filter preset by id
func (fps *FilterPresetStorage) Delete(id int) error {
	fps.lock.Lock()
	defer fps.lock.Unlock()

	if id < 100 {
		return errors.New("deleting system presets is not allowed")
	}

	toRemove := FilterPreset{ID: id}
	return fps.storage.Delete(bucketName, &toRemove)
}

func (fps *FilterPresetStorage) nextID() (int, error) {
	var last FilterPreset
	err := fps.storage.GetLast(bucketName, &last)
	if err != nil {
		if err.Error() == errMsgBoltNotFound {
			return startingID, nil
		}
		return 0, err
	}
	return last.ID + 1, err
}

var defaultPresets = []FilterPreset{
	{
		ID:                1,
		Name:              "Media Streaming",
		NodeType:          Residential,
		QualityLowerBound: 1.4,
	},
	{
		ID:                2,
		Name:              "Browsing",
		QualityLowerBound: 1.5,
	},
	{
		ID:       3,
		Name:     "Download",
		NodeType: Hosting,
	},
}

// NodeType represents type of node
type NodeType string

const (
	// Residential node type value
	Residential NodeType = "residential"
	// Hosting node type value
	Hosting = "hosting"
	// Business node type value
	Business = "business"
	// Cellular node type value
	Cellular = "cellular"
	// Dialup node type value
	Dialup = "dialup"
	// College node type value
	College = "college"
)

// FilterPreset represent predefined or user stored proposal filter preset
type FilterPreset struct {
	ID                     int
	Name                   string
	NodeType               NodeType
	UpperTimeMinPriceBound string
	UpperGBPriceBound      string
	QualityLowerBound      float64
}

func filterPresets(entries []FilterPreset) *FilterPresets {
	return &FilterPresets{Entries: entries}
}

// FilterPresets convenience wrapper
type FilterPresets struct {
	Entries []FilterPreset
}

func (ls *FilterPresets) prependDefault() *FilterPresets {
	var result = make([]FilterPreset, len(defaultPresets))
	copy(result, defaultPresets)
	ls.Entries = append(result, ls.Entries...)
	return ls
}

func (ls *FilterPresets) byId(id int) (FilterPreset, bool) {
	for _, e := range ls.Entries {
		if e.ID == id {
			return e, true
		}
	}

	return FilterPreset{}, false
}
