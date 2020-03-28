package domain

//AuthParams describes authorization parameters that come from `pipes-ui`.
//This structure is used for both oAuth1 and oAuth2 params.
type AuthParams struct {
	// Fields used for OAuth1
	AccountName string `json:"account_name,omitempty"`
	Token       string `json:"oauth_token,omitempty"`
	Verifier    string `json:"oauth_verifier,omitempty"`

	// Fields used for OAuth2
	Code string `json:"code,omitempty"`
}

type UserParams struct {
	IDs         []int `json:"ids"`
	SendInvites bool  `json:"send_invites"`
}
