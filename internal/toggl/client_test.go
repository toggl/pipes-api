package toggl

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/domain"
)

func TestApiClient_Ping(t *testing.T) {
	t.Run("Api Ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {}))
		client := &TogglApiClient{URL: srv.URL}
		err := client.Ping()

		assert.NoError(t, err)
	})

	t.Run("Api Not Healthy", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(500) }))
		client := &TogglApiClient{URL: srv.URL}
		err := client.Ping()

		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrApiNotHealthy))
	})

	t.Run("Api Bad Url", func(t *testing.T) {
		client := &TogglApiClient{URL: "UnknownUrl"}
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
	client := &TogglApiClient{URL: "http://localhost"}
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

			tes := []domain.TimeEntry{
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

		client := &TogglApiClient{URL: srv.URL}
		client.WithAuthToken("test")
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.NoError(t, err)
		assert.NotEmpty(t, te)
	})

	t.Run("GetTimeEntries Server Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(500) }))

		client := &TogglApiClient{URL: srv.URL}
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GET time_entries failed")
		assert.Empty(t, te)
	})

	t.Run("GetTimeEntries Error Read Body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := &TogglApiClient{URL: srv.URL}
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, te)
	})

	t.Run("GetTimeEntries Server Down", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))
		srv.Close()

		client := &TogglApiClient{URL: srv.URL}
		te, err := client.GetTimeEntries(time.Now(), []int{}, []int{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.Empty(t, te)
	})

	t.Run("GetTimeEntries Bad Url", func(t *testing.T) {
		client := &TogglApiClient{URL: "http://bad\\wtf"}
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

			wr := domain.WorkspaceResponse{
				Workspace: &domain.Workspace{
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

		client := &TogglApiClient{URL: srv.URL}
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.NoError(t, err)
		assert.Equal(t, 1, id)
	})

	t.Run("GetWorkspaceIdByToken Server Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(500) }))

		client := &TogglApiClient{URL: srv.URL}
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GET workspace failed")
		assert.Empty(t, id)
	})

	t.Run("GetWorkspaceIdByToken Error Read Body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := &TogglApiClient{URL: srv.URL}
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, id)
	})

	t.Run("GetWorkspaceIdByToken Server Down", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))
		srv.Close()

		client := &TogglApiClient{URL: srv.URL}
		id, err := client.GetWorkspaceIdByToken("test123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.Empty(t, id)
	})

	t.Run("GetWorkspaceIdByToken Bad Url", func(t *testing.T) {
		client := &TogglApiClient{URL: "http://bad\\wtf"}
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

		client := &TogglApiClient{URL: srv.URL}
		client.WithAuthToken("test123")
		res, err := client.postPipesAPI("test", nil)

		assert.NoError(t, err)
		assert.Equal(t, []byte("test"), res)
	})

	t.Run("postPipesAPI Server Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(500) }))

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.postPipesAPI("test", nil)

		assert.Error(t, err)
		assert.Empty(t, res)
	})

	t.Run("postPipesAPI Error Read Body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.postPipesAPI("test", nil)

		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("postPipesAPI Server Down", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))
		srv.Close()

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.postPipesAPI("test", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.Empty(t, res)
	})

	t.Run("postPipesAPI Bad Payload", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))
		srv.Close()

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.postPipesAPI("test", func() {})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type")
		assert.Empty(t, res)
	})

	t.Run("postPipesAPI Bad Url", func(t *testing.T) {
		client := &TogglApiClient{URL: "http://bad\\wtf"}
		res, err := client.postPipesAPI("test", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
		assert.Empty(t, res)
	})

}

func TestApiClient_PostClients(t *testing.T) {
	t.Run("PostClients Ok", func(t *testing.T) {
		in := &domain.ClientsImport{
			Clients: []*domain.Client{
				{
					ID:        1,
					Name:      "test",
					ForeignID: "test",
				},
			},
			Notifications: []string{""},
		}

		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			b, err := json.Marshal(in)
			assert.NoError(t, err)
			res.Write(b)
		}))

		client := &TogglApiClient{URL: srv.URL}
		out, err := client.PostClients("clients", nil)
		assert.NoError(t, err)
		assert.Equal(t, in, out)
	})

	t.Run("PostClients Response Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.PostClients("clients", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, res)
	})
}

func TestApiClient_PostProjects(t *testing.T) {
	t.Run("PostProjects Ok", func(t *testing.T) {
		in := &domain.ProjectsImport{
			Projects: []*domain.Project{
				{
					ID:        1,
					Name:      "test1",
					Active:    true,
					Billable:  true,
					ClientID:  1,
					ForeignID: "test2",
				},
			},
			Notifications: []string{""},
		}

		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			b, err := json.Marshal(in)
			assert.NoError(t, err)
			res.Write(b)
		}))

		client := &TogglApiClient{URL: srv.URL}
		out, err := client.PostProjects("projects", nil)
		assert.NoError(t, err)
		assert.EqualValues(t, *in, *out)
	})

	t.Run("PostProjects Response Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.PostProjects("projects", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, res)
	})
}

func TestApiClient_PostTasks(t *testing.T) {
	t.Run("PostTasks Ok", func(t *testing.T) {
		in := &domain.TasksImport{
			Tasks: []*domain.Task{
				{
					ID:        1,
					Name:      "test1",
					Active:    false,
					ProjectID: 1,
					ForeignID: "test2",
				},
			},
			Notifications: []string{},
		}

		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			b, err := json.Marshal(in)
			assert.NoError(t, err)
			res.Write(b)
		}))

		client := &TogglApiClient{URL: srv.URL}
		out, err := client.PostTasks("tasks", nil)
		assert.NoError(t, err)
		assert.EqualValues(t, *in, *out)
	})

	t.Run("PostTasks Response Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.PostTasks("tasks", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, res)
	})
}

func TestApiClient_PostTodoLists(t *testing.T) {
	t.Run("PostTodoLists Ok", func(t *testing.T) {
		in := &domain.TasksImport{
			Tasks: []*domain.Task{
				{
					ID:        1,
					Name:      "test1",
					Active:    false,
					ProjectID: 1,
					ForeignID: "test2",
				},
			},
			Notifications: []string{},
		}

		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			b, err := json.Marshal(in)
			assert.NoError(t, err)
			res.Write(b)
		}))

		client := &TogglApiClient{URL: srv.URL}
		out, err := client.PostTodoLists("todos", nil)
		assert.NoError(t, err)
		assert.EqualValues(t, *in, *out)
	})

	t.Run("PostTodoLists Response Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.PostTodoLists("todos", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, res)
	})
}

func TestApiClient_PostUsers(t *testing.T) {
	t.Run("PostUsers Ok", func(t *testing.T) {
		in := &domain.UsersImport{
			WorkspaceUsers: []*domain.User{
				{
					ID:             1,
					Email:          "test",
					Name:           "test2",
					SendInvitation: false,
					ForeignID:      "test3",
				},
			},
			Notifications: []string{},
		}

		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			b, err := json.Marshal(in)
			assert.NoError(t, err)
			res.Write(b)
		}))

		client := &TogglApiClient{URL: srv.URL}
		out, err := client.PostUsers("users", nil)
		assert.NoError(t, err)
		assert.EqualValues(t, *in, *out)
	})

	t.Run("PostUsers Response Error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { res.WriteHeader(200) }))

		client := &TogglApiClient{URL: srv.URL}
		res, err := client.PostUsers("users", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
		assert.Empty(t, res)
	})
}

func generateTasks(nr int) []*domain.Task {
	var ts []*domain.Task
	for i := 0; i < nr; i++ {
		t := domain.Task{ID: i, Name: `Name`, Active: i%2 == 0, ForeignID: fmt.Sprintf("%d", i), ProjectID: i}
		ts = append(ts, &t)
	}
	return ts
}

func TestTaskSplitting(t *testing.T) {
	c := &TogglApiClient{URL: ""}

	taskCount := 9007
	for i := 1; i < 5; i++ {
		ts := generateTasks(taskCount * i)
		trs, err := c.AdjustRequestSize(ts, 1)
		if err != nil {
			t.Error(err)
		}
		if len(trs) != i {
			t.Errorf("Expected split %d\n", i)
		}
		recievedTaskCount := 0
		for _, tr := range trs {
			recievedTaskCount += len(tr.Tasks)
		}
		if recievedTaskCount != taskCount*i {
			t.Errorf("Expected to get %d tasks but got %d", taskCount, recievedTaskCount)
		}
	}
}

func TestTaskSplittingSmallCount(t *testing.T) {
	c := &TogglApiClient{URL: ""}

	ts := generateTasks(3)
	trs, err := c.AdjustRequestSize(ts, 3)
	if err != nil {
		t.Error(err)
	}
	if len(trs) != 3 {
		t.Error("Expected split 3")
	}
	for _, tr := range trs {
		if len(tr.Tasks) != 1 {
			t.Error("Expected 1 task per request")
		}
	}
}

func TestTaskSplittingSmallDifferent(t *testing.T) {
	cl := &TogglApiClient{URL: ""}

	counts := []int{3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 103, 107, 109, 113, 127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179, 181, 191, 193, 197, 199, 211, 223, 227, 229, 233, 239, 241, 251, 257, 263, 269, 271, 277, 281, 283, 293, 307, 311, 313, 317, 331, 337, 347, 349, 353, 359, 367, 373, 379, 383, 389, 397, 401, 409, 419, 421, 431, 433, 439, 443, 449, 457, 461, 463, 467, 479, 487, 491, 499, 503, 509, 521, 523, 541, 547, 557, 563, 569, 571, 577, 587, 593, 599, 601, 607, 613, 617, 619, 631, 641, 643, 647, 653, 659, 661, 673, 677, 683, 691, 701, 709, 719, 727, 733, 739, 743, 751, 757, 761, 769, 773, 787, 797, 809, 811, 821, 823, 827, 829, 839, 853, 857, 859, 863, 877, 881, 883, 887, 907, 911, 919, 929, 937, 941, 947, 953, 967, 971, 977, 983, 991, 997}
	for _, c := range counts {
		ts := generateTasks(c)
		trs, err := cl.AdjustRequestSize(ts, 3)
		if err != nil {
			t.Error(err)
		}
		if len(trs) != 3 {
			t.Error("Expected split 3")
		}
		totalCount := 0
		for _, tr := range trs {
			totalCount += len(tr.Tasks)
		}
		if totalCount != c {
			t.Errorf("Expected total of %d tasks but got %d\n", c, totalCount)
		}
	}
}
