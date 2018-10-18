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
	"log"
	"sync"
)

type SessionsCollection struct {
	mux      sync.Mutex
	index    map[string]map[string]*Session // device.sessionid (prefix is synonym to device id)
	sessions map[string]*Session
}

var sessionsCollection *SessionsCollection
var onceSessionsCollection sync.Once

func Sessions() *SessionsCollection {
	onceSessionsCollection.Do(func() {
		ConnectorLog{Connected: true}.Send()
		ClearPts()
		sessionsCollection = &SessionsCollection{
			index:    map[string]map[string]*Session{},
			sessions: map[string]*Session{},
		}
	})
	return sessionsCollection
}

func (this *SessionsCollection) GetSessions() (result map[string]*Session) {
	this.mux.Lock()
	defer this.mux.Unlock()
	result = this.sessions
	return
}

func (this *SessionsCollection) Dispatch(prefix string, msg string) {
	this.mux.Lock()
	for _, session := range this.index[prefix] {
		err := session.SendCommand(msg)
		if err != nil {
			log.Println("error ", prefix, session.Cred.User, err)
		}
	}
	this.mux.Unlock()
}

func (this *SessionsCollection) Register(session *Session) {
	this.mux.Lock()
	this.sessions[session.Id] = session
	this.mux.Unlock()
}

func (this *SessionsCollection) RegisterPrefix(session *Session, prefix string) (err error) {
	this.mux.Lock()
	if _, exists := this.index[prefix]; !exists {
		this.index[prefix] = map[string]*Session{}
		err = RegisterPts(prefix)
	}
	if err != nil {
		log.Println("ERROR in RegisterPts(): ", err)
	} else {
		this.index[prefix][session.Id] = session
	}
	this.mux.Unlock()
	return
}

func (this *SessionsCollection) Deregister(session *Session) {
	this.mux.Lock()
	delete(this.sessions, session.Id)
	for _, prefix := range session.Prefixes {
		delete(this.index[prefix], session.Id)
		if len(this.index[prefix]) == 0 {
			DeregisterPts(prefix)
			delete(this.index, prefix)
		}
	}
	this.mux.Unlock()
}

func (this *SessionsCollection) DeregisterPrefix(session *Session, prefix string) (err error) {
	this.mux.Lock()
	delete(this.index[prefix], session.Id)
	if len(this.index[prefix]) == 0 {
		err = DeregisterPts(prefix)
		if err != nil {
			log.Println("ERROR in DeregisterPts(): ", err)
		} else {
			delete(this.index, prefix)
		}
	}
	this.mux.Unlock()
	return
}

func (this *SessionsCollection) Close() {
	log.Println("shuting connector down...")
	this.mux.Lock()
	for _, sessionByPrefix := range this.index {
		for _, session := range sessionByPrefix {
			session.Close("connector shutdown")
		}
	}
	ConnectorLog{Connected: false}.Send()
	this.mux.Unlock()
}
