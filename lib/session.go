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
	"errors"
	"time"

	"github.com/SmartEnergyPlatform/formatter-lib"

	"log"

	"sync"

	"github.com/SmartEnergyPlatform/platform-connector/model"

	"encoding/json"

	"net"

	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

type Session struct {
	Id                         string
	Prefixes                   []string
	Cred                       *Credentials
	ws                         *websocket.Conn
	wsMux                      sync.Mutex
	stopPing                   chan bool
	Mux                        sync.Mutex
	UriCache                   map[string]model.DeviceServiceEntity
	eventTransformerCollection map[string]formatter_lib.EventTransformer
	ConsecutiveErrors          int64
	Gateway                    string
	closing                    bool
	activePing                 bool
}

func NewSession(connection *websocket.Conn) {
	connection.SetPongHandler(func(msg string) error {
		err := connection.SetReadDeadline(time.Now().Add(time.Second * time.Duration(util.Config.WsTimeout)))
		if err != nil {
			log.Println("ERROR in SetReadDeadline: (", msg, ") ", err.Error())
		}
		return nil
	})

	cred, err := WsHandshake(connection)
	if err != nil {
		log.Println("handshake error", err)
		connection.Close()
		return
	}
	gateway, err := GetGateway(cred.Gateway, cred)
	if err != nil {
		log.Println("error while geting gateway", err)
		connection.Close()
		return
	}
	if gateway.Id == "" {
		log.Println("gateway error", err)
		connection.Close()
		return
	}
	uuid := uuid.NewV4()
	if err != nil {
		connection.Close()
		return
	}
	session := Session{
		Id:                         uuid.String(),
		Cred:                       cred,
		ws:                         connection,
		UriCache:                   map[string]model.DeviceServiceEntity{},
		eventTransformerCollection: map[string]formatter_lib.EventTransformer{},
		ConsecutiveErrors:          0,
		closing:                    false,
		Gateway:                    gateway.Id,
		stopPing:                   make(chan bool),
		activePing:                 true,
	}

	Sessions().Register(&session)

	cred.ErrorHandler = func(err error) {
		session.Close("auth error: " + err.Error())
	}

	connection.SetPingHandler(func(msg string) error {
		connection.SetReadDeadline(time.Now().Add(time.Second * time.Duration(util.Config.WsTimeout)))
		session.activePing = false
		err := session.SendWsMsg(websocket.PongMessage, msg)
		if err != nil {
			log.Println("ERROR: SetPingHandler::SendWsMsg ", err)
		}
		if err != nil && err != websocket.ErrCloseSent {
			e, ok := err.(net.Error)
			if !ok {
				log.Println("ERROR: SetPingHandler::SendWsMsg is not a net error", err)
				return err
			} else if !e.Temporary() {
				log.Println("ERROR: SetPingHandler::SendWsMsg is permanent", e)
				return err
			}
		}
		return nil
	})

	cache := &map[string]model.ShortDeviceType{}
	for _, device := range gateway.Devices {
		entity, err := DeviceInstanceToDeviceServiceEntity(device, cache, session.Cred)
		if err != nil {
			session.Close("ERROR while creating device-service-entity: " + err.Error())
			return
		}
		err = session.ListenToEntity(entity)
		if err != nil {
			session.Close("ERROR: while trying to listen to device-service-entity: " + err.Error())
			return
		}
	}

	session.SendResponse(Message{Payload: map[string]string{"gid": gateway.Id, "hash": gateway.Hash}, Token: cred.Token, Status: 200, Handler: "response"})
	session.LogGatewayConnect()
	session.Start()
}

func (session *Session) Start() {
	session.startPing()

	closer := func(msg *string) {
		session.Close(*msg)
	}

	go func() {
		closeMsg := "read-error"
		defer closer(&closeMsg)
		for {
			err := session.ws.SetReadDeadline(time.Now().Add(time.Second * time.Duration(util.Config.WsTimeout)))
			if err != nil {
				log.Println("ERROR in SetReadDeadline: ", err.Error())
			}
			msgtype, buff, err := session.ws.ReadMessage()
			if err != nil {
				log.Println("lost ws connection: ", err, "\nmsg: "+string(buff))
				closeMsg = "read-error: " + err.Error() + "\nmsg: " + string(buff)
				return
			}
			if msgtype == websocket.TextMessage {
				session.HandleMessage(string(buff))
			}
		}
	}()
}

func (session *Session) HandleMessage(message string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR: Recovered in HandleMessage", r)
		}
	}()
	log.Println("handle message: ", session.Cred.User, session.Id, message)

	request, err := session.NewRequest(message)

	if err != nil {
		session.SendError(Message{Payload: "unable to parse request from message (" + message + ")", Token: "", Handler: "response", Status: 400})
	}

	if handler, ok := MessageHandler[request.Handler]; ok {
		handler(session, request)
	} else {
		log.Println("unknown handler: ", request)
		session.SendError(Message{Payload: "unknown handler in request (" + message + ")", Token: request.Token, Handler: "response", Status: 400})
	}
}

func (session *Session) startPing() {
	ticker := time.NewTicker(time.Second * time.Duration(util.Config.WsPingperiod))
	go func() {
		for {
			select {
			case <-session.stopPing:
				log.Println("stop signal for ping")
				ticker.Stop()
				return
			case t := <-ticker.C:
				if session.activePing {
					if err := session.SendWsMsg(websocket.PingMessage, t.String()); err != nil {
						log.Println("ERROR on ws ping: ", err)
						session.Close("ERROR on ws ping: " + err.Error())
					}
				} else {
					ticker.Stop()
				}
			}
		}
	}()
}

func (session *Session) Close(reason string) {
	log.Println("close connection to gateway: ", session.Gateway, "; is already closing: ", session.closing)
	if !session.closing {
		session.closing = true
		log.Println("send closing msg to ", session.Gateway, session.SendClose(reason))
		log.Println("close websocket to", session.Gateway, session.ws.Close())
		session.LogDisconnect()
		Sessions().Deregister(session)
		//non-blocking-channel-send to stop ping ticker
		select {
		case session.stopPing <- true:
		}
	}
}

func WsHandshake(conn *websocket.Conn) (credentials *Credentials, err error) {
	credentials = &Credentials{}
	err = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(util.Config.WsTimeout)))
	if err != nil {
		return
	}
	msgType, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}
	log.Println("debug: handshake: ", msgType, string(msg))
	err = json.Unmarshal(msg, credentials)
	if err != nil {
		return
	}
	err = credentials.EnsureAccess()
	if err != nil {
		log.Println("ERROR: WsHandshake::EnsureAccess", err)
		err = errors.New("authentication error")
	}
	return
}

func (session *Session) SendResponse(response Message) error {
	session.ConsecutiveErrors = 0
	return session.SendWsMsg(websocket.TextMessage, response.Str())

}

func (session *Session) SendError(response Message) (err error) {
	session.ConsecutiveErrors++
	err = session.SendWsMsg(websocket.TextMessage, response.Str())
	if session.ConsecutiveErrors > util.Config.MaxConsecutiveErrors && util.Config.MaxConsecutiveErrors >= 0 {
		session.Close("ERROR: max consecutive error count exceeded")
		return err
	}
	return
}

func (session *Session) SendClose(reason string) (err error) {
	session.wsMux.Lock()
	defer session.wsMux.Unlock()
	return session.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason))
}

func (session *Session) SendWsMsg(msgType int, msg string) (err error) {
	if msgType != websocket.PingMessage && msgType != websocket.PongMessage {
		log.Println("send msg to ws: ", session.Cred.User, session.Id, msg)
	}
	session.wsMux.Lock()
	defer session.wsMux.Unlock()
	return session.ws.WriteMessage(msgType, []byte(msg))
}

func (session *Session) GetFormater(cred *Credentials, deviceid string, serviceid string) (transformer formatter_lib.EventTransformer, err error) {
	key := deviceid + "." + serviceid
	transformer, ok := session.eventTransformerCollection[key]
	if !ok {
		transformer, err = formatter_lib.NewTransformer(util.Config.IotRepoUrl, cred, deviceid, serviceid)
		if err != nil {
			log.Println("ERROR: unable to create new format transformer for device:", deviceid, "and service:", serviceid, err)
			return transformer, err
		}
		session.eventTransformerCollection[key] = transformer
	}
	return transformer, err
}

func removeDuplicates(in []string) (out []string) {
	found := make(map[string]bool)
	for _, x := range in {
		if !found[x] {
			found[x] = true
			out = append(out, x)
		}
	}
	return
}

func removeSliceString(in []string, str string) (out []string) {
	for _, element := range in {
		if element != str {
			out = append(out, element)
		}
	}
	return
}

func (session *Session) ListenToEntity(entity model.DeviceServiceEntity) (err error) {
	session.Mux.Lock()
	session.UriCache[entity.Device.Url] = entity
	prefix := entity.Device.Id
	session.Prefixes = append(session.Prefixes, prefix)
	session.Prefixes = removeDuplicates(session.Prefixes)
	err = Sessions().RegisterPrefix(session, prefix)
	if err != nil {
		delete(session.UriCache, entity.Device.Url)
		session.Prefixes = removeSliceString(session.Prefixes, prefix)
		session.Mux.Unlock()
	} else {
		session.Mux.Unlock()
		session.LogConnectDevice(entity.Device.Id)
	}
	return
}

func (session *Session) MuteEntity(entity model.DeviceServiceEntity) (err error) {
	session.Mux.Lock()
	delete(session.UriCache, entity.Device.Url)
	prefix := entity.Device.Id
	session.Prefixes = removeSliceString(session.Prefixes, prefix)
	err = Sessions().DeregisterPrefix(session, prefix)
	if err != nil {
		session.UriCache[entity.Device.Url] = entity
		session.Prefixes = append(session.Prefixes, prefix)
		session.Mux.Unlock()
	} else {
		session.Mux.Unlock()
		session.LogDisconnectDevice(entity.Device.Id)
	}
	return
}

func (session *Session) GetEntity(uri string) (entity model.DeviceServiceEntity, err error) {
	session.Mux.Lock()
	var ok bool
	entity, ok = session.UriCache[uri]
	session.Mux.Unlock()
	if !ok {
		err = errors.New("not listening to device with uri '" + uri + "'")
	}
	return
}

func (session *Session) SendCommand(msg string) (err error) {
	var parsedMsg interface{}
	err = json.Unmarshal([]byte(msg), &parsedMsg)
	if err != nil {
		log.Println("ERROR: command parsing: ", err)
		return err
	}
	return session.SendWsMsg(websocket.TextMessage, Message{Handler: "command", Payload: parsedMsg}.Str())
}
