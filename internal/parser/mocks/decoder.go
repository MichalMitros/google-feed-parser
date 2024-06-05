// Code generated by mockery v2.43.1. DO NOT EDIT.

package mocks

import (
	context "context"
	io "io"

	mock "github.com/stretchr/testify/mock"

	models "github.com/MichalMitros/google-feed-parser/internal/platform/models"
)

// Decoder is an autogenerated mock type for the Decoder type
type Decoder struct {
	mock.Mock
}

// Decode provides a mock function with given fields: _a0, _a1, _a2
func (_m *Decoder) Decode(_a0 context.Context, _a1 io.Reader, _a2 chan<- models.ParsingResult) error {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for Decode")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, io.Reader, chan<- models.ParsingResult) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewDecoder creates a new instance of Decoder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDecoder(t interface {
	mock.TestingT
	Cleanup(func())
}) *Decoder {
	mock := &Decoder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}