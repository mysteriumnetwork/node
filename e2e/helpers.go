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

package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Mongo represents a mongo driver
type Mongo struct {
	db *mongo.Database
}

// NewMongo returns a new mongo driver
func NewMongo() (*Mongo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	uri := fmt.Sprintf("%v://%v:%v@%v/%v", "mongodb", "transactor", "transactor", "mongodb:27017", "transactor")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("could not connect to mongodb: %w", err)
	}

	return &Mongo{
		db: mongoClient.Database("transactor"),
	}, nil
}

type registrationBounty struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Identity string             `bson:"identity" json:"identity"`
}

// InsertRegistrationBounty inserts a new entry into the bounty collection
func (m *Mongo) InsertRegistrationBounty(identity common.Address) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	toInsert := registrationBounty{
		ID:       primitive.NewObjectID(),
		Identity: identity.Hex(),
	}
	_, err := m.db.Collection("registration_bounties").InsertOne(ctx, toInsert)
	return err
}
