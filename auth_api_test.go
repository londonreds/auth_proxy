package main

import (
	"encoding/json"
	"github.com/bmizerany/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAuthApiFromUrl(t *testing.T) {
	url := "http://test.buzzfeed.com"
	authApi, err := NewAuthApiFromUrl(url)
	assert.Equal(t, err, nil)
	assert.Equal(t, authApi.url.Scheme, "http")
	assert.Equal(t, authApi.url.Host, "test.buzzfeed.com")
}

func TestAuthApiValidate(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer backend.Close()
	authApi, err := NewAuthApiFromUrl(backend.URL)
	assert.Equal(t, err, nil)

	verified := authApi.Validate("test", "test")
	assert.Equal(t, verified, true)
}

func TestAuthApiValidateFailure(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer backend.Close()
	authApi, err := NewAuthApiFromUrl(backend.URL)
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
		data, err := json.Marshal(stubResponse)
		assert.Equal(t, err, nil)
		w.Write([]byte(data))
		w.WriteHeader(200)
	}))
	defer backend.Close()

	authApi, err := NewAuthApiFromUrl(backend.URL)
	assert.Equal(t, err, nil)
	response, _, err := authApi.FetchUserInfo("test")
	assert.Equal(t, err, nil)
	assert.Equal(t, response, stubResponse)
}

func TestAuthApiFetchUserInfoFailure(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer backend.Close()

	authApi, err := NewAuthApiFromUrl(backend.URL)
	assert.Equal(t, err, nil)
	response, _, err := authApi.FetchUserInfo("test")
	assert.Equal(t, response == nil, true)
	assert.Equal(t, err == nil, false)
}
