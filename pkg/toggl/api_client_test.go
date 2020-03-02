package toggl

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApiClient_Ping(t *testing.T) {
	t.Run("Api Ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {}))
		client := NewApiClient(srv.URL)
		err := client.Ping()

		assert.NoError(t, err)
	})

	t.Run("Api Not Healthy", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(500) }))
		client := NewApiClient(srv.URL)
		err := client.Ping()

		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrApiNotHealthy))
	})

	t.Run("Api Bad Url", func(t *testing.T) {
		client := NewApiClient("UnknownUrl")
		err := client.Ping()

		assert.Error(t, err)
	})
}

func TestApiClient_stringify(t *testing.T) {
	str := stringify([]int{})
	assert.Equal(t, "", str)

	str2 := stringify([]int{1, 2, 3, 4, 5, 6})
	assert.Equal(t, "1,2,3,4,5,6", str2)
}

func TestApiClient_WithAuthToken(t *testing.T) {
	client := NewApiClient("http://localhost")
	client.WithAuthToken("token")

	assert.Equal(t, "token", client.autoToken)
}

func TestApiClient_GetTimeEntries(t *testing.T) {
	t.Run("GetTimeEntries Ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "toggl-pipes", req.Header.Get("User-Agent"))
			assert.Equal(t, http.MethodGet, req.Method)

			u, p, ok := req.BasicAuth()
			assert.Equal(t, "test", u)
			assert.Equal(t, "api_token", p)
			assert.True(t, ok)

			tes := []TimeEntry{
				{
					ID:                0,
					ProjectID:         0,
					TaskID:            0,
					UserID:            0,
					Billable:          false,
					Start:             "",
					Stop:              "",
					DurationInSeconds: 0,
					Description:       "",
					ForeignID:         "",
					ForeignTaskID:     "",
					ForeignUserID:     "",
					ForeignProjectID:  "",
				},
			}

			b, err := json.Marshal(tes)
			if err != nil {
				res.WriteHeader(500)
				return
			}

			_, err = res.Write(b)
			if err != nil {
				res.WriteHeader(500)
				return
			}
		}))

		client := NewApiClient(srv.URL)
		client.WithAuthToken("test")
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.NoError(t, err)
		assert.NotEmpty(t, te)
	})

	t.Run("GetTimeEntries Server Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(500) }))

		client := NewApiClient(srv.URL)
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GET time_entries failed")
		assert.Empty(t, te)
	})

	t.Run("GetTimeEntries Error Read Body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := NewApiClient(srv.URL)
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, te)
	})

	t.Run("GetTimeEntries Server Down", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))
		srv.Close()

		client := NewApiClient(srv.URL)
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.Empty(t, te)
	})

	t.Run("GetTimeEntries Bad Url", func(t *testing.T) {
		client := NewApiClient("http://bad\\wtf")
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
		assert.Empty(t, te)
	})
}

func TestApiClient_GetWorkspaceIdByToken(t *testing.T) {
	t.Run("GetWorkspaceIdByToken Ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "toggl-pipes", req.Header.Get("User-Agent"))
			assert.Equal(t, http.MethodGet, req.Method)

			u, p, ok := req.BasicAuth()
			assert.Equal(t, "test123", u)
			assert.Equal(t, "api_token", p)
			assert.True(t, ok)

			wr := WorkspaceResponse{
				Workspace: &Workspace{
					ID:   1,
					Name: "",
				}}

			b, err := json.Marshal(wr)
			if err != nil {
				res.WriteHeader(500)
				return
			}

			_, err = res.Write(b)
			if err != nil {
				res.WriteHeader(500)
				return
			}
		}))

		client := NewApiClient(srv.URL)
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.NoError(t, err)
		assert.Equal(t, 1, id)
	})

	t.Run("GetWorkspaceIdByToken Server Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(500) }))

		client := NewApiClient(srv.URL)
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GET workspace failed")
		assert.Empty(t, id)
	})

	t.Run("GetWorkspaceIdByToken Error Read Body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := NewApiClient(srv.URL)
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, id)
	})

	t.Run("GetWorkspaceIdByToken Server Down", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))
		srv.Close()

		client := NewApiClient(srv.URL)
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.Empty(t, id)
	})

	t.Run("GetWorkspaceIdByToken Bad Url", func(t *testing.T) {
		client := NewApiClient("http://bad\\wtf")
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
		assert.Empty(t, id)
	})
}

func TestApiClient_postPipesAPI(t *testing.T) {
	t.Run("postPipesAPI Ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "toggl-pipes", req.Header.Get("User-Agent"))
			assert.Equal(t, http.MethodPost, req.Method)

			u, p, ok := req.BasicAuth()
			assert.Equal(t, "test123", u)
			assert.Equal(t, "api_token", p)
			assert.True(t, ok)

			res.Write([]byte("test"))
		}))

		client := NewApiClient(srv.URL)
		client.WithAuthToken("test123")
		res, err := client.postPipesAPI("test", nil)

		assert.NoError(t, err)
		assert.Equal(t, []byte("test"), res)
	})

	t.Run("postPipesAPI Server Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(500) }))

		client := NewApiClient(srv.URL)
		res, err := client.postPipesAPI("test", nil)

		assert.Error(t, err)
		assert.Empty(t, res)
	})

	t.Run("postPipesAPI Error Read Body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := NewApiClient(srv.URL)
		res, err := client.postPipesAPI("test", nil)

		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("postPipesAPI Server Down", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))
		srv.Close()

		client := NewApiClient(srv.URL)
		res, err := client.postPipesAPI("test", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.Empty(t, res)
	})

	t.Run("postPipesAPI Bad Payload", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))
		srv.Close()

		client := NewApiClient(srv.URL)
		res, err := client.postPipesAPI("test", func() {})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type")
		assert.Empty(t, res)
	})

	t.Run("postPipesAPI Bad Url", func(t *testing.T) {
		client := NewApiClient("http://bad\\wtf")
		res, err := client.postPipesAPI("test", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
		assert.Empty(t, res)
	})

}
