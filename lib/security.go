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
	"github.com/SmartEnergyPlatform/platform-connector/util"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

type Credentials struct {
	User         string          `json:"user"`
	Pw           string          `json:"pw"`
	Token        string          `json:"token"` //responsetoken if needed
	Gateway      string          `json:"gid"`
	Openid       *OpenidToken    `json:"-"`
	ErrorHandler func(err error) `json:"-"`
}

func (this *Credentials) EnsureAccess() (err error) {
	defer func() {
		if err != nil && this.ErrorHandler != nil {
			log.Println("ERROR: EnsureAccess() with ErrorHandler", err)
			this.ErrorHandler(err)
		}
	}()
	if this.Openid == nil {
		this.Openid = &OpenidToken{}
	}
	duration := time.Now().Sub(this.Openid.RequestTime).Seconds()

	if this.Openid.AccessToken != "" && this.Openid.ExpiresIn-util.Config.AuthExpirationTimeBuffer > duration {
		return
	}

	if this.Openid.RefreshToken != "" && this.Openid.RefreshExpiresIn-util.Config.AuthExpirationTimeBuffer < duration {
		log.Println("refresh token", this.Openid.RefreshExpiresIn, duration)
		err = RefreshOpenidToken(this.Openid)
		if err != nil {
			log.Println("WARNING: unable to use refreshtoken", err)
		} else {
			return
		}
	}

	log.Println("get new access token")
	err = GetOpenidToken(this.User, this.Pw, this.Openid)
	if err != nil {
		log.Println("ERROR: unable to get new access token", err)
		this.Openid = &OpenidToken{}
	}
	return
}

func (this *Credentials) Get(url string) (resp *http.Response, err error) {
	err = this.EnsureAccess()
	if err != nil {
		return
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", this.Openid.TokenType+" "+this.Openid.AccessToken)

	resp, err = http.DefaultClient.Do(req)

	if err == nil && resp.StatusCode == 401 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		resp.Body.Close()
		log.Println(buf.String())
		err = errors.New("access denied")
	}
	return
}

func (this *Credentials) Post(url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	err = this.EnsureAccess()
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", this.Openid.TokenType+" "+this.Openid.AccessToken)
	req.Header.Set("Content-Type", contentType)

	resp, err = http.DefaultClient.Do(req)

	if err == nil && resp.StatusCode == 401 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		resp.Body.Close()
		log.Println(buf.String())
		err = errors.New("access denied")
	}
	return
}

func (this *Credentials) Delete(url string, body io.Reader) (resp *http.Response, err error) {
	err = this.EnsureAccess()
	if err != nil {
		return
	}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", this.Openid.TokenType+" "+this.Openid.AccessToken)

	resp, err = http.DefaultClient.Do(req)

	if err == nil && resp.StatusCode == 401 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		resp.Body.Close()
		log.Println(buf.String())
		err = errors.New("access denied")
	}
	return
}
