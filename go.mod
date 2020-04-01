module github.com/toggl/pipes-api

replace code.google.com/p/goauth2/oauth => ./vendor/code.google.com/goauth2/oauth

// replaced packages
require code.google.com/p/goauth2/oauth v0.0.0

require (
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bugsnag/bugsnag-go v1.5.3
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/gorilla/context v1.1.1
	github.com/gorilla/mux v1.7.4
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lib/pq v1.3.0
	github.com/namsral/flag v1.7.4-pre
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/range-labs/go-asana v0.0.0-20200127233601-f09b5bdfed8d
	github.com/stretchr/objx v0.1.1 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/tambet/oauthplain v0.0.0-20140905172838-bbbd263fa701
	github.com/toggl/go-basecamp v0.1.0
	github.com/toggl/go-freshbooks v0.0.0-20140904111550-aacdf55e408d
	github.com/toggl/go-teamweek v0.3.0
)

go 1.13
