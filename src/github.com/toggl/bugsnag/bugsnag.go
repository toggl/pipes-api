package bugsnag

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
)

var (
	APIKey              string
	AppVersion          string
	OSVersion           string
	Hostname            string
	WorkingDir          string
	ReleaseStage        = "production"
	NotifyReleaseStages = []string{ReleaseStage}
	AutoNotify          = true
	UseSSL              = true
	Verbose             = false
	Notifier            = &bugsnagNotifier{
		Name:    "Bugsnag Go client",
		Version: "0.0.2",
		URL:     "https://github.com/toggl/bugsnag_client",
	}
)

func init() {
	if hostname, err := os.Hostname(); err != nil {
		log.Println("Warning: could not detect hostname for Bugsnag client.")
		Hostname = "unknown"
	} else {
		Hostname = hostname
	}
	if wdir, err := os.Getwd(); err != nil {
		log.Println("Warning: could not get the working directory for Bugsnag client.")
		WorkingDir = "undetermined"
	} else {
		WorkingDir = wdir
	}
}

type (
	bugsnagNotifier struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	bugsnagPayload struct {
		APIKey   string           `json:"apiKey"`
		Notifier *bugsnagNotifier `json:"notifier"`
		Events   []*bugsnagEvent  `json:"events"`
	}
	bugsnagException struct {
		ErrorClass string              `json:"errorClass"`
		Message    string              `json:"message,omitempty"`
		Stacktrace []bugsnagStacktrace `json:"stacktrace,omitempty"`
	}
	bugsnagStacktrace struct {
		File       string `json:"file"`
		LineNumber string `json:"lineNumber"`
		Method     string `json:"method"`
		InProject  bool   `json:"inProject,omitempty"`
	}
	bugsnagEvent struct {
		UserID       string                            `json:"userId,omitempty"`
		AppVersion   string                            `json:"appVersion,omitempty"`
		OSVersion    string                            `json:"osVersion,omitempty"`
		ReleaseStage string                            `json:"releaseStage"`
		Context      string                            `json:"context,omitempty"`
		Exceptions   []bugsnagException                `json:"exceptions"`
		MetaData     map[string]map[string]interface{} `json:"metaData,omitempty"`
	}
	bugsnagError struct {
		error
		stacktrace []bugsnagStacktrace
	}
)

func Error(err error) error {
	if err == nil {
		return err
	}
	if serr, ok := err.(*bugsnagError); ok {
		return serr
	}
	return &bugsnagError{err, getStacktrace(err)}
}

func send(events []*bugsnagEvent) error {
	if APIKey == "" {
		return errors.New("Missing APIKey")
	}
	payload := &bugsnagPayload{
		Notifier: Notifier,
		APIKey:   APIKey,
		Events:   events,
	}
	protocol := "http"
	if UseSSL {
		protocol = "https"
	}
	if b, err := json.MarshalIndent(payload, "", "\t"); err != nil {
		return err
	} else if resp, err := http.Post(protocol+"://notify.bugsnag.com", "application/json", bytes.NewBuffer(b)); err != nil {
		return err
	} else if resp.StatusCode != 200 {
		return fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	} else if Verbose {
		println(string(b))
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		println(resp.StatusCode)
		println(resp.Status)
		println(string(b))
	}
	return nil
}

func getStacktrace(err error) []bugsnagStacktrace {
	var stacktrace []bugsnagStacktrace
	if serr, ok := err.(*bugsnagError); ok {
		return serr.stacktrace
	}
	i := 0
	for {
		if pc, file, line, ok := runtime.Caller(i); !ok {
			break
		} else {
			methodName := "unnamed"
			if f := runtime.FuncForPC(pc); f != nil {
				methodName = f.Name()
			}
			traceLine := bugsnagStacktrace{
				File:       file,
				LineNumber: strconv.Itoa(line),
				Method:     methodName,
			}
			stacktrace = append(stacktrace, traceLine)
		}
		i += 1
	}
	return stacktrace
}

// Notify sends an error to bugsnag
func Notify(err error) error {
	return New(err).Notify()
}

// NotifyRequest sends an error to bugsnag, and sets request
// URL as the event context.
func NotifyRequest(err error, r *http.Request) error {
	return New(err).WithRequest(r).Notify()
}

// CapturePanic reports panics happening while processing a HTTP request
func CapturePanic(r *http.Request) {
	OnCapturePanic(r, func(event EventDescriber) {})
}

// OnCapturePanic allows to set bugsnag variables before the
// error is sent off after a HTTP request handler panicked.
func OnCapturePanic(r *http.Request, handler func(event EventDescriber)) {
	if recovered := recover(); recovered != nil {
		var e error
		if err, ok := recovered.(error); ok {
			e = err
		} else if err, ok := recovered.(string); ok {
			e = errors.New(err)
		}
		if e != nil {
			New(e).WithRequest(r).withCallback(handler).Notify()
		}
		panic(recovered)
	}
}

// New returns returns a bugsnag event instance, that can be further configured
// before sending it.
func New(err error) *bugsnagEvent {
	return &bugsnagEvent{
		AppVersion:   AppVersion,
		OSVersion:    OSVersion,
		ReleaseStage: ReleaseStage,
		Exceptions: []bugsnagException{
			bugsnagException{
				ErrorClass: reflect.TypeOf(err).String(),
				Message:    err.Error(),
				Stacktrace: getStacktrace(err),
			},
		},
	}
}

// Notify sends the configured event off to bugsnag.
func (event *bugsnagEvent) Notify() error {
	event.WithMetaData("host", "name", Hostname)
	event.WithMetaData("host", "working_directory", WorkingDir)
	for _, stage := range NotifyReleaseStages {
		if stage == event.ReleaseStage {
			return send([]*bugsnagEvent{event})
		}
	}
	return nil
}

func (event *bugsnagEvent) WithRequest(r *http.Request) *bugsnagEvent {
	return event.WithContext(r.URL.String())
}

func (event *bugsnagEvent) withCallback(callback func(describer EventDescriber)) *bugsnagEvent {
	callback(event)
	return event
}

// EventDescriber is the public interface of a bugsnag event
type EventDescriber interface {
	WithUserID(userID string) *bugsnagEvent
	WithContext(context string) *bugsnagEvent
	WithMetaDataValues(tab string, values map[string]interface{}) *bugsnagEvent
	WithMetaData(tab string, name string, value interface{}) *bugsnagEvent
}

// WithUserID sets the user_id property on the bugsnag event.
func (event *bugsnagEvent) WithUserID(userID string) *bugsnagEvent {
	event.UserID = userID
	return event
}

func (event *bugsnagEvent) WithContext(context string) *bugsnagEvent {
	event.Context = context
	return event
}

// WithMetaDataValues sets bunch of key-value pairs under a tab in bugsnag
func (event *bugsnagEvent) WithMetaDataValues(tab string, values map[string]interface{}) *bugsnagEvent {
	if event.MetaData == nil {
		event.MetaData = make(map[string]map[string]interface{})
	}
	event.MetaData[tab] = values
	return event
}

// WithMetaData adds a key-value pair under a tab in bugsnag
func (event *bugsnagEvent) WithMetaData(tab string, name string, value interface{}) *bugsnagEvent {
	if event.MetaData == nil {
		event.MetaData = make(map[string]map[string]interface{})
	}
	if event.MetaData[tab] == nil {
		event.MetaData[tab] = make(map[string]interface{})
	}
	event.MetaData[tab][name] = value
	return event
}
