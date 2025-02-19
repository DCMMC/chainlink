// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	job "github.com/DCMMC/chainlink/core/services/job"
	mock "github.com/stretchr/testify/mock"
)

// Delegate is an autogenerated mock type for the Delegate type
type Delegate struct {
	mock.Mock
}

// AfterJobCreated provides a mock function with given fields: spec
func (_m *Delegate) AfterJobCreated(spec job.Job) {
	_m.Called(spec)
}

// BeforeJobDeleted provides a mock function with given fields: spec
func (_m *Delegate) BeforeJobDeleted(spec job.Job) {
	_m.Called(spec)
}

// JobType provides a mock function with given fields:
func (_m *Delegate) JobType() job.Type {
	ret := _m.Called()

	var r0 job.Type
	if rf, ok := ret.Get(0).(func() job.Type); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(job.Type)
	}

	return r0
}

// ServicesForSpec provides a mock function with given fields: spec
func (_m *Delegate) ServicesForSpec(spec job.Job) ([]job.Service, error) {
	ret := _m.Called(spec)

	var r0 []job.Service
	if rf, ok := ret.Get(0).(func(job.Job) []job.Service); ok {
		r0 = rf(spec)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]job.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(job.Job) error); ok {
		r1 = rf(spec)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
