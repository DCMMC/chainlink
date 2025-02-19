// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	context "context"

	eth "github.com/DCMMC/chainlink/core/services/eth"
	mock "github.com/stretchr/testify/mock"
)

// HeadTrackable is an autogenerated mock type for the HeadTrackable type
type HeadTrackable struct {
	mock.Mock
}

// OnNewLongestChain provides a mock function with given fields: ctx, head
func (_m *HeadTrackable) OnNewLongestChain(ctx context.Context, head eth.Head) {
	_m.Called(ctx, head)
}
