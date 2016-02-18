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
	api            AuthApi
	refresh        time.Duration
	cookieExpire   time.Duration
	cookieDomain   string
	cookieName     string
	cookieSeed     string
	cookieHttpOnly bool
	cookieSecure   bool
}

type UserInfo struct {
	User    string
	Email   string
	Roles   []string
	Created time.Time
}

func (m *UserInfoHandler) Handle(rw http.ResponseWriter, req *http.Request, ses *providers.SessionState) error {
	// by default, check to see if user information exists in a cookie
	userInfo, updated, err := m.GetUserInfo(req, ses)
	if err != nil {
		m.ClearCookie(rw, req)
		return err
	}

	if updated {
		err := m.SetCookie(rw, req, userInfo)
		if err != nil {
			return err
		}
	}

	m.WriteForwardedHeaders(req, userInfo)

	return nil
}

func (m *UserInfoHandler) IsUserInfoStale(info *UserInfo) bool {
	cookieAge := time.Now().Sub(info.Created)
	return cookieAge > m.refresh
}

func (m *UserInfoHandler) GetUserInfo(req *http.Request, ses *providers.SessionState) (*UserInfo, bool, error) {
	fetch := false

	cachedInfo, err := m.LoadCookiedData(req)
	if err != nil {
		fetch = true
	} else {
		fetch = m.IsUserInfoStale(cachedInfo)
	}

	if fetch {
		updatedInfo, userExists, err := m.FetchUserInfo(ses.User)
		if err != nil {
			if cachedInfo != nil {
				return cachedInfo, false, nil
			} else {
				return nil, false, err
			}
		}
		if !userExists {
			// If the user no longer exists, clear this session.
			return nil, false, fmt.Errorf("user no longer exists: %s", ses.User)
		}
		if updatedInfo != nil {
			return updatedInfo, true, nil
		}
	}

	return cachedInfo, false, nil
}

func (m *UserInfoHandler) FetchUserInfo(username string) (*UserInfo, bool, error) {
	res, userExists, err := m.api.FetchUserInfo(username)
	if err != nil || !userExists {
		return nil, userExists, err
	}

	userInfo := &UserInfo{
		User:    res.Username,
		Email:   res.Email,
		Roles:   res.Roles,
		Created: time.Now(),
	}
	return userInfo, userExists, nil
}

func (m *UserInfoHandler) MakeCookie(req *http.Request, value string, expiration time.Duration, now time.Time) *http.Cookie {
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

	if value != "" {
		value = cookie.SignedValue(m.cookieSeed, m.cookieName, value, now)
	}
	return &http.Cookie{
		Name:     m.cookieName,
		Value:    value,
		Path:     "/",
		Domain:   domain,
		HttpOnly: m.cookieHttpOnly,
		Secure:   m.cookieSecure,
		Expires:  now.Add(expiration),
	}
}

func (m *UserInfoHandler) SetCookie(rw http.ResponseWriter, req *http.Request, userInfo *UserInfo) error {
	data := map[string]string{
		"User":  userInfo.User,
		"Email": userInfo.Email,
		"Roles": strings.Join(userInfo.Roles, ","),
	}
	jsonStr, err := json.Marshal(data)
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

	httpCookie := m.MakeCookie(req, encryptedData, m.cookieExpire, userInfo.Created)

	http.SetCookie(rw, httpCookie)

	return nil
}

func (m *UserInfoHandler) ClearCookie(rw http.ResponseWriter, req *http.Request) {
	http.SetCookie(rw, m.MakeCookie(req, "", time.Hour*-1, time.Now()))
}

func (m *UserInfoHandler) LoadCookiedData(req *http.Request) (*UserInfo, error) {
	c, err := req.Cookie(m.cookieName)
	if err != nil {
		return nil, errors.New("No cookie found")
	}

	value, created, ok := cookie.Validate(c, m.cookieSeed, m.cookieExpire)
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

	data := make(map[string]string, 3)
	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return nil, err
	}

	userInfo := &UserInfo{
		User:    data["User"],
		Email:   data["Email"],
		Roles:   strings.Split(data["Roles"], ","),
		Created: created,
	}

	return userInfo, nil
}

func (m *UserInfoHandler) WriteForwardedHeaders(req *http.Request, userInfo *UserInfo) {
	data := map[string]string{
		"User":  userInfo.User,
		"Email": userInfo.Email,
		"Roles": strings.Join(userInfo.Roles, ","),
	}
	for key, value := range data {
		headerKey := fmt.Sprintf("X-Forwarded-%s", key)
		req.Header.Set(headerKey, value)
	}
}
