go-freshbooks
=====
This project implements a [Go](http://golang.org) client library for the [freshbooks API](http://developers.freshbooks.com/).
Supports token-based and OAuth authentication.

Example usage
---------------

```go
api := freshbooks.NewApi("<<AccountName>>", "<<AuthToken>>")

users, err := api.Users()
tasks, err := api.Tasks()
clients, err := api.Clients()
projects, err := api.Projects()
```

OAuth authentication
---------------
The FreshBooks API also supports OAuth to authorize applications. [oauthplain](https://github.com/tambet/oauthplain) package is used to generate 'Authorization' headers.

```go
token := &oauthplain.Token{
  ConsumerKey:      "<<ConsumerKey>>",
  ConsumerSecret:   "<<ConsumerSecret>>",
  OAuthToken:       "<<OAuthToken>>",
  OAuthTokenSecret: "<<OAuthTokenSecret>>",
}

api := freshbooks.NewApi("<<AccountName>>", token)
```
