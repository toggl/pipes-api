// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import integrations "github.com/toggl/pipes-api/pkg/integrations"
import mock "github.com/stretchr/testify/mock"

import time "time"
import toggl "github.com/toggl/pipes-api/pkg/toggl"

// TogglClient is an autogenerated mock type for the TogglClient type
type TogglClient struct {
	mock.Mock
}

// AdjustRequestSize provides a mock function with given fields: tasks, split
func (_m *TogglClient) AdjustRequestSize(tasks []*toggl.Task, split int) ([]*toggl.TaskRequest, error) {
	ret := _m.Called(tasks, split)

	var r0 []*toggl.TaskRequest
	if rf, ok := ret.Get(0).(func([]*toggl.Task, int) []*toggl.TaskRequest); ok {
		r0 = rf(tasks, split)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*toggl.TaskRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]*toggl.Task, int) error); ok {
		r1 = rf(tasks, split)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTimeEntries provides a mock function with given fields: lastSync, userIDs, projectsIDs
func (_m *TogglClient) GetTimeEntries(lastSync time.Time, userIDs []int, projectsIDs []int) ([]toggl.TimeEntry, error) {
	ret := _m.Called(lastSync, userIDs, projectsIDs)

	var r0 []toggl.TimeEntry
	if rf, ok := ret.Get(0).(func(time.Time, []int, []int) []toggl.TimeEntry); ok {
		r0 = rf(lastSync, userIDs, projectsIDs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]toggl.TimeEntry)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(time.Time, []int, []int) error); ok {
		r1 = rf(lastSync, userIDs, projectsIDs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetWorkspaceIdByToken provides a mock function with given fields: token
func (_m *TogglClient) GetWorkspaceIdByToken(token string) (int, error) {
	ret := _m.Called(token)

	var r0 int
	if rf, ok := ret.Get(0).(func(string) int); ok {
		r0 = rf(token)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(token)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Ping provides a mock function with given fields:
func (_m *TogglClient) Ping() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PostClients provides a mock function with given fields: clientsPipeID, clients
func (_m *TogglClient) PostClients(clientsPipeID integrations.PipeID, clients interface{}) (*toggl.ClientsImport, error) {
	ret := _m.Called(clientsPipeID, clients)

	var r0 *toggl.ClientsImport
	if rf, ok := ret.Get(0).(func(integrations.PipeID, interface{}) *toggl.ClientsImport); ok {
		r0 = rf(clientsPipeID, clients)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*toggl.ClientsImport)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(integrations.PipeID, interface{}) error); ok {
		r1 = rf(clientsPipeID, clients)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PostProjects provides a mock function with given fields: projectsPipeID, projects
func (_m *TogglClient) PostProjects(projectsPipeID integrations.PipeID, projects interface{}) (*toggl.ProjectsImport, error) {
	ret := _m.Called(projectsPipeID, projects)

	var r0 *toggl.ProjectsImport
	if rf, ok := ret.Get(0).(func(integrations.PipeID, interface{}) *toggl.ProjectsImport); ok {
		r0 = rf(projectsPipeID, projects)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*toggl.ProjectsImport)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(integrations.PipeID, interface{}) error); ok {
		r1 = rf(projectsPipeID, projects)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PostTasks provides a mock function with given fields: tasksPipeID, tasks
func (_m *TogglClient) PostTasks(tasksPipeID integrations.PipeID, tasks interface{}) (*toggl.TasksImport, error) {
	ret := _m.Called(tasksPipeID, tasks)

	var r0 *toggl.TasksImport
	if rf, ok := ret.Get(0).(func(integrations.PipeID, interface{}) *toggl.TasksImport); ok {
		r0 = rf(tasksPipeID, tasks)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*toggl.TasksImport)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(integrations.PipeID, interface{}) error); ok {
		r1 = rf(tasksPipeID, tasks)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PostTodoLists provides a mock function with given fields: tasksPipeID, tasks
func (_m *TogglClient) PostTodoLists(tasksPipeID integrations.PipeID, tasks interface{}) (*toggl.TasksImport, error) {
	ret := _m.Called(tasksPipeID, tasks)

	var r0 *toggl.TasksImport
	if rf, ok := ret.Get(0).(func(integrations.PipeID, interface{}) *toggl.TasksImport); ok {
		r0 = rf(tasksPipeID, tasks)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*toggl.TasksImport)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(integrations.PipeID, interface{}) error); ok {
		r1 = rf(tasksPipeID, tasks)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PostUsers provides a mock function with given fields: usersPipeID, users
func (_m *TogglClient) PostUsers(usersPipeID integrations.PipeID, users interface{}) (*toggl.UsersImport, error) {
	ret := _m.Called(usersPipeID, users)

	var r0 *toggl.UsersImport
	if rf, ok := ret.Get(0).(func(integrations.PipeID, interface{}) *toggl.UsersImport); ok {
		r0 = rf(usersPipeID, users)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*toggl.UsersImport)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(integrations.PipeID, interface{}) error); ok {
		r1 = rf(usersPipeID, users)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WithAuthToken provides a mock function with given fields: authToken
func (_m *TogglClient) WithAuthToken(authToken string) {
	_m.Called(authToken)
}