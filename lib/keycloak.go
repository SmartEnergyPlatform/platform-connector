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
	"errors"
	"net/http"
	"net/url"
	"time"
)

type OpenidToken struct {
	AccessToken      string    `json:"access_token"`
	ExpiresIn        float64   `json:"expires_in"`
	RefreshExpiresIn float64   `json:"refresh_expires_in"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	RequestTime      time.Time `json:"-"`
}

func GetOpenidToken(username, password string, token *OpenidToken) (err error) {
	requesttime := time.Now()
	resp, err := http.PostForm(util.Config.AuthEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {util.Config.AuthClientId},
		"client_secret": {util.Config.AuthClientSecret},
		"username":      {username},
		"password":      {password},
		"grant_type":    {"password"},
	})

	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		err = errors.New("access denied")
		return
	}
	err = json.NewDecoder(resp.Body).Decode(token)
	token.RequestTime = requesttime
	return
}

func RefreshOpenidToken(token *OpenidToken) (err error) {
	requesttime := time.Now()
	resp, err := http.PostForm(util.Config.AuthEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {util.Config.AuthClientId},
		"client_secret": {util.Config.AuthClientSecret},
		"refresh_token": {token.RefreshToken},
		"grant_type":    {"refresh_token"},
	})

	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		err = errors.New("access denied")
		return
	}
	err = json.NewDecoder(resp.Body).Decode(token)
	token.RequestTime = requesttime
	return
}
