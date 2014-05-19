About
=====

[bugsnag](http://bugsnag.com) client in Go

Please see https://bugsnag.com/docs/notifier-api for more info about the bugsnag API.

How to use
==========

```go
package main

import (
        "errors"
        "net/http"
        "github.com/toggl/bugsnag"
)

func main() {
        // First, configure bugsnag. Only APIKey is mandatory, other settings are optional.
        bugsnag.APIKey = "c9d60ae4c7e70c4b6c4ebd3e8056d2b8"

        bugsnag.AppVersion = "1.0.2"
        bugsnag.OSVersion = "Windows XP"
        bugsnag.ReleaseStage = "production"
        bugsnag.NotifyReleaseStages = []string{"production"}
        bugsnag.UseSSL = true
        bugsnag.Verbose = true

        // Then, out of the blue, an error happens:
        err := errors.New("Something bad just happened")
        bugsnag.Notify(err)

        // In case you need to supply the user ID:
        bugsnag.New(err).WithUserID("12345").Notify()

        // To report what your app was doing while error happened:
        bugsnag.New(err).WithContext("something").Notify()

        // Metadata can be set all at once:
        values := map[string]interface{}{
                "user_agent": "ie4",
                "account_id": 5555,
        }
        bugsnag.New(err).WithMetaDataValues("tab_name_in_bugsnag", values).Notify()

        // Or one key-value pair at a time:
        bugsnag.New(err).WithMetaData("tab_name_in_bugsnag", "user_agent", "ie4").WithMetaData("tab_name_in_bugsnag", "account_id", 5555).Notify()

        r, _ := http.NewRequest("GET", "http://google.com", nil)

        // To notify about a HTTP handler error:
        bugsnag.NotifyRequest(err, r)

        // To capture a HTTP handler panic, add this to your handler (assuming r is a *http.Request):
        defer bugsnag.CapturePanic(r)

        // If you need to add some error details when a panic is captured:
        defer bugsnag.OnCapturePanic(r, func(event bugsnag.EventDescriber) {
                event.WithMetaData("tab_name_in_bugsnag", "foo", "bar")
        })
}
```
