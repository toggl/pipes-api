package freshbooks

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type (
	Api struct {
		apiUrl   string
		apiToken string
		perPage  int
		users    []User
		tasks    []Task
		clients  []Client
		projects []Project
	}
	Request struct {
		XMLName xml.Name `xml:"request"`
		Method  string   `xml:"method,attr"`
		PerPage int      `xml:"per_page"`
		Page    int      `xml:"page"`
	}
	TimeEntryRequest struct {
		XMLName   xml.Name  `xml:"request"`
		Method    string    `xml:"method,attr"`
		TimeEntry TimeEntry `xml:"time_entry"`
	}
	Response struct {
		Error    string      `xml:"error"`
		Clients  ClientList  `xml:"clients"`
		Projects ProjectList `xml:"projects"`
		Tasks    TaskList    `xml:"tasks"`
		Users    UserList    `xml:"staff_members"`
	}
	TimeEntryResponse struct {
		Status      string `xml:"status,attr"`
		Error       string `xml:"error"`
		Code        string `xml:"code"`
		Field       string `xml:"field"`
		TimeEntryId int    `xml:"time_entry_id"`
	}
	Pagination struct {
		Page    int `xml:"page,attr"`
		Total   int `xml:"total,attr"`
		PerPage int `xml:"per_page,attr"`
	}
	ClientList struct {
		Pagination
		Clients []Client `xml:"client"`
	}
	ProjectList struct {
		Pagination
		Projects []Project `xml:"project"`
	}
	TaskList struct {
		Pagination
		Tasks []Task `xml:"task"`
	}
	UserList struct {
		Pagination
		Users []User `xml:"member"`
	}
	Client struct {
		ClientId int    `xml:"client_id"`
		Name     string `xml:"organization"`
	}
	Project struct {
		ProjectId int    `xml:"project_id"`
		ClientId  string `xml:"client_id"`
		Name      string `xml:"name"`
		TaskIds   []int  `xml:"tasks>task>task_id"`
		UserIds   []int  `xml:"staff>staff>staff_id"`
	}
	Task struct {
		TaskId int    `xml:"task_id"`
		Name   string `xml:"name"`
	}
	User struct {
		UserId    int    `xml:"staff_id"`
		Email     string `xml:"email"`
		FirstName string `xml:"first_name"`
		LastName  string `xml:"last_name"`
	}
	TimeEntry struct {
		TimeEntryId int     `xml:"time_entry_id"`
		ProjectId   int     `xml:"project_id"` // Required
		TaskId      int     `xml:"task_id"`    // Required
		UserId      int     `xml:"staff_id"`   // Required
		Date        string  `xml:"date"`       // Required
		Notes       string  `xml:"notes"`
		Hours       float64 `xml:"hours"`
	}
)

func NewApi(account string, token string) *Api {
	url := fmt.Sprintf("https://%s.freshbooks.com/api/2.1/xml-in", account)
	fb := Api{apiUrl: url, apiToken: token, perPage: 25}
	fb.users = make([]User, 0)
	fb.tasks = make([]Task, 0)
	fb.clients = make([]Client, 0)
	fb.projects = make([]Project, 0)
	return &fb
}

func (this *Api) Clients() ([]Client, error) {
	err := this.fetchClients(1)
	return this.clients, err
}

func (this *Api) Projects() ([]Project, error) {
	err := this.fetchProjects(1)
	return this.projects, err
}

func (this *Api) Tasks() ([]Task, error) {
	err := this.fetchTasks(1)
	return this.tasks, err
}

func (this *Api) Users() ([]User, error) {
	err := this.fetchUsers(1)
	return this.users, err
}

func (this *Api) fetchClients(page int) error {
	request := &Request{Method: "client.list", Page: page, PerPage: this.perPage}
	result, err := this.makeRequest(request)
	if err != nil {
		return err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return err
	}
	if len(parsedInto.Error) > 0 {
		return errors.New(parsedInto.Error)
	}
	this.clients = append(this.clients, parsedInto.Clients.Clients...)
	if parsedInto.Clients.Total > parsedInto.Clients.PerPage*page {
		return this.fetchClients(page + 1)
	}
	return nil
}

func (this *Api) fetchProjects(page int) error {
	request := &Request{Method: "project.list", Page: page, PerPage: this.perPage}
	result, err := this.makeRequest(request)
	if err != nil {
		return err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return (err)
	}
	if len(parsedInto.Error) > 0 {
		return errors.New(parsedInto.Error)
	}
	this.projects = append(this.projects, parsedInto.Projects.Projects...)
	if parsedInto.Projects.Total > parsedInto.Projects.PerPage*page {
		return this.fetchProjects(page + 1)
	}
	return nil
}

func (this *Api) fetchTasks(page int) error {
	request := &Request{Method: "task.list", Page: page, PerPage: this.perPage}
	result, err := this.makeRequest(request)
	if err != nil {
		return err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return err
	}
	if len(parsedInto.Error) > 0 {
		return errors.New(parsedInto.Error)
	}
	this.tasks = append(this.tasks, parsedInto.Tasks.Tasks...)
	if parsedInto.Tasks.Total > parsedInto.Tasks.PerPage*page {
		return this.fetchTasks(page + 1)
	}
	return nil
}

func (this *Api) fetchUsers(page int) error {
	request := &Request{Method: "staff.list", Page: page, PerPage: this.perPage}
	result, err := this.makeRequest(request)
	if err != nil {
		return err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return err
	}
	if len(parsedInto.Error) > 0 {
		return errors.New(parsedInto.Error)
	}
	this.users = append(this.users, parsedInto.Users.Users...)
	if parsedInto.Users.Total > parsedInto.Users.PerPage*page {
		return this.fetchUsers(page + 1)
	}
	return nil
}

func (this *Api) SaveTimeEntry(timeEntry *TimeEntry) (int, error) {
	var method string
	if timeEntry.TimeEntryId != 0 {
		method = "time_entry.update"
	} else {
		method = "time_entry.create"
	}
	request := &TimeEntryRequest{Method: method, TimeEntry: *timeEntry}
	result, err := this.makeRequest(request)
	if err != nil {
		return 0, err
	}
	parsedInto := TimeEntryResponse{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return 0, err
	}
	if parsedInto.Status == "ok" {
		return parsedInto.TimeEntryId, nil
	}
	return 0, errors.New(parsedInto.Error)
}

func (this *Api) makeRequest(request interface{}) (*[]byte, error) {
	xmlRequest, err := xml.MarshalIndent(request, "", "  ")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", this.apiUrl, bytes.NewBuffer(xmlRequest))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(this.apiToken, "X")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(response.Status)
	}

	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
