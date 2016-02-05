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

type AuthApi struct {
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

func NewAuthApiFromUrl(rawurl string) (*AuthApi, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	return &AuthApi{
		url: url,
	}, nil
}

func (a *AuthApi) FetchUserInfo(username string) (*AuthApiResponse, error) {
	url := fmt.Sprintf("%s/refresh?username=%s", a.url.String(), username)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("error instantiating auth api request: ", url)
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	log.Printf("making request to auth_api: %s for user: %q", a.url.String(), username)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("error making request to auth api: ", err)
		return nil, err
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading response body: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	var authApiResponse AuthApiResponse
	err = json.Unmarshal(body, &authApiResponse)
	if err != nil {
		log.Printf("unable to convert auth api response into correct response")
		return nil, err
	}

	return &authApiResponse, nil
}

func (a *AuthApi) Validate(username string, password string) bool {
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
