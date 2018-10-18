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
	"bytes"
	"github.com/SmartEnergyPlatform/platform-connector/model"
	"github.com/SmartEnergyPlatform/platform-connector/util"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	iot_model "github.com/SmartEnergyPlatform/iot-device-repository/lib/model"
)

func GetGateway(id string, cred *Credentials) (gateway model.Gateway, err error) {
	resp, err := cred.Get(util.Config.IotRepoUrl + "/gateway/" + url.QueryEscape(id) + "/provide")
	if err != nil {
		log.Println("ERROR on GetGateway()", err)
		return gateway, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&gateway)
	if err != nil {
		log.Println("ERROR on GetGateway() json decode", err)
	}
	return gateway, err
}

func ClearGateway(id string, cred *Credentials) (err error) {
	var resp *http.Response
	for i := 0; i < 30; i++ {
		resp, err = cred.Post(util.Config.IotRepoUrl+"/gateway/"+url.QueryEscape(id)+"/clear", "application/json", nil)
		if resp.StatusCode == http.StatusPreconditionFailed {
			time.Sleep(1 * time.Second) //retry
			continue
		} else {
			break
		}
	}

	if err != nil {
		log.Println("error while doing ClearGateway() http request: ", err)
		return err
	}
	resp.Body.Close()
	return
}

func CommitGateway(id string, gateway model.GatewayRef, cred *Credentials) (err error) {
	var resp *http.Response
	for i := 0; i < 30; i++ {
		b := new(bytes.Buffer)
		err = json.NewEncoder(b).Encode(gateway)
		if err != nil {
			return err
		}
		resp, err = cred.Post(util.Config.IotRepoUrl+"/gateway/"+url.QueryEscape(id)+"/commit", "application/json", b)
		if resp.StatusCode == http.StatusPreconditionFailed {
			time.Sleep(1 * time.Second) //retry
			continue
		} else {
			break
		}
	}
	if err != nil {
		log.Println("error while doing CommitGateway() http request: ", err)
		return err
	}
	resp.Body.Close()
	return
}

func GetDeviceType(id string, cred *Credentials) (dt model.ShortDeviceType, err error) {
	resp, err := cred.Get(util.Config.IotRepoUrl + "/deviceType/" + url.QueryEscape(id))
	if err != nil {
		log.Println("ERROR on GetDeviceType()", err)
		return dt, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&dt)
	if err != nil {
		log.Println("ERROR on GetDeviceType() json decode", err)
	}
	return dt, err
}

func DeviceInstanceToDeviceServiceEntity(device iot_model.DeviceInstance, dtCache *map[string]model.ShortDeviceType, cred *Credentials) (entity model.DeviceServiceEntity, err error) {
	if dtCache == nil {
		dtCache = &map[string]model.ShortDeviceType{}
	}
	dt, ok := (*dtCache)[device.DeviceType]
	if !ok {
		dt, err = GetDeviceType(device.DeviceType, cred)
		if err != nil {
			log.Println("ERROR in DeviceInstanceToDeviceServiceEntity()::GetDeviceType()", device)
			return
		}
		(*dtCache)[device.DeviceType] = dt
	}
	return model.DeviceServiceEntity{Device: device, Services: dt.Services}, err
}

func DeviceUrlToIotDevice(deviceUrl string, cred *Credentials) (entities []model.DeviceServiceEntity, err error) {
	resp, err := cred.Get(util.Config.IotRepoUrl + "/url_to_devices/" + url.QueryEscape(deviceUrl))
	if err != nil {
		log.Println("error on ConnectorDeviceToIotDevice", err)
		return entities, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&entities)
	if err != nil {
		log.Println("error on ConnectorDeviceToIotDevice", err)
		return entities, err
	}
	//log.Println("debug: DeviceUrlToIotDevice() = ", entities)
	return
}

type IotErrorMessage struct {
	StatusCode int      `json:"status_code,omitempty"`
	Message    string   `json:"message"`
	ErrorCode  string   `json:"error_code,omitempty"`
	Detail     []string `json:"detail,omitempty"`
}

func CreateIotDevice(device model.ConnectorDevice, cred *Credentials) (result iot_model.DeviceInstance, err error) {
	typeid := device.IotType

	if typeid == "" {
		return result, errors.New("empty iot_type")
	}
	resp, err := cred.Get(util.Config.IotRepoUrl + "/ui/deviceInstance/resourceSkeleton/" + url.QueryEscape(typeid))
	if err != nil {
		log.Println("error on CreateIotDevice() resourceSkeleton", err)
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errmsg := IotErrorMessage{}
		json.NewDecoder(resp.Body).Decode(&errmsg)
		err = errors.New("error while creating device skeleton: " + errmsg.Message)
		log.Println("ERROR: CreateIotDevice()", err, errmsg)
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Println("error on CreateIotDevice() decode", err)
		return result, err
	}
	result.Name = device.Name
	result.Url = device.Uri
	result.Tags = device.Tags

	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(result)
	if err != nil {
		return result, err
	}
	resp, err = cred.Post(util.Config.IotRepoUrl+"/deviceInstance", "application/json", b)
	if err != nil {
		log.Println("error on CreateIotDevice() create in repository", err)
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errmsg := IotErrorMessage{}
		json.NewDecoder(resp.Body).Decode(&errmsg)
		err = errors.New("error while creating new device: " + errmsg.Message)
		log.Println("ERROR: CreateIotDevice()", err, errmsg)
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}

func UpdateDevice(new model.ConnectorDevice, old model.DeviceServiceEntity, cred *Credentials) (result model.DeviceServiceEntity, err error) {
	clientTags := IndexTags(new.Tags)
	platformTags := IndexTags(old.Device.Tags)
	mergedTags := MergeTagIndexes(platformTags, clientTags)
	tagsChanged := !reflect.DeepEqual(platformTags, mergedTags)
	nameChanged := new.Name != old.Device.Name

	result = old
	if tagsChanged || nameChanged {
		if nameChanged {
			result.Device.Name = new.Name
		}
		if tagsChanged {
			result.Device.Tags = TagIndexToTagList(mergedTags)
		}
		err = UpdateDeviceInstance(result.Device, cred)
	}
	return
}

func UpdateDeviceInstance(device iot_model.DeviceInstance, cred *Credentials) (err error) {
	err = cred.EnsureAccess()
	if err != nil {
		return
	}
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(device)
	if err != nil {
		return err
	}
	resp, err := cred.Post(util.Config.IotRepoUrl+"/deviceInstance/"+url.QueryEscape(device.Id), "application/json", b)
	if err != nil {
		log.Println("error while doing UpdateDevice() http request: ", err)
		return err
	}
	defer resp.Body.Close()
	return
}

func IndexTags(tags []string) (result map[string]string) {
	result = map[string]string{}
	for _, tag := range tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) != 2 {
			log.Println("ERROR: wrong tag syntax; ", tag)
			continue
		}
		result[parts[0]] = parts[1]
	}
	return result
}

func TagIndexToTagList(index map[string]string) (tags []string) {
	for key, value := range index {
		tags = append(tags, key+":"+value)
	}
	return
}

func MergeTagIndexes(platform map[string]string, client map[string]string) (result map[string]string) {
	result = map[string]string{}
	for key, value := range platform {
		result[key] = value
	}
	for key, value := range client {
		result[key] = value
	}
	return
}

func DeleteDeviceInstance(uri string, cred *Credentials) (err error) {
	err = cred.EnsureAccess()
	if err != nil {
		return
	}
	entities, err := DeviceUrlToIotDevice(uri, cred)
	if err != nil {
		return err
	}
	if len(entities) != 1 {
		return errors.New("cant find exactly one device with given uri")
	}
	_, err = cred.Delete(util.Config.IotRepoUrl+"/deviceInstance/"+url.QueryEscape(entities[0].Device.Id), nil)
	return
}

func CheckExecutionAccess(resource string, cred *Credentials) (err error) {
	resp, err := cred.Get("/check/access/" + url.QueryEscape(resource) + "/execute")
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("user may not execute events for the resource")
	}
	return nil
}
