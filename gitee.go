/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package gitee

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/apache/incubator-answer-plugins/util"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/apache/incubator-answer/pkg/checker"
	"github.com/apache/incubator-answer/plugin"
	"github.com/joyoyao/connector-gitee/i18n"
	"github.com/segmentfault/pacman/log"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
)

var (
	replaceUsernameReg = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	//go:embed  info.yaml
	Info embed.FS
)

type Connector struct {
	Config *ConnectorConfig
}

type ConnectorConfig struct {
	Name         string `json:"name"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func init() {
	plugin.Register(&Connector{
		Config: &ConnectorConfig{},
	})
}

func (g *Connector) Info() plugin.Info {
	info := &util.Info{}
	info.GetInfo(Info)

	return plugin.Info{
		Name:        plugin.MakeTranslator(i18n.InfoName),
		SlugName:    info.SlugName,
		Description: plugin.MakeTranslator(i18n.InfoDescription),
		Author:      info.Author,
		Version:     info.Version,
		Link:        info.Link,
	}
}

func (g *Connector) ConnectorLogoSVG() string {
	return "PHN2ZyB0PSIxNzM1MjY3MTEwOTE4IiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwMjQgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjQ1NDYiIHdpZHRoPSIxMjgiIGhlaWdodD0iMTI4Ij48cGF0aCBkPSJNNTEyIDEwMjEuNzI0NDQ0NDVBNTA5LjcyNDQ0NDQ1IDUwOS43MjQ0NDQ0NSAwIDEgMSA1MTIgMi4yNzU1NTU1NWE1MDkuNzI0NDQ0NDUgNTA5LjcyNDQ0NDQ1IDAgMCAxIDAgMTAxOS40NDg4ODg5eiBtMjU3Ljk5MzM4NjY3LTU2Ni4zNzY2NzU1Nkg0ODAuNTQyNzJhMjUuMTk0OTUxMTEgMjUuMTk0OTUxMTEgMCAwIDAtMjUuMTk0OTUxMTEgMjUuMTk0OTUxMTF2NjIuOTE0NTZjMCAxMy45MDgxOTU1NSAxMS4yODY3NTU1NSAyNS4xOTQ5NTExMSAyNS4xMjIxMzMzMyAyNS4xOTQ5NTExMWgxNzYuMjE5MDIyMjNjMTMuOTgxMDEzMzMgMCAyNS4xOTQ5NTExMSAxMS4yODY3NTU1NSAyNS4xOTQ5NTExIDI1LjEyMjEzMzM0djEyLjU5NzQ3NTU1YzAgNDEuNzI0NTg2NjctMzMuNzg3NDQ4ODkgNzUuNTEyMDM1NTUtNzUuNTEyMDM1NTUgNzUuNTEyMDM1NTVIMzY3LjIzODI1Nzc4YTI1LjE5NDk1MTExIDI1LjE5NDk1MTExIDAgMCAxLTI1LjEyMjEzMzMzLTI1LjEyMjEzMzMzVjQxNy42MjgxNmMwLTQxLjcyNDU4NjY3IDMzLjc4NzQ0ODg5LTc1LjUxMjAzNTU1IDc1LjQzOTIxNzc3LTc1LjUxMjAzNTU1aDM1Mi40MzgwNDQ0NWMxMy44MzUzNzc3OCAwIDI1LjEyMjEzMzMzLTExLjI4Njc1NTU1IDI1LjEyMjEzMzMzLTI1LjE5NDk1MTEydi02Mi45MTQ1NmEyNS4xOTQ5NTExMSAyNS4xOTQ5NTExMSAwIDAgMC0yNS4xMjIxMzMzMy0yNS4xOTQ5NTExMWgtMzUyLjQzODA0NDQ1YTE4OC43NDM2OCAxODguNzQzNjggMCAwIDAtMTg4Ljc0MzY4IDE4OC44MTY0OTc3OHYzNTIuMzY1MjI2NjdjMCAxMy45MDgxOTU1NSAxMS4yODY3NTU1NSAyNS4xOTQ5NTExMSAyNS4xOTQ5NTExMSAyNS4xOTQ5NTExMWgzNzEuMjI1MDMxMTJhMTY5Ljg4Mzg3NTU1IDE2OS44ODM4NzU1NSAwIDAgMCAxNjkuOTU2NjkzMzMtMTY5Ljg4Mzg3NTU2VjQ4MC41NDI3MmEyNS4xOTQ5NTExMSAyNS4xOTQ5NTExMSAwIDAgMC0yNS4xOTQ5NTExMS0yNS4xOTQ5NTExMXoiIGZpbGw9IiNDNzFEMjMiIHAtaWQ9IjQ1NDciPjwvcGF0aD48L3N2Zz4="
}

func (g *Connector) ConnectorName() plugin.Translator {
	if len(g.Config.Name) > 0 {
		return plugin.MakeTranslator(g.Config.Name)
	}
	return plugin.MakeTranslator(i18n.ConnectorName)
}

func (g *Connector) ConnectorSlugName() string {
	return "gitee"
}

func (g *Connector) ConnectorSender(ctx *plugin.GinContext, receiverURL string) (redirectURL string) {
	oauth2Config := &oauth2.Config{
		ClientID:     g.Config.ClientID,
		ClientSecret: g.Config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://gitee.com/oauth/authorize",
			TokenURL: "https://gitee.com/oauth/token",
		},
		RedirectURL: receiverURL,
		Scopes:      strings.Split("user_info,emails", ","),
	}
	return oauth2Config.AuthCodeURL("state")
}

func (g *Connector) ConnectorReceiver(ctx *plugin.GinContext, receiverURL string) (userInfo plugin.ExternalLoginUserInfo, err error) {
	code := ctx.Query("code")
	// Exchange code for token
	oauth2Config := &oauth2.Config{
		ClientID:     g.Config.ClientID,
		ClientSecret: g.Config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://gitee.com/oauth/authorize",
			TokenURL:  "https://gitee.com/oauth/token",
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
		RedirectURL: receiverURL,
	}
	token, err := oauth2Config.Exchange(context.Background(), code)
	if err != nil {
		return userInfo, fmt.Errorf("code exchange failed: %s", err.Error())
	}

	// Exchange token for user info
	client := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token.AccessToken},
	))
	client.Timeout = 15 * time.Second

	response, err := client.Get("https://gitee.com/api/v5/user")
	if err != nil {
		return userInfo, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)

	userInfo = plugin.ExternalLoginUserInfo{
		MetaInfo: string(data),
	}

	userInfo.ExternalID = gjson.GetBytes(data, "id").String()
	if len(userInfo.ExternalID) == 0 {
		log.Errorf("fail to get user id")
		return userInfo, nil
	}
	userInfo.DisplayName = gjson.GetBytes(data, "name").String()
	userInfo.Username = gjson.GetBytes(data, "login").String()
	userInfo.Avatar = gjson.GetBytes(data, "avatar_url").String()
	emailResponse, err := client.Get("https://gitee.com/api/v5/emails")
	if err != nil {
		return userInfo, fmt.Errorf("failed getting user email: %s", err.Error())
	}
	defer emailResponse.Body.Close()
	emailData, _ := io.ReadAll(emailResponse.Body)
	userInfo.Email = gjson.GetBytes(emailData, "0.email").String()
	userInfo = g.formatUserInfo(userInfo)
	return userInfo, nil
}

func (g *Connector) formatUserInfo(userInfo plugin.ExternalLoginUserInfo) (
	userInfoFormatted plugin.ExternalLoginUserInfo) {
	userInfoFormatted = userInfo
	if checker.IsInvalidUsername(userInfoFormatted.Username) {
		userInfoFormatted.Username = replaceUsernameReg.ReplaceAllString(userInfoFormatted.Username, "_")
	}

	usernameLength := utf8.RuneCountInString(userInfoFormatted.Username)
	if usernameLength < 4 {
		userInfoFormatted.Username = userInfoFormatted.Username + strings.Repeat("_", 4-usernameLength)
	} else if usernameLength > 30 {
		userInfoFormatted.Username = string([]rune(userInfoFormatted.Username)[:30])
	}
	return userInfoFormatted
}

func (g *Connector) ConfigFields() []plugin.ConfigField {
	fields := make([]plugin.ConfigField, 0)
	fields = append(fields, createTextInput("name",
		i18n.ConfigNameTitle, i18n.ConfigNameDescription, g.Config.Name, true))
	fields = append(fields, createTextInput("client_id",
		i18n.ConfigClientIDTitle, i18n.ConfigClientIDDescription, g.Config.ClientID, true))
	fields = append(fields, createTextInput("client_secret",
		i18n.ConfigClientSecretTitle, i18n.ConfigClientSecretDescription, g.Config.ClientSecret, true))
	return fields
}

func createTextInput(name, title, desc, value string, require bool) plugin.ConfigField {
	return plugin.ConfigField{
		Name:        name,
		Type:        plugin.ConfigTypeInput,
		Title:       plugin.MakeTranslator(title),
		Description: plugin.MakeTranslator(desc),
		Required:    require,
		UIOptions: plugin.ConfigFieldUIOptions{
			InputType: plugin.InputTypeText,
		},
		Value: value,
	}
}

func (g *Connector) ConfigReceiver(config []byte) error {
	c := &ConnectorConfig{}
	_ = json.Unmarshal(config, c)
	g.Config = c
	return nil
}
