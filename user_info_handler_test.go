package main

import (
	"encoding/json"
	"github.com/bmizerany/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUserInfoLoadCookiedData(t *testing.T) {
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "127.0.0.1", nil)
	handler := &UserInfoHandler{
		api:            nil,
		cookieExpire:   time.Minute,
		cookieDomain:   "127.0.0.1",
		cookieName:     "test",
		cookieSeed:     "testtesttesttest",
		cookieHttpOnly: true,
		cookieSecure:   false,
	}
	userInfo := map[string]string{
		"test": "test",
	}
	err := handler.WriteCookie(rw, req, userInfo)
	assert.Equal(t, err, nil)
	assert.Equal(t, rw.Header()["Set-Cookie"][0] != "", true)
	req.Header.Set("Cookie", rw.Header()["Set-Cookie"][0])

	loadedUserInfo, err := handler.LoadCookiedData(req)
	assert.Equal(t, err, nil)
	assert.Equal(t, loadedUserInfo["test"], userInfo["test"])
}

func TestUserInfoHeaders(t *testing.T) {
	userInfo := map[string]string{
		"test": "test",
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

	assert.Equal(t, req.Header["X-Forwarded-Test"][0], "test")
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

	authApi, err := NewAuthApiFromUrl(backend.URL)
	handler := &UserInfoHandler{
		api:            authApi,
		cookieExpire:   time.Minute,
		cookieDomain:   "127.0.0.1",
		cookieName:     "test",
		cookieSeed:     "testtesttesttest",
		cookieHttpOnly: true,
		cookieSecure:   false,
	}

	userInfo, err := handler.FetchUserInfo("test")
	assert.Equal(t, err, nil)
	assert.Equal(t, userInfo["Roles"], "test1,test2")
	assert.Equal(t, userInfo["Username"], "test")
	assert.Equal(t, userInfo["Email"], "test@buzzfeed.com")
}
