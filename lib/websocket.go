/*
 * Copyright 2018 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"github.com/SmartEnergyPlatform/platform-connector/util"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func WsStart() {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }} // allow x origin

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Print("upgrade:", err)
			return
		}
		NewSession(c)
	})

	if util.Config.WssPort != "" && util.Config.TlsCertFile != "" && util.Config.TlsKeyFile != "" {
		log.Println("start wss on port: ", util.Config.WssPort)
		log.Fatal(http.ListenAndServeTLS(":"+util.Config.WssPort, util.Config.TlsCertFile, util.Config.TlsKeyFile, nil))
	} else {
		log.Println("start ws on port: ", util.Config.WsPort)
		log.Fatal(http.ListenAndServe(":"+util.Config.WsPort, nil))
	}
}
