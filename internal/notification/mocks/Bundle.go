// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// Bundle is an autogenerated mock type for the Bundle type
type Bundle struct {
	mock.Mock
}

// CancelNotificationEvent provides a mock function with given fields:
func (_m *Bundle) CancelNotificationEvent() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateNotificationEvent provides a mock function with given fields:
func (_m *Bundle) CreateNotificationEvent() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateNotificationEvent provides a mock function with given fields:
func (_m *Bundle) UpdateNotificationEvent() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}