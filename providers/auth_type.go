package providers

type AuthType int

const (
	AuthTypeNone = iota
	AuthTypeOAuth2
	AuthTypeBasic
	AuthTypeApi
)

func (authType AuthType) String() string {
	return map[AuthType]string{
		AuthTypeNone:   "none",
		AuthTypeOAuth2: "oauth2",
		AuthTypeBasic:  "basic_auth",
		AuthTypeApi:    "auth_api",
	}[authType]
}

func AuthTypeFromString(s string) AuthType {
	return map[string]AuthType{
		"none":       AuthTypeNone,
		"oauth2":     AuthTypeOAuth2,
		"basic_auth": AuthTypeBasic,
		"auth_api":   AuthTypeApi,
	}[s]
}
