// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	domain "github.com/toggl/pipes-api/pkg/domain"
)

// IDMappingsStorage is an autogenerated mock type for the IDMappingsStorage type
type IDMappingsStorage struct {
	mock.Mock
}

// Delete provides a mock function with given fields: workspaceID, pipeConnectionKey, pipeStatusKey
func (_m *IDMappingsStorage) Delete(workspaceID int, pipeConnectionKey string, pipeStatusKey string) error {
	ret := _m.Called(workspaceID, pipeConnectionKey, pipeStatusKey)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, string, string) error); ok {
		r0 = rf(workspaceID, pipeConnectionKey, pipeStatusKey)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Load provides a mock function with given fields: workspaceID, key
func (_m *IDMappingsStorage) Load(workspaceID int, key string) (*domain.IDMapping, error) {
	ret := _m.Called(workspaceID, key)

	var r0 *domain.IDMapping
	if rf, ok := ret.Get(0).(func(int, string) *domain.IDMapping); ok {
		r0 = rf(workspaceID, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.IDMapping)
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

// LoadReversed provides a mock function with given fields: workspaceID, key
func (_m *IDMappingsStorage) LoadReversed(workspaceID int, key string) (*domain.ReversedIDMapping, error) {
	ret := _m.Called(workspaceID, key)

	var r0 *domain.ReversedIDMapping
	if rf, ok := ret.Get(0).(func(int, string) *domain.ReversedIDMapping); ok {
		r0 = rf(workspaceID, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.ReversedIDMapping)
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

// Save provides a mock function with given fields: c
func (_m *IDMappingsStorage) Save(c *domain.IDMapping) error {
	ret := _m.Called(c)

	var r0 error
	if rf, ok := ret.Get(0).(func(*domain.IDMapping) error); ok {
		r0 = rf(c)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
