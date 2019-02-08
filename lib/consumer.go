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
	"github.com/SENERGY-Platform/iot-broker-client-lib"
	"github.com/SmartEnergyPlatform/platform-connector/util"
	"log"
)
var consumer *iot_broker_client_lib.Consumer

func InitConsumer() (err error) {
	consumer, err = iot_broker_client_lib.NewConsumer(util.Config.AmqpUrl, util.Config.ConsumerName, util.Config.ProtocolTopic, false, func(msg []byte) error {
		go HandleMessage(string(msg))
		return nil
	})
	return
}

func CloseConsumer(){
	consumer.Close()
}

type Envelope struct {
	DeviceId  string      `json:"device_id,omitempty"`
	ServiceId string      `json:"service_id,omitempty"`
	Value     interface{} `json:"value"`
}

func HandleMessage(msg string) {
	log.Println("consume kafka msg: ", msg)
	envelope := Envelope{}
	err := json.Unmarshal([]byte(msg), &envelope)
	if err != nil {
		log.Println("ERROR: ", err)
		return
	}
	payload, err := json.Marshal(envelope.Value)
	if err != nil {
		log.Println("ERROR: ", err)
		return
	}
	Sessions().Dispatch(envelope.DeviceId, string(payload))
}


func BindDevice(device string) (err error) {
	return consumer.Bind(device, "#")
}

func UnbindDevice(device string) (err error) {
	return consumer.Unbind(device, "#")
}

func ClearBindings() {
	err := consumer.ResetBindings()
	if err != nil {
		log.Println("WARNING: error while ClearBindings()", err)
	}
}
