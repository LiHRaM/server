// Copyright 2017 Vector Creations Ltd
// Copyright 2018 New Vector Ltd
// Copyright 2019-2020 The Matrix.org Foundation C.I.C.
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

package main

import (
	"flag"
	"os"

	"github.com/lihram/server/v2"
	"github.com/sirupsen/logrus"
)

func main() {
	instanceName := flag.String("name", "dendrite-p2p", "the name of this P2P demo instance")
	instancePort := flag.Int("port", 0, "the port that the client API will listen on")
	instancePath := flag.String("path", "./build", "the path where databases will be stored")
	flag.Parse()

	// Create the build directory if it does not exist
	if _, err := os.Stat(*instancePath); os.IsNotExist(err) {
		err := os.MkdirAll(*instancePath, 0755)
		if err != nil {
			panic(nil)
		}
	}

	server.Init(
		*instancePath,
		*instanceName,
		*instancePort,
		simpleCallback{},
	)
}

type simpleCallback struct{}

func (cb simpleCallback) SetPort(port int) {
	logrus.Info("Listening on :", port)
}
