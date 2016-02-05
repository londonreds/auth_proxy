package main

import (
	"errors"
	"fmt"
	"github.com/buzzfeed/auth_proxy/providers"
	"net/http"
	"strings"
	"time"
)

type UserInfoHandler struct {
	api     *AuthApi
	refresh time.Duration
}

type UserInfo map[string]string

func (m *UserInfoHandler) Handle(rw http.ResponseWriter, req *http.Request, ses *providers.SessionState) error {
	// by default, check to see if user information exists in a cookie
	data, err := m.LoadCookiedData(req)
	if err == nil {
		goto Save
	}
	data, err = m.FetchUserInfo(ses)
	if err != nil {
		return err
	}

Save:
	m.WriteForwardedHeaders(req, data)
	// TODO: add in a method to write cookie information
	return nil
}

func (m *UserInfoHandler) LoadCookiedData(req *http.Request) (UserInfo, error) {
	return nil, errors.New("Not implemented")
}

func (m *UserInfoHandler) FetchUserInfo(ses *providers.SessionState) (UserInfo, error) {
	res, err := m.api.FetchUserInfo(ses.User)
	if err != nil {
		return nil, err
	}

	userInfo := make(map[string]string, 3)
	userInfo["Roles"] = strings.Join(res.Roles, ",")
	userInfo["Username"] = res.Username
	userInfo["Email"] = res.Email

	return userInfo, nil
}

func (m *UserInfoHandler) WriteForwardedHeaders(req *http.Request, data UserInfo) {
	for key, value := range data {
		headerKey := fmt.Sprintf("X-Forwarded-%s", key)
		req.Header.Add(headerKey, value)
	}
}
