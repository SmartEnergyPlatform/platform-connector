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
	"encoding/json"
	"log"
	"time"

	"github.com/SmartEnergyPlatform/platform-connector/util"

	"github.com/SmartEnergyPlatform/amqp-wrapper-lib"
)

var conn *amqp_wrapper_lib.Connection

func InitConnectionLog() {
	var err error
	conn, err = amqp_wrapper_lib.Init(util.Config.AmqpUrl, []string{util.Config.ConnectorLogTopic, util.Config.DeviceLogTopic, util.Config.GatewayLogTopic}, util.Config.AmqpReconnectTimeout)
	if err != nil {
		log.Fatal("ERROR: while initializing amqp connection", err)
		return
	}

	ticker := time.NewTicker(time.Duration(util.Config.LogTime) * time.Second)
	go func() {
		for tick := range ticker.C {
			connectionLog := getCurrentConnectionLog()
			log.Println("DEBUG: log connection state", connectionLog)
			if err = connectionLog.Send(); err != nil {
				log.Println("ERROR logging connection state", tick, err)
			}
		}
	}()
	return
}

func getCurrentConnectionLog() (connectionLog ConnectorLog) {
	connectionLog = ConnectorLog{Connected: true}
	deviceusedMap := map[string]bool{}
	gwusedMap := map[string]bool{}
	for _, session := range Sessions().GetSessions() {
		gwused := gwusedMap[session.Gateway]
		if session.Gateway == "" {
			log.Println("WARNING: session without gateway", session.Cred, session.Id, session.UriCache)
			continue
		}
		if !gwused {
			gwusedMap[session.Gateway] = true
			connectionLog.Gateways = append(connectionLog.Gateways, GatewayLog{Connected: true, Gateway: session.Gateway})
		}
		for _, deviceServiceEntity := range session.UriCache {
			used := deviceusedMap[deviceServiceEntity.Device.Id]
			if !used {
				connectionLog.Devices = append(connectionLog.Devices, DeviceLog{Connected: true, Device: deviceServiceEntity.Device.Id})
				deviceusedMap[deviceServiceEntity.Device.Id] = true
			}
		}
	}
	return
}

func sendEvent(topic string, event interface{}) error {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Println("ERROR: event marshaling:", err)
		return err
	}
	log.Println("DEBUG: send amqp event: ", topic, string(payload))
	return conn.Publish(topic, payload)
}

type ConnectorLog struct {
	Connected bool         `json:"connected"`
	Connector string       `json:"connector"`
	Gateways  []GatewayLog `json:"gateways"`
	Devices   []DeviceLog  `json:"devices"`
	Time      time.Time    `json:"time"`
}

type GatewayLog struct {
	Connected bool      `json:"connected"`
	Connector string    `json:"connector"`
	Gateway   string    `json:"gateway"`
	Time      time.Time `json:"time"`
}

type DeviceLog struct {
	Connected bool      `json:"connected"`
	Connector string    `json:"connector"`
	Device    string    `json:"device"`
	Time      time.Time `json:"time"`
}

func (this DeviceLog) Send() error {
	this.Connector = util.Config.ConsumerName
	this.Time = time.Now()
	return sendEvent(util.Config.DeviceLogTopic, this)
}

func (this GatewayLog) Send() error {
	this.Connector = util.Config.ConsumerName
	this.Time = time.Now()
	return sendEvent(util.Config.GatewayLogTopic, this)
}

func (this ConnectorLog) Send() error {
	this.Connector = util.Config.ConsumerName
	this.Time = time.Now()
	return sendEvent(util.Config.ConnectorLogTopic, this)
}
