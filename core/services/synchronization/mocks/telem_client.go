// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	context "context"

	telem "github.com/DCMMC/chainlink/core/services/synchronization/telem"
	mock "github.com/stretchr/testify/mock"
)

// TelemClient is an autogenerated mock type for the TelemClient type
type TelemClient struct {
	mock.Mock
}

// Telem provides a mock function with given fields: ctx, in
func (_m *TelemClient) Telem(ctx context.Context, in *telem.TelemRequest) (*telem.TelemResponse, error) {
	ret := _m.Called(ctx, in)

	var r0 *telem.TelemResponse
	if rf, ok := ret.Get(0).(func(context.Context, *telem.TelemRequest) *telem.TelemResponse); ok {
		r0 = rf(ctx, in)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*telem.TelemResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *telem.TelemRequest) error); ok {
		r1 = rf(ctx, in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
