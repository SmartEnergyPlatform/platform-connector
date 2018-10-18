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
	"encoding/json"
	"log"

	iot_model "github.com/SmartEnergyPlatform/iot-device-repository/lib/model"
)

type DeviceLogMsg struct {
	Id    string `json:"id"`
	State string `json:"state"`
	Desc  string `json:"desc"`
}

//deprecated
func LogDeviceState(id string, state string, desc string) (err error) {
	msg, err := json.Marshal(DeviceLogMsg{Id: id, State: state, Desc: desc})
	if err != nil {
		return err
	}
	Produce(util.Config.KafkaDeviceLogTopic, string(msg))
	return
}

func (session *Session) LogDisconnect() {
	ids := []string{}
	session.Mux.Lock()
	for _, device := range session.UriCache {
		ids = append(ids, device.Device.Id)
	}
	session.Mux.Unlock()
	for _, id := range ids {
		err := DeviceLog{Device: id, Connected: false}.Send()
		if err != nil {
			log.Println("WARNING: unable to log device connection state ", err)
		}
		err = LogDeviceState(id, "disconnected", util.Config.KafkaConsumerTopic)
		if err != nil {
			log.Println("WARNING: unable to log device state ", err)
		}
	}
	err := GatewayLog{Gateway: session.Gateway, Connected: false}.Send()
	if err != nil {
		log.Println("WARNING: unable to log gateway connection state ", err)
	}
	err = LogDeviceState("gw-"+session.Gateway, "disconnected", util.Config.KafkaConsumerTopic)
	if err != nil {
		log.Println("WARNING: unable to log gateway state ", err)
	}
}

func (session *Session) LogDisconnectDevice(id string) {
	err := DeviceLog{Device: id, Connected: false}.Send()
	if err != nil {
		log.Println("WARNING: unable to log device connection state ", err)
	}
	err = LogDeviceState(id, "disconnected", util.Config.KafkaConsumerTopic)
	if err != nil {
		log.Println("WARNING: unable to log device state ", err)
	}
}

func (session *Session) LogGatewayConnect() {
	err := GatewayLog{Gateway: session.Gateway, Connected: true}.Send()
	if err != nil {
		log.Println("WARNING: unable to log gateway connection state ", err)
	}
	err = LogDeviceState("gw-"+session.Gateway, "connected", util.Config.KafkaConsumerTopic)
	if err != nil {
		log.Println("WARNING: unable to log gateway state ", err)
	}
}

func (session *Session) LogConnect() {
	ids := []string{}
	session.Mux.Lock()
	for _, device := range session.UriCache {
		ids = append(ids, device.Device.Id)
	}
	session.Mux.Unlock()
	for _, id := range ids {
		err := DeviceLog{Device: id, Connected: true}.Send()
		if err != nil {
			log.Println("WARNING: unable to log device connection state ", err)
		}
		err = LogDeviceState(id, "connected", util.Config.KafkaConsumerTopic)
		if err != nil {
			log.Println("WARNING: unable to log device state ", err)
		}
	}
	err := GatewayLog{Gateway: session.Gateway, Connected: true}.Send()
	if err != nil {
		log.Println("WARNING: unable to log gateway connection state ", err)
	}
	err = LogDeviceState("gw-"+session.Gateway, "connected", util.Config.KafkaConsumerTopic)
	if err != nil {
		log.Println("WARNING: unable to log gateway state ", err)
	}
}

func (session *Session) LogConnectDevice(id string) {
	err := DeviceLog{Device: id, Connected: true}.Send()
	if err != nil {
		log.Println("WARNING: unable to log device connection state ", err)
	}
	err = LogDeviceState(id, "connected", util.Config.KafkaConsumerTopic)
	if err != nil {
		log.Println("WARNING: unable to log device state ", err)
	}
}

func (session *Session) LogCreated(devices []iot_model.DeviceInstance) {
	ids := []string{}
	for _, device := range devices {
		ids = append(ids, device.Id)
	}
	for _, id := range ids {
		err := LogDeviceState(id, "registered", util.Config.KafkaConsumerTopic)
		if err != nil {
			log.Println("WARNING: unable to log device state ", err)
		}
	}
}
