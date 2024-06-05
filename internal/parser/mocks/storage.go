// Code generated by mockery v2.43.1. DO NOT EDIT.

package mocks

import (
	context "context"

	models "github.com/MichalMitros/google-feed-parser/internal/platform/models"
	mock "github.com/stretchr/testify/mock"
)

// Storage is an autogenerated mock type for the Storage type
type Storage struct {
	mock.Mock
}

// DeleteOldProducts provides a mock function with given fields: ctx, shopID, version, batchSize
func (_m *Storage) DeleteOldProducts(ctx context.Context, shopID int, version int64, batchSize uint) (int32, error) {
	ret := _m.Called(ctx, shopID, version, batchSize)

	if len(ret) == 0 {
		panic("no return value specified for DeleteOldProducts")
	}

	var r0 int32
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int64, uint) (int32, error)); ok {
		return rf(ctx, shopID, version, batchSize)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int, int64, uint) int32); ok {
		r0 = rf(ctx, shopID, version, batchSize)
	} else {
		r0 = ret.Get(0).(int32)
	}

	if rf, ok := ret.Get(1).(func(context.Context, int, int64, uint) error); ok {
		r1 = rf(ctx, shopID, version, batchSize)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FinishRun provides a mock function with given fields: ctx, run
func (_m *Storage) FinishRun(ctx context.Context, run *models.Run) error {
	ret := _m.Called(ctx, run)

	if len(ret) == 0 {
		panic("no return value specified for FinishRun")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Run) error); ok {
		r0 = rf(ctx, run)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StartRun provides a mock function with given fields: ctx, shopURL, version
func (_m *Storage) StartRun(ctx context.Context, shopURL string, version int64) (*models.Run, error) {
	ret := _m.Called(ctx, shopURL, version)

	if len(ret) == 0 {
		panic("no return value specified for StartRun")
	}

	var r0 *models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int64) (*models.Run, error)); ok {
		return rf(ctx, shopURL, version)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, int64) *models.Run); ok {
		r0 = rf(ctx, shopURL, version)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, int64) error); ok {
		r1 = rf(ctx, shopURL, version)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateProducts provides a mock function with given fields: ctx, products, shopID
func (_m *Storage) UpdateProducts(ctx context.Context, products []models.Product, shopID int) (int32, int32, error) {
	ret := _m.Called(ctx, products, shopID)

	if len(ret) == 0 {
		panic("no return value specified for UpdateProducts")
	}

	var r0 int32
	var r1 int32
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, []models.Product, int) (int32, int32, error)); ok {
		return rf(ctx, products, shopID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []models.Product, int) int32); ok {
		r0 = rf(ctx, products, shopID)
	} else {
		r0 = ret.Get(0).(int32)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []models.Product, int) int32); ok {
		r1 = rf(ctx, products, shopID)
	} else {
		r1 = ret.Get(1).(int32)
	}

	if rf, ok := ret.Get(2).(func(context.Context, []models.Product, int) error); ok {
		r2 = rf(ctx, products, shopID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// NewStorage creates a new instance of Storage. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewStorage(t interface {
	mock.TestingT
	Cleanup(func())
}) *Storage {
	mock := &Storage{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
