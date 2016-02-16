package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buzzfeed/auth_proxy/cookie"
	"github.com/buzzfeed/auth_proxy/providers"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type UserInfoHandler struct {
	api            *AuthApi
	cookieExpire   time.Duration
	cookieDomain   string
	cookieName     string
	cookieSeed     string
	cookieHttpOnly bool
	cookieSecure   bool
}

type UserInfo map[string]string

func (m *UserInfoHandler) Handle(rw http.ResponseWriter, req *http.Request, ses *providers.SessionState) error {
	// by default, check to see if user information exists in a cookie
	createCookie := true
	userInfo, err := m.LoadCookiedData(req)
	if err == nil {
		createCookie = false
		goto Save
	}
	userInfo, err = m.FetchUserInfo(ses.User)
	if err != nil {
		return err
	}

Save:
	m.WriteForwardedHeaders(req, userInfo)
	if createCookie {
		err := m.WriteCookie(rw, req, userInfo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *UserInfoHandler) LoadCookiedData(req *http.Request) (UserInfo, error) {
	c, err := req.Cookie(m.cookieName)
	if err != nil {
		return nil, errors.New("No cookie found")
	}

	value, _, ok := cookie.Validate(c, m.cookieSeed, m.cookieExpire)
	if !ok {
		return nil, errors.New("Cookie not valid")
	}

	cipher, err := cookie.NewCipher(m.cookieSeed)
	if err != nil {
		return nil, err
	}

	jsonString, err := cipher.Decrypt(value)
	if err != nil {
		return nil, err
	}

	userInfo := make(map[string]string, 3)
	err = json.Unmarshal([]byte(jsonString), &userInfo)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

func (m *UserInfoHandler) WriteCookie(rw http.ResponseWriter, req *http.Request, userInfo UserInfo) error {
	jsonStr, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	cipher, err := cookie.NewCipher(m.cookieSeed)
	if err != nil {
		return err
	}

	encryptedData, err := cipher.Encrypt(string(jsonStr))
	if err != nil {
		return err
	}
	cookieData := cookie.SignedValue(m.cookieSeed, m.cookieName, encryptedData, time.Now())

	domain := req.Host
	if h, _, err := net.SplitHostPort(domain); err == nil {
		domain = h
	}
	if m.cookieDomain != "" {
		if !strings.HasSuffix(domain, m.cookieDomain) {
			log.Printf("Warning: request host is %q but using configured cookie domain of %q", domain, m.cookieDomain)
		}
		domain = m.cookieDomain
	}

	httpCookie := &http.Cookie{
		Name:     m.cookieName,
		Value:    cookieData,
		Path:     "/",
		Domain:   domain,
		HttpOnly: m.cookieHttpOnly,
		Secure:   m.cookieSecure,
		Expires:  time.Now().Add(m.cookieExpire),
	}
	http.SetCookie(rw, httpCookie)

	return nil
}

func (m *UserInfoHandler) FetchUserInfo(username string) (UserInfo, error) {
	res, err := m.api.FetchUserInfo(username)
	if err != nil {
		return nil, err
	}

	userInfo := make(map[string]string, 3)
	userInfo["Roles"] = strings.Join(res.Roles, ",")
	userInfo["User"] = res.Username
	userInfo["Email"] = res.Email

	return userInfo, nil
}

func (m *UserInfoHandler) WriteForwardedHeaders(req *http.Request, data UserInfo) {
	for key, value := range data {
		headerKey := fmt.Sprintf("X-Forwarded-%s", key)
		req.Header.Set(headerKey, value)
	}
}
