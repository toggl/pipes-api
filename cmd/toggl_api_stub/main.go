// This package uses only for testing/reverse engineering purpose.
// It imitates `toggl_api` so you can run pipes-api without running real `toggl_api`.
// If you don't know what is this all about, just ignore this package.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type stubHandler struct{}

func (h *stubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	u, err := url.ParseRequestURI(r.RequestURI)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch strings.TrimSuffix(u.Path, "/") {
	case "/app/pipes-oauth":
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		code := r.Form.Get("code")
		ecode, err := url.QueryUnescape(code)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf(`Authorization code: %s`, ecode)))

	case "/api/pipes/workspace":

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"id":1,"name":"test"}}`))

	case "/api/pipes/users":

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"users":[{"id":2,"email":"support@toggl.com","name":"Toggl • Support", "foreign_id":"96440724390141"},{"id":1,"email":"anton.kucherov@toggl.com","name":"Anton Kucherov", "foreign_id":"1163511801267893"}],"notifications":["test","test2"]}`))

	case "/api/pipes/projects":

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"projects":[{"id":1,"name":"test project","active":true,"foreign_id":"1167139632192241"},{"id":2,"name":"test 2 project","active":true,"foreign_id":"1167621479402017"}],"notifications":["p1","p2"]}`))

	case "/api/pipes/tasks":

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tasks":[{"id":100,"name":"One task","active":true,"pid":1,"foreign_id":"1167621479402021"},{"id":200,"name":"Another task","active":true,"pid":1,"foreign_id":"1167621479402023"},{"id":300,"name":"","active":true,"pid":1,"foreign_id":"1167621479402025"},{"id":400,"name":"Thidr task","active":true,"pid":2,"foreign_id":"1167621479402027"},{"id":500,"name":"Fourth task","active":true,"pid":2,"foreign_id":"1167621479402029"},{"id":600,"name":"Six task","active":true,"pid":2,"foreign_id":"1167621479402031"},{"id":700,"name":"","active":true,"pid":2,"foreign_id":"1167621479402033"}],"notifications":["t1","t2"]}`))

	}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("%s %s\n", r.Method, r.RequestURI)
	log.Printf("Request Headers: %s\n", r.Header)
	log.Printf("Request Body: %s\n\n", body)
}

var port string
var server bool

func main() {
	flag.BoolVar(&server, "server", false, "")
	flag.StringVar(&port, "address", ":8888", "Listen address")
	flag.Parse()

	if server {
		log.Fatal(http.ListenAndServe(port, &stubHandler{}))
	}
}
