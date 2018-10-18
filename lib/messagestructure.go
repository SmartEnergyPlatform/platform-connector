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
	"reflect"
)

type Message struct {
	Status      int         `json:"status"`
	Handler     string      `json:"handler"`
	Token       string      `json:"token"`
	ContentType string      `json:"content_type"`
	Payload     interface{} `json:"payload"`
}

func (this Message) Str() string {
	errmsg := "ERROR: unable to create response"
	if this.Payload == nil {
		log.Println(errmsg + " (payload is nil)")
		return errmsg
	}
	temp, err := json.Marshal(this.Payload)
	if err != nil {
		log.Println(errmsg, err)
		return errmsg
	}
	var ct interface{}
	err = json.Unmarshal(temp, &ct)
	if err != nil {
		log.Println(errmsg, err)
		return errmsg
	}
	this.ContentType = reflect.TypeOf(ct).Kind().String()
	msg, err := json.Marshal(this)
	if err != nil {
		log.Println(errmsg, err)
		return errmsg
	}
	return string(msg)
}

type RawRequest struct {
	Handler string      `json:"handler,omitempty"`
	Token   string      `json:"token,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

func (session *Session) NewRequest(message string) (result Request, err error) {
	raw := RawRequest{}
	err = json.Unmarshal([]byte(message), &raw)
	if err != nil {
		log.Println("ERROR: unable to parse request from message ("+message+")", err)
		return
	}

	result.session = session
	result.Handler = raw.Handler
	result.Token = raw.Token
	if raw.Payload != nil {
		result.ContentType = reflect.TypeOf(raw.Payload).Kind().String()
	} else {
		result.ContentType = "nil"
	}
	result.RawPayload = raw.Payload
	return
}

type Request struct {
	session        *Session
	Handler        string
	Token          string
	ContentType    string
	RawPayload     interface{}
	_payloadstring []byte
}

func (this *Request) Payload(result interface{}) (err error) {
	if this._payloadstring == nil || len(this._payloadstring) == 0 {
		this._payloadstring, err = json.Marshal(this.RawPayload)
		if err != nil {
			return err
		}
	}
	return json.Unmarshal(this._payloadstring, result)
}

func (this *Request) Respond(payload interface{}) (err error) {
	return this.session.SendResponse(Message{Payload: payload, Token: this.Token, Handler: "response", Status: 200})
}

func (this *Request) Error(msg string) (err error) {
	return this.session.SendError(Message{Payload: msg, Token: this.Token, Handler: "response", Status: 500})
}

func (this *Request) UserError(msg string) (err error) {
	return this.session.SendError(Message{Payload: msg, Token: this.Token, Handler: "response", Status: 400})
}
