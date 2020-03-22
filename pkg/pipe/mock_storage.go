// Code generated by mockery v1.0.0. DO NOT EDIT.

package pipe

import integrations "github.com/toggl/pipes-api/pkg/integrations"
import mock "github.com/stretchr/testify/mock"

// MockStorage is an autogenerated mock type for the Storage type
type MockStorage struct {
	mock.Mock
}

// Delete provides a mock function with given fields: p, workspaceID
func (_m *MockStorage) Delete(p *Pipe, workspaceID int) error {
	ret := _m.Called(p, workspaceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(*Pipe, int) error); ok {
		r0 = rf(p, workspaceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteAuthorization provides a mock function with given fields: workspaceID, externalServiceID
func (_m *MockStorage) DeleteAuthorization(workspaceID int, externalServiceID integrations.ExternalServiceID) error {
	ret := _m.Called(workspaceID, externalServiceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, integrations.ExternalServiceID) error); ok {
		r0 = rf(workspaceID, externalServiceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteIDMappings provides a mock function with given fields: workspaceID, pipeConnectionKey, pipeStatusKey
func (_m *MockStorage) DeleteIDMappings(workspaceID int, pipeConnectionKey string, pipeStatusKey string) error {
	ret := _m.Called(workspaceID, pipeConnectionKey, pipeStatusKey)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, string, string) error); ok {
		r0 = rf(workspaceID, pipeConnectionKey, pipeStatusKey)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeletePipesByWorkspaceIDServiceID provides a mock function with given fields: workspaceID, sid
func (_m *MockStorage) DeletePipesByWorkspaceIDServiceID(workspaceID int, sid integrations.ExternalServiceID) error {
	ret := _m.Called(workspaceID, sid)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, integrations.ExternalServiceID) error); ok {
		r0 = rf(workspaceID, sid)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IsDown provides a mock function with given fields:
func (_m *MockStorage) IsDown() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// LoadAuthorization provides a mock function with given fields: workspaceID, sid
func (_m *MockStorage) LoadAuthorization(workspaceID int, sid integrations.ExternalServiceID) (*Authorization, error) {
	ret := _m.Called(workspaceID, sid)

	var r0 *Authorization
	if rf, ok := ret.Get(0).(func(int, integrations.ExternalServiceID) *Authorization); ok {
		r0 = rf(workspaceID, sid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Authorization)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, integrations.ExternalServiceID) error); ok {
		r1 = rf(workspaceID, sid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadIDMapping provides a mock function with given fields: workspaceID, key
func (_m *MockStorage) LoadIDMapping(workspaceID int, key string) (*IDMapping, error) {
	ret := _m.Called(workspaceID, key)

	var r0 *IDMapping
	if rf, ok := ret.Get(0).(func(int, string) *IDMapping); ok {
		r0 = rf(workspaceID, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*IDMapping)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, string) error); ok {
		r1 = rf(workspaceID, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadLastSync provides a mock function with given fields: p
func (_m *MockStorage) LoadLastSync(p *Pipe) {
	_m.Called(p)
}

// LoadPipe provides a mock function with given fields: workspaceID, sid, pid
func (_m *MockStorage) LoadPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Pipe, error) {
	ret := _m.Called(workspaceID, sid, pid)

	var r0 *Pipe
	if rf, ok := ret.Get(0).(func(int, integrations.ExternalServiceID, integrations.PipeID) *Pipe); ok {
		r0 = rf(workspaceID, sid, pid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Pipe)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, integrations.ExternalServiceID, integrations.PipeID) error); ok {
		r1 = rf(workspaceID, sid, pid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadPipeStatus provides a mock function with given fields: workspaceID, sid, pid
func (_m *MockStorage) LoadPipeStatus(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Status, error) {
	ret := _m.Called(workspaceID, sid, pid)

	var r0 *Status
	if rf, ok := ret.Get(0).(func(int, integrations.ExternalServiceID, integrations.PipeID) *Status); ok {
		r0 = rf(workspaceID, sid, pid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Status)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, integrations.ExternalServiceID, integrations.PipeID) error); ok {
		r1 = rf(workspaceID, sid, pid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadPipeStatuses provides a mock function with given fields: workspaceID
func (_m *MockStorage) LoadPipeStatuses(workspaceID int) (map[string]*Status, error) {
	ret := _m.Called(workspaceID)

	var r0 map[string]*Status
	if rf, ok := ret.Get(0).(func(int) map[string]*Status); ok {
		r0 = rf(workspaceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]*Status)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(workspaceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadPipes provides a mock function with given fields: workspaceID
func (_m *MockStorage) LoadPipes(workspaceID int) (map[string]*Pipe, error) {
	ret := _m.Called(workspaceID)

	var r0 map[string]*Pipe
	if rf, ok := ret.Get(0).(func(int) map[string]*Pipe); ok {
		r0 = rf(workspaceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]*Pipe)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(workspaceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadReversedIDMapping provides a mock function with given fields: workspaceID, key
func (_m *MockStorage) LoadReversedIDMapping(workspaceID int, key string) (*ReversedIDMapping, error) {
	ret := _m.Called(workspaceID, key)

	var r0 *ReversedIDMapping
	if rf, ok := ret.Get(0).(func(int, string) *ReversedIDMapping); ok {
		r0 = rf(workspaceID, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ReversedIDMapping)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, string) error); ok {
		r1 = rf(workspaceID, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadWorkspaceAuthorizations provides a mock function with given fields: workspaceID
func (_m *MockStorage) LoadWorkspaceAuthorizations(workspaceID int) (map[integrations.ExternalServiceID]bool, error) {
	ret := _m.Called(workspaceID)

	var r0 map[integrations.ExternalServiceID]bool
	if rf, ok := ret.Get(0).(func(int) map[integrations.ExternalServiceID]bool); ok {
		r0 = rf(workspaceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[integrations.ExternalServiceID]bool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(workspaceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Save provides a mock function with given fields: p
func (_m *MockStorage) Save(p *Pipe) error {
	ret := _m.Called(p)

	var r0 error
	if rf, ok := ret.Get(0).(func(*Pipe) error); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SaveAuthorization provides a mock function with given fields: a
func (_m *MockStorage) SaveAuthorization(a *Authorization) error {
	ret := _m.Called(a)

	var r0 error
	if rf, ok := ret.Get(0).(func(*Authorization) error); ok {
		r0 = rf(a)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SaveIDMapping provides a mock function with given fields: c
func (_m *MockStorage) SaveIDMapping(c *IDMapping) error {
	ret := _m.Called(c)

	var r0 error
	if rf, ok := ret.Get(0).(func(*IDMapping) error); ok {
		r0 = rf(c)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SavePipeStatus provides a mock function with given fields: p
func (_m *MockStorage) SavePipeStatus(p *Status) error {
	ret := _m.Called(p)

	var r0 error
	if rf, ok := ret.Get(0).(func(*Status) error); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
