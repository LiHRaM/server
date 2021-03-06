// Copyright 2020 The Matrix.org Foundation C.I.C.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"net/url"

	"github.com/lihram/server/v2/storage/postgreswithdht"
	"github.com/lihram/server/v2/storage/postgreswithpubsub"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/matrix-org/dendrite/publicroomsapi/storage"
	"github.com/matrix-org/dendrite/publicroomsapi/storage/sqlite3"
)

const schemePostgres = "postgres"
const schemeFile = "file"

// NewPublicRoomsServerDatabase opens a database connection.
func NewPublicRoomsServerDatabaseWithDHT(dataSourceName string, dht *dht.IpfsDHT) (storage.Database, error) {
	uri, err := url.Parse(dataSourceName)
	if err != nil {
		return postgreswithdht.NewPublicRoomsServerDatabase(dataSourceName, dht)
	}
	switch uri.Scheme {
	case schemePostgres:
		return postgreswithdht.NewPublicRoomsServerDatabase(dataSourceName, dht)
	case schemeFile:
		return sqlite3.NewPublicRoomsServerDatabase(dataSourceName)
	default:
		return postgreswithdht.NewPublicRoomsServerDatabase(dataSourceName, dht)
	}
}

// NewPublicRoomsServerDatabase opens a database connection.
func NewPublicRoomsServerDatabaseWithPubSub(dataSourceName string, pubsub *pubsub.PubSub) (storage.Database, error) {
	uri, err := url.Parse(dataSourceName)
	if err != nil {
		return postgreswithpubsub.NewPublicRoomsServerDatabase(dataSourceName, pubsub)
	}
	switch uri.Scheme {
	case schemePostgres:
		return postgreswithpubsub.NewPublicRoomsServerDatabase(dataSourceName, pubsub)
	case schemeFile:
		return sqlite3.NewPublicRoomsServerDatabase(dataSourceName)
	default:
		return postgreswithpubsub.NewPublicRoomsServerDatabase(dataSourceName, pubsub)
	}
}
