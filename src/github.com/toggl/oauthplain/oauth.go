package oauthplain

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type (
	Config struct {
		ConsumerKey       string
		ConsumerSecret    string
		RequestTokenUrl   string
		AuthorizeTokenUrl string
		AccessTokenUrl    string
	}
	Token struct {
		ConsumerKey      string
		ConsumerSecret   string
		OAuthToken       string
		OAuthTokenSecret string
		OAuthVerifier    string
		AuthorizeUrl     string
		Extra            map[string]string
	}
	Transport struct {
		*Config
	}
)

func (c *Config) UpdateURLs(s ...interface{}) *Config {
	config := *c
	config.AccessTokenUrl = fmt.Sprintf(c.AccessTokenUrl, s...)
	config.RequestTokenUrl = fmt.Sprintf(c.RequestTokenUrl, s...)
	config.AuthorizeTokenUrl = fmt.Sprintf(c.AuthorizeTokenUrl, s...)
	return &config
}

func (t *Token) AuthHeader() string {
	params := map[string]string{
		"oauth_version":          "1.0",
		"oauth_signature_method": "PLAINTEXT",
		"oauth_consumer_key":     t.ConsumerKey,
		"oauth_timestamp":        strconv.FormatInt(time.Now().Unix(), 10),
		"oauth_nonce":            strconv.FormatInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63(), 10),
		"oauth_signature":        fmt.Sprintf("%s&%s", t.ConsumerSecret, t.OAuthTokenSecret),
	}
	if t.OAuthToken != "" {
		params["oauth_token"] = t.OAuthToken
	}
	if t.OAuthVerifier != "" {
		params["oauth_verifier"] = t.OAuthVerifier
	}
	for key, value := range t.Extra {
		params[key] = value
	}
	authString := "OAuth "
	for key, value := range params {
		authString = authString + fmt.Sprintf(`%s="%s", `, key, value)
	}
	return strings.TrimRight(authString, ", ")
}

func (t *Transport) NewToken(extra map[string]string) *Token {
	token := &Token{
		Extra:          extra,
		ConsumerKey:    t.ConsumerKey,
		ConsumerSecret: t.ConsumerSecret,
	}
	return token
}

func (t *Transport) Client() *http.Client {
	return &http.Client{}
}

func (t *Transport) AuthCodeURL(callbackURL string) (*Token, error) {
	token := t.NewToken(map[string]string{
		"oauth_callback": callbackURL,
	})
	values, err := t.request(t.RequestTokenUrl, token)
	if err != nil {
		return nil, err
	}
	url_, err := url.Parse(t.AuthorizeTokenUrl)
	if err != nil {
		return nil, err
	}

	url_.RawQuery = url.Values{
		"oauth_callback": {callbackURL},
		"oauth_token":    {values.Get("oauth_token")},
	}.Encode()

	token.AuthorizeUrl = url_.String()
	token.OAuthToken = values.Get("oauth_token")
	token.OAuthTokenSecret = values.Get("oauth_token_secret")
	return token, nil
}

func (t *Transport) Exchange(token *Token) error {
	values, err := t.request(t.AccessTokenUrl, token)
	if err != nil {
		return err
	}
	token.OAuthToken = values.Get("oauth_token")
	token.OAuthTokenSecret = values.Get("oauth_token_secret")
	return nil
}

func (t *Transport) request(reqUrl string, token *Token) (url.Values, error) {
	req, err := http.NewRequest("POST", reqUrl, nil)
	if err != nil {
		return nil, err
	}
	if token.ConsumerKey == "" || token.ConsumerSecret == "" {
		token.ConsumerKey = t.ConsumerKey
		token.ConsumerSecret = t.ConsumerSecret
	}
	req.Header.Set("Authorization", token.AuthHeader())
	resp, err := t.Client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%s: %s", resp.Status, string(b)))
	}
	v, err := url.ParseQuery(string(b))
	if err != nil {
		return nil, err
	}
	return v, nil
}
