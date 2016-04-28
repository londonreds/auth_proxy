package main

import (
	"encoding/json"
	"errors"
	"github.com/bmizerany/assert"
	"github.com/buzzfeed/auth_proxy/providers"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestUserInfoLoadCookiedData(t *testing.T) {
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "127.0.0.1", nil)
	handler := &UserInfoHandler{
		api:            nil,
		refresh:        time.Minute,
		cookieExpire:   time.Minute * 30,
		cookieDomain:   "127.0.0.1",
		cookieName:     "test",
		cookieSeed:     "testtesttesttest",
		cookieHttpOnly: true,
		cookieSecure:   false,
	}
	userInfo := &UserInfo{
		User:    "test",
		Email:   "test@test.com",
		Roles:   []string{"admin", "advertiser"},
		Created: time.Now(),
	}
	err := handler.SetCookie(rw, req, userInfo)
	assert.Equal(t, err, nil)
	assert.Equal(t, rw.Header()["Set-Cookie"][0] != "", true)
	req.Header.Set("Cookie", rw.Header()["Set-Cookie"][0])

	loadedUserInfo, err := handler.LoadCookiedData(req)
	assert.Equal(t, err, nil)
	assert.Equal(t, loadedUserInfo.User, userInfo.User)
	assert.Equal(t, loadedUserInfo.Email, userInfo.Email)
	assert.Equal(t, loadedUserInfo.Roles, userInfo.Roles)
	assert.Equal(t, loadedUserInfo.Created.Truncate(time.Second).String(), userInfo.Created.Truncate(time.Second).String())
}

func TestUserInfoHeaders(t *testing.T) {
	userInfo := &UserInfo{
		User:    "test",
		Email:   "test@test.com",
		Roles:   []string{"admin", "advertiser"},
		Created: time.Now(),
	}
	handler := &UserInfoHandler{
		api:            nil,
		cookieExpire:   time.Minute,
		cookieDomain:   "127.0.0.1",
		cookieName:     "test",
		cookieSeed:     "testtesttesttest",
		cookieHttpOnly: true,
		cookieSecure:   false,
	}

	req, _ := http.NewRequest("GET", "/", nil)
	handler.WriteForwardedHeaders(req, userInfo)

	assert.Equal(t, req.Header["X-Forwarded-User"][0], "test")
}

func TestIsUserInfoStale(t *testing.T) {
	handler := &UserInfoHandler{
		refresh:        time.Second * 30,
		cookieExpire:   time.Minute,
		cookieDomain:   "127.0.0.1",
		cookieName:     "test",
		cookieSeed:     "testtesttesttest",
		cookieHttpOnly: true,
		cookieSecure:   false,
	}

	info := &UserInfo{
		User:  "test",
		Roles: []string{"admin", "advertiser"},
		Email: "test@test.com",
	}

	info.Created = time.Now()
	assert.Equal(t, handler.IsUserInfoStale(info), false)

	info.Created = time.Now().Add(-1*handler.refresh - time.Second)
	assert.Equal(t, handler.IsUserInfoStale(info), true)
}

type StubAuthApi struct {
	url        *url.URL
	status     string
	userExists bool
}

func (a *StubAuthApi) FetchUserInfo(username string) (*AuthApiResponse, bool, error) {
	if a.status == "down" {
		return nil, false, errors.New("error making request to auth api")
	}

	if !a.userExists {
		return nil, false, nil
	}

	response := &AuthApiResponse{
		Username: username,
		Email:    "test@test.com",
		Roles:    []string{"admin", "advertiser"},
	}
	return response, true, nil
}

func (a *StubAuthApi) Validate(username string, password string) bool {
	return true
}

func NewTestRequest(t *testing.T, handler *UserInfoHandler, info *UserInfo) *http.Request {
	req, _ := http.NewRequest("GET", "127.0.0.1", nil)
	rw := httptest.NewRecorder()

	if info != nil {
		err := handler.SetCookie(rw, req, info)
		assert.Equal(t, err, nil)
		req.Header.Set("Cookie", rw.Header()["Set-Cookie"][0])
	}
	return req
}

func TestGetUserInfo(t *testing.T) {
	session := &providers.SessionState{User: "test"}
	url, _ := url.Parse("http://127.0.0.1")
	handler := &UserInfoHandler{
		refresh:        time.Second * 30,
		cookieExpire:   time.Minute,
		cookieDomain:   "127.0.0.1",
		cookieName:     "test",
		cookieSeed:     "testtesttesttest",
		cookieHttpOnly: true,
		cookieSecure:   false,
	}

	stubInfo := &UserInfo{
		User:    "test",
		Roles:   []string{"admin", "advertiser"},
		Email:   "test@test.com",
		Created: time.Now(),
	}

	testCases := []struct {
		input  map[string]string
		output map[string]bool
	}{
		{
			map[string]string{"cache": "none", "api": "down"},
			map[string]bool{"error": true, "updated": false, "info": false},
		},
		{
			map[string]string{"cache": "none", "api": "up"},
			map[string]bool{"error": false, "updated": true, "info": true},
		},
		{
			map[string]string{"cache": "none", "api": "dne"},
			map[string]bool{"error": true, "updated": false, "info": false},
		},
		{
			map[string]string{"cache": "valid", "api": "down"},
			map[string]bool{"error": false, "updated": false, "info": true},
		},
		{
			map[string]string{"cache": "valid", "api": "up"},
			map[string]bool{"error": false, "updated": false, "info": true},
		},
		{
			map[string]string{"cache": "valid", "api": "dne"},
			map[string]bool{"error": false, "updated": false, "info": true},
		},
		{
			// If the api is down, continue to use the stale cache.
			map[string]string{"cache": "stale", "api": "down"},
			map[string]bool{"error": false, "updated": false, "info": true},
		},
		{
			map[string]string{"cache": "stale", "api": "up"},
			map[string]bool{"error": false, "updated": true, "info": true},
		},
		{
			map[string]string{"cache": "stale", "api": "dne"},
			map[string]bool{"error": true, "updated": false, "info": false},
		},
	}
	for _, testCase := range testCases {
		var apiStatus string
		var userExists bool
		if testCase.input["api"] == "down" {
			apiStatus = "down"
			userExists = true
		} else if testCase.input["api"] == "dne" {
			apiStatus = "up"
			userExists = false

		} else {
			apiStatus = "up"
			userExists = true
		}

		var cachedInfo *UserInfo
		if testCase.input["cache"] == "none" {
			cachedInfo = nil
		} else if testCase.input["cache"] == "stale" {
			cachedInfo = stubInfo
			cachedInfo.Created = time.Now().Add(-1*handler.refresh - time.Second)
		} else {
			cachedInfo = stubInfo
			cachedInfo.Created = time.Now()
		}

		handler.api = &StubAuthApi{
			url:        url,
			status:     apiStatus,
			userExists: userExists,
		}
		req := NewTestRequest(t, handler, cachedInfo)
		info, updated, err := handler.GetUserInfo(req, session)

		log.Printf("input: %+v expected output: %+v", testCase.input, testCase.output)

		assert.Equal(t, info != nil, testCase.output["info"])
		assert.Equal(t, updated, testCase.output["updated"])
		assert.Equal(t, err != nil, testCase.output["error"])
	}
}

func TestUserInfoFetch(t *testing.T) {
	userInfoResponse := &AuthApiResponse{
		Username: "test",
		Email:    "test@buzzfeed.com",
		Roles:    []string{"test1", "test2"},
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := json.Marshal(userInfoResponse)
		assert.Equal(t, err, nil)
		w.Write([]byte(data))
		w.WriteHeader(200)
	}))
	defer backend.Close()

	authApi, err := NewAuthApi(backend.URL, "")
	handler := &UserInfoHandler{
		api:            authApi,
		cookieExpire:   time.Minute,
		cookieDomain:   "127.0.0.1",
		cookieName:     "test",
		cookieSeed:     "testtesttesttest",
		cookieHttpOnly: true,
		cookieSecure:   false,
	}

	userInfo, _, err := handler.FetchUserInfo("test")
	assert.Equal(t, err, nil)
	assert.Equal(t, userInfo.User, "test")
	assert.Equal(t, userInfo.Email, "test@buzzfeed.com")
	assert.Equal(t, userInfo.Roles, []string{"test1", "test2"})
}
