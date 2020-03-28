// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	domain "github.com/toggl/pipes-api/pkg/domain"
)

// Queue is an autogenerated mock type for the Queue type
type Queue struct {
	mock.Mock
}

// LoadScheduledPipes provides a mock function with given fields:
func (_m *Queue) LoadScheduledPipes() ([]*domain.Pipe, error) {
	ret := _m.Called()

	var r0 []*domain.Pipe
	if rf, ok := ret.Get(0).(func() []*domain.Pipe); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*domain.Pipe)
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

// ScheduleAutomaticPipesSynchronization provides a mock function with given fields:
func (_m *Queue) ScheduleAutomaticPipesSynchronization() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SchedulePipeSynchronization provides a mock function with given fields: _a0
func (_m *Queue) SchedulePipeSynchronization(_a0 *domain.Pipe) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*domain.Pipe) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MarkPipeSynchronized provides a mock function with given fields: _a0
func (_m *Queue) MarkPipeSynchronized(_a0 *domain.Pipe) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*domain.Pipe) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
