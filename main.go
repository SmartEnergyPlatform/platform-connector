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

package main

import (
	"github.com/SmartEnergyPlatform/platform-connector/lib"
	"github.com/SmartEnergyPlatform/platform-connector/util"
	"flag"
	"log"
	"os"
	"os/signal"

	"syscall"

	"github.com/Shopify/sarama"
)

func main() {

	configLocation := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	err := util.LoadConfig(*configLocation)
	if err != nil {
		log.Fatal(err)
	}

	if util.Config.SaramaLog == "true" {
		sarama.Logger = log.New(os.Stderr, "[Sarama] ", log.LstdFlags)
	}

	lib.InitConnectionLog()

	defer lib.Sessions().Close()

	err = lib.InitConsumer()
	if err != nil {
		log.Fatal("ERROR: unable to start consumer", err)
	}
	defer lib.CloseConsumer()
	defer lib.ClearBindings()

	err = lib.InitProducer()
	if err != nil {
		log.Fatal("ERROR: unable to start producer", err)
	}
	defer lib.CloseProducer()

	go lib.WsStart()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	sig := <-shutdown
	log.Println("received shutdown signal", sig)
}
