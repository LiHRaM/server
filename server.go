// Copyright 2017 Vector Creations Ltd
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

package server

import (
	"crypto/ed25519"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/lihram/server/v2/storage"

	gostream "github.com/libp2p/go-libp2p-gostream"
	p2phttp "github.com/libp2p/go-libp2p-http"
	p2pdisc "github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/matrix-org/dendrite/appservice"
	"github.com/matrix-org/dendrite/clientapi"
	"github.com/matrix-org/dendrite/clientapi/producers"
	"github.com/matrix-org/dendrite/common"
	"github.com/matrix-org/dendrite/common/config"
	"github.com/matrix-org/dendrite/common/keydb"
	"github.com/matrix-org/dendrite/common/transactions"
	"github.com/matrix-org/dendrite/eduserver"
	"github.com/matrix-org/dendrite/federationapi"
	"github.com/matrix-org/dendrite/federationsender"
	"github.com/matrix-org/dendrite/mediaapi"
	"github.com/matrix-org/dendrite/publicroomsapi"
	"github.com/matrix-org/dendrite/roomserver"
	"github.com/matrix-org/dendrite/syncapi"
	"github.com/matrix-org/gomatrixserverlib"

	"github.com/matrix-org/dendrite/eduserver/cache"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func createKeyDB(
	p2p *p2pDendrite,
) keydb.Database {
	db, err := keydb.NewDatabase(
		string(p2p.Base.Cfg.Database.ServerKey),
		p2p.Base.Cfg.DbProperties(),
		p2p.Base.Cfg.Matrix.ServerName,
		p2p.Base.Cfg.Matrix.PrivateKey.Public().(ed25519.PublicKey),
		p2p.Base.Cfg.Matrix.KeyID,
	)
	if err != nil {
		logrus.WithError(err).Panicf("failed to connect to keys db")
	}
	mdns := mDNSListener{
		host:  p2p.LibP2P,
		keydb: db,
	}
	serv, err := p2pdisc.NewMdnsService(
		p2p.LibP2PContext,
		p2p.LibP2P,
		time.Second*10,
		"_matrix-dendrite-p2p._tcp",
	)
	if err != nil {
		panic(err)
	}
	serv.RegisterNotifee(&mdns)
	return db
}

func createFederationClient(
	p2p *p2pDendrite,
) *gomatrixserverlib.FederationClient {
	fmt.Println("Running in libp2p federation mode")
	fmt.Println("Warning: Federation with non-libp2p homeservers will not work in this mode yet!")
	tr := &http.Transport{}
	tr.RegisterProtocol(
		"matrix",
		p2phttp.NewTransport(p2p.LibP2P, p2phttp.ProtocolOption("/matrix")),
	)
	return gomatrixserverlib.NewFederationClientWithTransport(
		p2p.Base.Cfg.Matrix.ServerName, p2p.Base.Cfg.Matrix.KeyID, p2p.Base.Cfg.Matrix.PrivateKey, tr,
	)
}

// Callback provides the the caller a way to respond to the port being set.
type Callback interface {
	SetPort(int)
}

// Init starts the Dendrite server in p2p mode
func Init(path string, instanceName string, instancePort int, callback Callback) {
	filename := fmt.Sprintf("%s/%s-private.key", path, instanceName)
	_, err := os.Stat(filename)
	var privKey ed25519.PrivateKey
	if os.IsNotExist(err) {
		_, privKey, _ = ed25519.GenerateKey(nil)
		if err = ioutil.WriteFile(filename, privKey, 0600); err != nil {
			fmt.Printf("Couldn't write private key to file '%s': %s\n", filename, err)
		}
	} else {
		privKey, err = ioutil.ReadFile(filename)
		if err != nil {
			fmt.Printf("Couldn't read private key from file '%s': %s\n", filename, err)
			_, privKey, _ = ed25519.GenerateKey(nil)
		}
	}

	cfg := config.Dendrite{}
	cfg.Matrix.ServerName = "p2p"
	cfg.Matrix.PrivateKey = privKey
	cfg.Matrix.KeyID = gomatrixserverlib.KeyID(fmt.Sprintf("ed25519:%s", instanceName))
	cfg.Kafka.UseNaffka = true
	cfg.Kafka.Topics.OutputRoomEvent = "roomserverOutput"
	cfg.Kafka.Topics.OutputClientData = "clientapiOutput"
	cfg.Kafka.Topics.OutputTypingEvent = "typingServerOutput"
	cfg.Kafka.Topics.UserUpdates = "userUpdates"
	cfg.Database.Account = config.DataSource(fmt.Sprintf("file:%s/%s-account.db", path, instanceName))
	cfg.Database.Device = config.DataSource(fmt.Sprintf("file:%s/%s-device.db", path, instanceName))
	cfg.Database.MediaAPI = config.DataSource(fmt.Sprintf("file:%s/%s-mediaapi.db", path, instanceName))
	cfg.Database.SyncAPI = config.DataSource(fmt.Sprintf("file:%s/%s-syncapi.db", path, instanceName))
	cfg.Database.RoomServer = config.DataSource(fmt.Sprintf("file:%s/%s-roomserver.db", path, instanceName))
	cfg.Database.ServerKey = config.DataSource(fmt.Sprintf("file:%s/%s-serverkey.db", path, instanceName))
	cfg.Database.FederationSender = config.DataSource(fmt.Sprintf("file:%s/%s-federationsender.db", path, instanceName))
	cfg.Database.AppService = config.DataSource(fmt.Sprintf("file:%s/%s-appservice.db", path, instanceName))
	cfg.Database.PublicRoomsAPI = config.DataSource(fmt.Sprintf("file:%s/%s-publicroomsa.db", path, instanceName))
	cfg.Database.Naffka = config.DataSource(fmt.Sprintf("file:%s/%s-naffka.db", path, instanceName))
	if err = cfg.Derive(); err != nil {
		panic(err)
	}

	p2p := newP2PDendrite(&cfg, "Monolith")
	defer p2p.Base.Close() // nolint: errcheck

	accountDB := p2p.Base.CreateAccountsDB()
	deviceDB := p2p.Base.CreateDeviceDB()
	keyDB := createKeyDB(p2p)
	federation := createFederationClient(p2p)
	keyRing := keydb.CreateKeyRing(federation.Client, keyDB, cfg.Matrix.KeyPerspectives)

	rsAPI := roomserver.SetupRoomServerComponent(
		&p2p.Base, keyRing, federation,
	)
	eduInputAPI := eduserver.SetupEDUServerComponent(
		&p2p.Base, cache.New(),
	)
	asAPI := appservice.SetupAppServiceAPIComponent(
		&p2p.Base, accountDB, deviceDB, federation, rsAPI, transactions.New(),
	)
	fsAPI := federationsender.SetupFederationSenderComponent(
		&p2p.Base, federation, rsAPI, &keyRing,
	)
	rsAPI.SetFederationSenderAPI(fsAPI)

	clientapi.SetupClientAPIComponent(
		&p2p.Base, deviceDB, accountDB,
		federation, &keyRing, rsAPI,
		eduInputAPI, asAPI, transactions.New(), fsAPI,
	)
	eduProducer := producers.NewEDUServerProducer(eduInputAPI)
	federationapi.SetupFederationAPIComponent(&p2p.Base, accountDB, deviceDB, federation, &keyRing, rsAPI, asAPI, fsAPI, eduProducer)
	mediaapi.SetupMediaAPIComponent(&p2p.Base, deviceDB)
	publicRoomsDB, err := storage.NewPublicRoomsServerDatabaseWithPubSub(string(p2p.Base.Cfg.Database.PublicRoomsAPI), p2p.LibP2PPubsub)
	if err != nil {
		logrus.WithError(err).Panicf("failed to connect to public rooms db")
	}
	publicroomsapi.SetupPublicRoomsAPIComponent(&p2p.Base, deviceDB, publicRoomsDB, rsAPI, federation, nil) // Check this later
	syncapi.SetupSyncAPIComponent(&p2p.Base, deviceDB, accountDB, rsAPI, federation, &cfg)

	httpHandler := common.WrapHandlerInCORS(p2p.Base.APIMux)

	// Set up the API endpoints we handle. /metrics is for prometheus, and is
	// not wrapped by CORS, while everything else is
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", httpHandler)

	// Expose the matrix APIs directly rather than putting them under a /api path.
	go func() {
		httpBindAddr := fmt.Sprintf(":%d", instancePort)
		listener, err := net.Listen("tcp", httpBindAddr)
		if err != nil {
			logrus.Fatal(err)
		}
		instancePort = listener.Addr().(*net.TCPAddr).Port
		callback.SetPort(instancePort)
		logrus.Fatal(http.Serve(listener, nil))
	}()
	// Expose the matrix APIs also via libp2p
	if p2p.LibP2P != nil {
		go func() {
			logrus.Info("Listening on libp2p host ID ", p2p.LibP2P.ID())
			listener, err := gostream.Listen(p2p.LibP2P, "/matrix")
			if err != nil {
				panic(err)
			}
			defer func() {
				logrus.Fatal(listener.Close())
			}()
			logrus.Fatal(http.Serve(listener, nil))
		}()
	}

	// We want to block forever to let the HTTP and HTTPS handler serve the APIs
	select {}
}
