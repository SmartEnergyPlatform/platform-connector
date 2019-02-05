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
	"github.com/SmartEnergyPlatform/platform-connector/model"
	"github.com/SmartEnergyPlatform/platform-connector/util"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/SmartEnergyPlatform/formatter-lib"
)

type MessageHandlerFunction func(session *Session, request Request)

var MessageHandler = map[string]MessageHandlerFunction{
	"clear":      clear,
	"commit":     commit,
	"put":        put,
	"disconnect": remove,
	"response":   response,
	"event":      event,
	"delete":     deleteHandler,
}

func clear(session *Session, request Request) {
	err := ClearGateway(session.Gateway, session.Cred)
	if err != nil {
		request.Error(err.Error())
		return
	}
	devices := []model.DeviceServiceEntity{}
	session.Mux.Lock()
	for _, device := range session.UriCache {
		devices = append(devices, device)
	}
	session.Mux.Unlock()

	for _, device := range devices {
		err = session.MuteEntity(device)
		if err != nil {
			request.Error(err.Error())
			return
		}
	}

	request.Respond("ok")
}

func commit(session *Session, request Request) {
	hash, ok := request.RawPayload.(string)
	if !ok {
		request.UserError("expect hash as string in payload")
		return
	}
	devices := []string{}
	session.Mux.Lock()
	for _, device := range session.UriCache {
		devices = append(devices, device.Device.Id)
	}
	session.Mux.Unlock()
	gateway := model.GatewayRef{Hash: hash, Devices: devices}
	err := CommitGateway(session.Gateway, gateway, session.Cred)
	if err != nil {
		request.Error(err.Error())
		return
	}
	request.Respond("ok")
}

func response(session *Session, request Request) {
	msg, err := json.Marshal(request.RawPayload)
	if err != nil {
		request.UserError("ERROR: cannot parse response msg: " + err.Error())
		return
	}
	ProduceKafka(util.Config.KafkaResponseTopic, string(msg))
	err = Produce(util.Config.KafkaResponseTopic, msg)
	if err != nil {
		log.Println("ERROR: unable to send response to platform", err)
	}
	request.Respond("ok")
}

func put(session *Session, request Request) {
	clientDevice := model.ConnectorDevice{}
	err := request.Payload(&clientDevice)
	if err != nil {
		log.Println("ERROR: put::Payload", err)
		request.Error(err.Error())
		return
	}
	entities, err := DeviceUrlToIotDevice(clientDevice.Uri, session.Cred)
	if err != nil {
		log.Println("ERROR: put::DeviceUrlToIotDevice", err)
		request.Error(err.Error())
		return
	}
	if len(entities) > 1 {
		request.UserError("found more than one device with the given uri '" + clientDevice.Uri + "'. please delete duplicate devices or change their URIs.")
		return
	}

	var entity model.DeviceServiceEntity
	if len(entities) == 0 {
		log.Println("create new device from ", clientDevice)
		instance, err := CreateIotDevice(clientDevice, session.Cred)
		if err != nil {
			log.Println("ERROR: put::CreateIotDevice", err)
			request.Error(err.Error())
			return
		}
		entity, err = DeviceInstanceToDeviceServiceEntity(instance, nil, session.Cred)
		if err != nil {
			log.Println("ERROR: put::DeviceInstanceToDeviceServiceEntity", err)
			request.Error(err.Error())
			return
		}
	}

	if len(entities) == 1 {
		entity, err = UpdateDevice(clientDevice, entities[0], session.Cred)
		if err != nil {
			log.Println("ERROR: put::UpdateDevice", err)
			entity = entities[0]
		}
	}

	err = session.ListenToEntity(entity)

	if err != nil {
		log.Println("ERROR: put::ListenToEntity", err)
		request.Error(err.Error())
		return
	}
	request.Respond("ok")
}

func remove(session *Session, request Request) {
	uri, ok := request.RawPayload.(string)
	if !ok {
		request.UserError("expect uri as string in payload")
		return
	}
	entity, err := session.GetEntity(uri)
	if err != nil {
		request.Error(err.Error())
		return
	}
	err = session.MuteEntity(entity)
	if err != nil {
		request.Error(err.Error())
		return
	}
	request.Respond("ok")
}

func deleteHandler(session *Session, request Request) {
	uri, ok := request.RawPayload.(string)
	if !ok {
		request.UserError("expect uri as string in payload")
		return
	}
	session.Mux.Lock()
	entity, ok := session.UriCache[uri]
	session.Mux.Unlock()
	if ok {
		err := session.MuteEntity(entity)
		if err != nil {
			request.Error(err.Error())
			return
		}
	}
	err := DeleteDeviceInstance(uri, session.Cred)
	if err != nil {
		request.Error(err.Error())
		return
	}
	request.Respond("ok")
}

func formatEvent(session *Session, deviceid string, serviceid string, event formatter_lib.EventMsg) (string, error) {
	err := session.Cred.EnsureAccess()
	if err != nil {
		return "", err
	}
	formater, err := session.GetFormater(session.Cred, deviceid, serviceid)
	if err != nil {
		return "", err
	}
	return formater.Transform(event)
}

func event(session *Session, request Request) {
	event := model.EventMessage{}
	err := request.Payload(&event)
	if err != nil {
		request.UserError(err.Error())
		log.Println("error event: ", err)
		return
	}
	session.Mux.Lock()
	entity, ok := session.UriCache[event.DeviceUri]
	session.Mux.Unlock()
	if !ok {
		errMsg := "error event: not listen to this device " + fmt.Sprint(event)
		request.UserError(errMsg)
		log.Println(errMsg)
		return
	}
	for _, service := range entity.Services {
		if service.Url == event.ServiceUri {
			serviceTopic := formatId(service.Id)
			prefixMsg := model.PrefixMessage{DeviceId: entity.Device.Id, ServiceId: service.Id}

			var eventValue interface{}
			formatedEvent, err := formatEvent(session, entity.Device.Id, service.Id, event.Value)
			if err != nil {
				log.Println("ERROR: formatEvent() ", err)
				request.Error(err.Error())
				return
			}
			err = json.Unmarshal([]byte(formatedEvent), &eventValue)
			if err != nil {
				log.Println("ERROR: formatedEvent unmarshaling ", err)
				request.Error(err.Error())
				return
			}

			prefixMsg.Value = eventValue

			jsonPrefixMsg, err := json.Marshal(prefixMsg)
			if err != nil {
				log.Println("ERROR: creating jsonPrefixMsg failed: ", err)
				request.Error(err.Error())
			} else {
				ProduceKafka(serviceTopic, string(jsonPrefixMsg))
				err = Produce(util.Config.EventTopic, jsonPrefixMsg, prefixMsg.DeviceId, prefixMsg.DeviceId)
				if err != nil {
					log.Println("ERROR: unable to send event to platform", err)
				}
				request.Respond("ok")
			}
			return
		}
	}
	log.Println("debug: ", entity, session.UriCache)
	request.UserError("no matching service to '" + event.ServiceUri + "' found in" + fmt.Sprintln(entity.Services))
}

func formatId(id string) string {
	return strings.Replace(id, "#", "_", -1)
}
