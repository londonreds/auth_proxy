package main

import (
	"encoding/json"
	"github.com/bmizerany/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAuthApi(t *testing.T) {
	url := "http://test.buzzfeed.com"
	authApi, err := NewAuthApi(url, "")
	assert.Equal(t, err, nil)
	assert.Equal(t, authApi.url.Scheme, "http")
	assert.Equal(t, authApi.url.Host, "test.buzzfeed.com")

	authApi, err = NewAuthApi(url, "deadbeef")
	assert.Equal(t, err, nil)
	assert.Equal(t, authApi.authToken, "deadbeef")
}

func TestAuthApiValidate(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer backend.Close()
	authApi, err := NewAuthApi(backend.URL, "")
	assert.Equal(t, err, nil)

	verified := authApi.Validate("test", "test")
	assert.Equal(t, verified, true)
}

func TestAuthApiValidateFailure(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer backend.Close()
	authApi, err := NewAuthApi(backend.URL, "")
	assert.Equal(t, err, nil)

	verified := authApi.Validate("test", "test")
	assert.Equal(t, verified, false)
}

func TestAuthApiFetchUserInfo(t *testing.T) {
	stubResponse := &AuthApiResponse{
		Username: "test",
		Email:    "test@buzzfeed.com",
		Roles:    []string{"test"},
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Header.Get("X-Auth-Token"), "")

		data, err := json.Marshal(stubResponse)
		assert.Equal(t, err, nil)
		w.Write([]byte(data))
		w.WriteHeader(200)
	}))
	defer backend.Close()

	authApi, err := NewAuthApi(backend.URL, "")
	assert.Equal(t, err, nil)
	response, userExists, err := authApi.FetchUserInfo("test")
	assert.Equal(t, err, nil)
	assert.Equal(t, response, stubResponse)
	assert.Equal(t, userExists, true)
}

func TestAuthApiAuthTokenHeader(t *testing.T) {
	stubResponse := &AuthApiResponse{
		Username: "test",
		Email:    "test@buzzfeed.com",
		Roles:    []string{"test"},
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Header.Get("X-Auth-Token"), "deadbeef")

		data, err := json.Marshal(stubResponse)
		assert.Equal(t, err, nil)
		w.Write([]byte(data))
		w.WriteHeader(200)
	}))
	defer backend.Close()

	authApi, err := NewAuthApi(backend.URL, "deadbeef")
	assert.Equal(t, err, nil)

	// Test the refresh endpoint.
	_, _, err = authApi.FetchUserInfo("test")
	assert.Equal(t, err, nil)

	// Test the auth endpoint.
	verified := authApi.Validate("test", "test")
	assert.Equal(t, verified, true)
}

func TestAuthApiFetchUserInfoFailure(t *testing.T) {
	testCases := []struct {
		status     int
		userExists bool
		err        bool
	}{
		{400, false, true},
		{404, false, false},
		{403, false, false},
	}
	for _, testCase := range testCases {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(testCase.status)
		}))

		authApi, err := NewAuthApi(backend.URL, "")
		assert.Equal(t, err, nil)
		response, userExists, err := authApi.FetchUserInfo("test")
		assert.Equal(t, err != nil, testCase.err)
		assert.Equal(t, response == nil, true)
		assert.Equal(t, userExists, testCase.userExists)

		defer backend.Close()
	}
}
