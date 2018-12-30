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
	"errors"
	"testing"
	"time"

	"github.com/asdine/storm"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/boltdbtest"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrations"
	"github.com/stretchr/testify/assert"
)

var (
	mockMigration = migrations.Migration{
		Name: "test",
		Migrate: func(*storm.DB) error {
			return nil
		},
		Date: time.Now().UTC(),
	}
)

type mockMigrationApplier struct {
	calledAt time.Time
	mockErr  error
}

func (mma *mockMigrationApplier) Migrate(*storm.DB) error {
	mma.calledAt = time.Now().UTC()
	return mma.mockErr
}

func createDBAndMigrator(t *testing.T, dir string) (*Bolt, *Migrator) {
	bolt, err := NewStorage(dir)
	assert.Nil(t, err)

	return bolt, NewMigrator(bolt)
}

func TestSavesMigrationRun(t *testing.T) {
	dir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, dir)

	bolt, migrator := createDBAndMigrator(t, dir)

	err := migrator.saveMigrationRun(mockMigration)
	assert.Nil(t, err)

	migrations := []migrations.Migration{}
	err = bolt.GetAllFrom(migrationIndexBucketName, &migrations)
	assert.Nil(t, err)
	assert.Len(t, migrations, 1)
	assert.Equal(t, migrations[0].Date, mockMigration.Date)
	assert.Equal(t, migrations[0].Name, mockMigration.Name)
}

func TestDetectsAppliedMigration(t *testing.T) {
	dir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, dir)

	_, migrator := createDBAndMigrator(t, dir)

	err := migrator.saveMigrationRun(mockMigration)
	assert.Nil(t, err)

	res, err := migrator.isApplied(mockMigration)
	assert.Nil(t, err)
	assert.True(t, res)
}

func TestDetectsNotAppliedMigration(t *testing.T) {
	dir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, dir)

	_, migrator := createDBAndMigrator(t, dir)

	res, err := migrator.isApplied(mockMigration)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestAppliesMigration(t *testing.T) {
	dir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, dir)

	bolt, migrator := createDBAndMigrator(t, dir)

	mockApplier := &mockMigrationApplier{}
	migrationCopy := mockMigration
	migrationCopy.Migrate = mockApplier.Migrate

	err := migrator.migrate(migrationCopy)
	assert.Nil(t, err)
	assert.False(t, mockApplier.calledAt.IsZero())

	migrations := []migrations.Migration{}
	err = bolt.GetAllFrom(migrationIndexBucketName, &migrations)
	assert.Nil(t, err)
	assert.Len(t, migrations, 1)
	assert.Equal(t, migrations[0].Date, mockMigration.Date)
	assert.Equal(t, migrations[0].Name, mockMigration.Name)
}

func TestReturnsMigrationError(t *testing.T) {
	dir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, dir)

	_, migrator := createDBAndMigrator(t, dir)

	mockApplier := &mockMigrationApplier{
		mockErr: errors.New("test"),
	}
	migrationCopy := mockMigration
	migrationCopy.Migrate = mockApplier.Migrate

	err := migrator.migrate(migrationCopy)
	assert.Equal(t, mockApplier.mockErr, err)
}

func TestAppliesMigrationOnce(t *testing.T) {
	dir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, dir)

	_, migrator := createDBAndMigrator(t, dir)

	mockApplier := &mockMigrationApplier{}
	migrationCopy := mockMigration
	migrationCopy.Migrate = mockApplier.Migrate

	err := migrator.migrate(migrationCopy)
	assert.Nil(t, err)
	assert.False(t, mockApplier.calledAt.IsZero())

	mockApplier.calledAt = time.Time{}
	err = migrator.migrate(migrationCopy)
	assert.Nil(t, err)
	assert.True(t, mockApplier.calledAt.IsZero())
}

func TestMigrationSorter(t *testing.T) {
	dir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, dir)

	firstMigration := mockMigration
	firstMigration.Date = time.Date(2018, 12, 04, 12, 00, 00, 0, time.UTC)

	secondMigration := mockMigration
	secondMigration.Date = time.Date(2018, 12, 05, 12, 00, 00, 0, time.UTC)

	migrations := []migrations.Migration{
		secondMigration, firstMigration,
	}

	_, migrator := createDBAndMigrator(t, dir)
	sorted := migrator.sortMigrations(migrations)

	assert.Equal(t, sorted[0].Date, firstMigration.Date)
	assert.Equal(t, sorted[1].Date, secondMigration.Date)
}

func TestRunsMigrationsInOrder(t *testing.T) {
	dir := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, dir)

	firstMockApplier := &mockMigrationApplier{}
	firstMigration := migrations.Migration{
		Name:    "first",
		Date:    time.Date(2018, 12, 04, 12, 00, 00, 0, time.UTC),
		Migrate: firstMockApplier.Migrate,
	}

	secondMockApplier := &mockMigrationApplier{}
	secondMigration := migrations.Migration{
		Name:    "second",
		Date:    time.Date(2018, 12, 05, 12, 00, 00, 0, time.UTC),
		Migrate: secondMockApplier.Migrate,
	}

	migrations := []migrations.Migration{
		secondMigration, firstMigration,
	}

	_, migrator := createDBAndMigrator(t, dir)
	err := migrator.RunMigrations(migrations)
	assert.Nil(t, err)

	assert.True(t, firstMockApplier.calledAt.Before(secondMockApplier.calledAt))
}
