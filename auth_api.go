package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type AuthApi interface {
	FetchUserInfo(string) (*AuthApiResponse, bool, error)
	Validate(string, password string) bool
}

type DefaultAuthApi struct {
	url *url.URL
}

type AuthApiRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthApiResponse struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

func NewAuthApiFromUrl(rawurl string) (*DefaultAuthApi, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	return &DefaultAuthApi{
		url: url,
	}, nil
}

// FetchUserInfo returns the response data and whether the user exists.
func (a *DefaultAuthApi) FetchUserInfo(username string) (*AuthApiResponse, bool, error) {
	url := fmt.Sprintf("%s/refresh?username=%s", a.url.String(), username)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("error instantiating auth api request: ", url)
		return nil, false, err
	}

	request.Header.Set("Content-Type", "application/json")

	log.Printf("making request to auth_api: %s", url)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("error making request to auth api: ", err)
		return nil, false, err
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading response body: ", err)
		return nil, false, err
	}
	defer resp.Body.Close()

	// The user doesn't exist anymore
	if resp.StatusCode == 404 {
		log.Printf("user no longer exists: %s", username)
		return nil, false, nil
	}

	if resp.StatusCode == 403 {
		log.Printf("user no longer active: %s", username)
		return nil, false, nil
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("got %d from %q %s", resp.StatusCode, url, body)
		log.Printf("unexpected status code: %s", err)
		return nil, false, err
	}

	var authApiResponse AuthApiResponse
	err = json.Unmarshal(body, &authApiResponse)
	if err != nil {
		log.Printf("unable to convert auth api response into correct response")
		return nil, false, err
	}

	return &authApiResponse, true, nil
}

func (a *DefaultAuthApi) Validate(username string, password string) bool {
	requestData := AuthApiRequest{
		Username: username,
		Password: password,
	}

	url := fmt.Sprintf("%s/auth", a.url.String())
	requestJson, _ := json.Marshal(requestData)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(requestJson))
	if err != nil {
		log.Printf("error instantiating auth api request: ", err)
		return false
	}
	request.Header.Set("Content-Type", "application/json")

	log.Printf("making request to auth_api: %s for user: %q", a.url.String(), username)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("error making request to auth api: ", err)
		return false
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading response body: ", err)
		return false
	}
	defer resp.Body.Close()

	// unwrap auth-api response and ensure that we received a 200 in response
	if resp.StatusCode != 200 {
		log.Printf("got %d from %q %s", resp.StatusCode, a.url, body)
		return false
	}

	return true
}
