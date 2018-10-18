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

package model

import (
	"github.com/SmartEnergyPlatform/formatter-lib"
	iot_model "github.com/SmartEnergyPlatform/iot-device-repository/lib/model"
)

type DeviceServiceEntity struct {
	Device   iot_model.DeviceInstance `json:"device"`
	Services []ShortService           `json:"services"`
}

type ShortService struct {
	Id          string `json:"id,omitempty"`
	ServiceType string `json:"service_type,omitempty"` // "Actuator" || "Sensor"
	Url         string `json:"url,omitempty"`
}

type ShortDeviceType struct {
	Id       string         `json:"id,omitempty"`
	Services []ShortService `json:"services,omitempty"`
}

type ConnectorDevice struct {
	IotType string   `json:"iot_type"`
	Uri     string   `json:"uri"` //device url should be unique for the user (even if multiple connector clients of the same kind are used) for example:  <<MAC>>+<<local_device_id>>
	Name    string   `json:"name"`
	Tags    []string `json:"tags"` // tag = <<id>>:<<display>>
}

type EventMessage struct {
	DeviceUri  string               `json:"device_uri"`
	ServiceUri string               `json:"service_uri"`
	Value      formatter_lib.EventMsg `json:"value"`
}

type PrefixMessage struct {
	DeviceId  string      `json:"device_id,omitempty"`
	ServiceId string      `json:"service_id,omitempty"`
	Value     interface{} `json:"value"`
}

type GatewayRef struct {
	Id      string   `json:"id,omitempty"`
	Devices []string `json:"devices,omitempty"`
	Hash    string   `json:"hash,omitempty"`
}

type Gateway struct {
	Id      string                     `json:"id,omitempty"`
	Name    string                     `json:"name,omitempty"`
	Hash    string                     `json:"hash,omitempty"`
	Devices []iot_model.DeviceInstance `json:"devices,omitempty"`
}
