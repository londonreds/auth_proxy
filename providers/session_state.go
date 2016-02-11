package providers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/buzzfeed/auth_proxy/cookie"
)

type SessionState struct {
	AccessToken  string
	ExpiresOn    time.Time
	RefreshToken string
	Email        string
	User         string
	AuthType     AuthType
}

func (s *SessionState) IsExpired() bool {
	if !s.ExpiresOn.IsZero() && s.ExpiresOn.Before(time.Now()) {
		return true
	}
	return false
}

func (s *SessionState) String() string {
	o := fmt.Sprintf("Session{auth_method:%s user:%s", s.AuthType, s.userOrEmail())
	if s.AccessToken != "" {
		o += " token:true"
	}
	if !s.ExpiresOn.IsZero() {
		o += fmt.Sprintf(" expires:%s", s.ExpiresOn)
	}
	if s.RefreshToken != "" {
		o += " refresh_token:true"
	}
	return o + "}"
}

func (s *SessionState) EncodeSessionState(c *cookie.Cipher) (string, error) {
	if c == nil || s.AccessToken == "" {
		return fmt.Sprintf("%s|%s", s.AuthType, s.userOrEmail()), nil
	}
	return s.EncryptedString(c)
}

func (s *SessionState) userOrEmail() string {
	u := s.User
	if s.Email != "" {
		u = s.Email
	}
	return u
}

func (s *SessionState) EncryptedString(c *cookie.Cipher) (string, error) {
	var err error
	if c == nil {
		panic("error. missing cipher")
	}
	a := s.AccessToken
	if a != "" {
		a, err = c.Encrypt(a)
		if err != nil {
			return "", err
		}
	}
	r := s.RefreshToken
	if r != "" {
		r, err = c.Encrypt(r)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s|%s|%s|%d|%s", s.AuthType, s.userOrEmail(), a, s.ExpiresOn.Unix(), r), nil
}

func DecodeSessionState(v string, c *cookie.Cipher) (s *SessionState, err error) {
	chunks := strings.Split(v, "|")
	authType := AuthTypeFromString(chunks[0])
	if len(chunks) == 2 {
		userOrEmail := chunks[1]
		if strings.Contains(userOrEmail, "@") {
			u := strings.Split(userOrEmail, "@")[0]

			return &SessionState{AuthType: authType, Email: userOrEmail, User: u}, nil
		}
		return &SessionState{AuthType: authType, User: userOrEmail}, nil
	}

	if len(chunks) != 5 {
		err = fmt.Errorf("invalid number of fields (got %d expected 5)", len(chunks))
		return
	}

	s = &SessionState{AuthType: authType}
	if c != nil && chunks[2] != "" {
		s.AccessToken, err = c.Decrypt(chunks[2])
		if err != nil {
			return nil, err
		}
	}
	if c != nil && chunks[4] != "" {
		s.RefreshToken, err = c.Decrypt(chunks[4])
		if err != nil {
			return nil, err
		}
	}
	if u := chunks[1]; strings.Contains(u, "@") {
		s.Email = u
		s.User = strings.Split(u, "@")[0]
	} else {
		s.User = u
	}
	ts, _ := strconv.Atoi(chunks[3])
	s.ExpiresOn = time.Unix(int64(ts), 0)
	return
}
