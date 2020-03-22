// Code generated by mockery v1.0.0. DO NOT EDIT.

package pipe

import integration "github.com/toggl/pipes-api/pkg/integration"
import mock "github.com/stretchr/testify/mock"

// MockIntegrationsStorage is an autogenerated mock type for the IntegrationsStorage type
type MockIntegrationsStorage struct {
	mock.Mock
}

// IsValidPipe provides a mock function with given fields: pipeID
func (_m *MockIntegrationsStorage) IsValidPipe(pipeID integration.PipeID) bool {
	ret := _m.Called(pipeID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(integration.PipeID) bool); ok {
		r0 = rf(pipeID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// IsValidService provides a mock function with given fields: serviceID
func (_m *MockIntegrationsStorage) IsValidService(serviceID integration.ID) bool {
	ret := _m.Called(serviceID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(integration.ID) bool); ok {
		r0 = rf(serviceID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// LoadAuthorizationType provides a mock function with given fields: serviceID
func (_m *MockIntegrationsStorage) LoadAuthorizationType(serviceID integration.ID) (string, error) {
	ret := _m.Called(serviceID)

	var r0 string
	if rf, ok := ret.Get(0).(func(integration.ID) string); ok {
		r0 = rf(serviceID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(integration.ID) error); ok {
		r1 = rf(serviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadIntegrations provides a mock function with given fields:
func (_m *MockIntegrationsStorage) LoadIntegrations() ([]*Integration, error) {
	ret := _m.Called()

	var r0 []*Integration
	if rf, ok := ret.Get(0).(func() []*Integration); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*Integration)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveAuthorizationType provides a mock function with given fields: serviceID, authType
func (_m *MockIntegrationsStorage) SaveAuthorizationType(serviceID integration.ID, authType string) error {
	ret := _m.Called(serviceID, authType)

	var r0 error
	if rf, ok := ret.Get(0).(func(integration.ID, string) error); ok {
		r0 = rf(serviceID, authType)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}