/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package boltdb

import (
	"sort"

	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrations"
)

const migrationIndexBucketName = "migrations"

// Migrator represents the component responsible for running migrations on bolt db
type Migrator struct {
	db *Bolt
}

// NewMigrator returns a new instance of migrator
func NewMigrator(db *Bolt) *Migrator {
	return &Migrator{
		db: db,
	}
}

func (m *Migrator) isApplied(migration migrations.Migration) (bool, error) {
	migrations := []migrations.Migration{}
	err := m.db.db.From(migrationIndexBucketName).All(&migrations)
	if err != nil {
		return true, err
	}

	for i := range migrations {
		if migration.Name == migrations[i].Name {
			return true, nil
		}
	}
	return false, nil
}

func (m *Migrator) saveMigrationRun(migration migrations.Migration) error {
	return m.db.db.From(migrationIndexBucketName).Save(&migration)
}

func (m *Migrator) migrate(migration migrations.Migration) error {
	isRun, err := m.isApplied(migration)
	if err != nil || isRun {
		return err
	}
	log.Info("running migration ", migration.Name)
	err = migration.Migrate(m.db.db)
	if err != nil {
		return err
	}
	log.Info("saving migration ", migration.Name)
	return m.saveMigrationRun(migration)
}

func (m *Migrator) sortMigrations(sequence []migrations.Migration) []migrations.Migration {
	sort.Slice(sequence, func(i, j int) bool {
		return sequence[i].Date.Before(sequence[j].Date)
	})
	return sequence
}

// RunMigrations runs the given sequence of migrations
func (m *Migrator) RunMigrations(sequence []migrations.Migration) error {
	sorted := m.sortMigrations(sequence)
	for i := range sorted {
		err := m.migrate(sorted[i])
		if err != nil {
			return err
		}
	}
	return nil
}
