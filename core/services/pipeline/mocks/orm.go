// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	context "context"

	gorm "gorm.io/gorm"

	mock "github.com/stretchr/testify/mock"

	models "github.com/DCMMC/chainlink/core/store/models"

	pipeline "github.com/DCMMC/chainlink/core/services/pipeline"

	postgres "github.com/DCMMC/chainlink/core/services/postgres"

	time "time"

	uuid "github.com/satori/go.uuid"
)

// ORM is an autogenerated mock type for the ORM type
type ORM struct {
	mock.Mock
}

// CreateRun provides a mock function with given fields: db, run
func (_m *ORM) CreateRun(db postgres.Queryer, run *pipeline.Run) error {
	ret := _m.Called(db, run)

	var r0 error
	if rf, ok := ret.Get(0).(func(postgres.Queryer, *pipeline.Run) error); ok {
		r0 = rf(db, run)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateSpec provides a mock function with given fields: ctx, tx, _a2, maxTaskTimeout
func (_m *ORM) CreateSpec(ctx context.Context, tx *gorm.DB, _a2 pipeline.Pipeline, maxTaskTimeout models.Interval) (int32, error) {
	ret := _m.Called(ctx, tx, _a2, maxTaskTimeout)

	var r0 int32
	if rf, ok := ret.Get(0).(func(context.Context, *gorm.DB, pipeline.Pipeline, models.Interval) int32); ok {
		r0 = rf(ctx, tx, _a2, maxTaskTimeout)
	} else {
		r0 = ret.Get(0).(int32)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *gorm.DB, pipeline.Pipeline, models.Interval) error); ok {
		r1 = rf(ctx, tx, _a2, maxTaskTimeout)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DB provides a mock function with given fields:
func (_m *ORM) DB() *gorm.DB {
	ret := _m.Called()

	var r0 *gorm.DB
	if rf, ok := ret.Get(0).(func() *gorm.DB); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*gorm.DB)
		}
	}

	return r0
}

// DeleteRun provides a mock function with given fields: id
func (_m *ORM) DeleteRun(id int64) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(int64) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteRunsOlderThan provides a mock function with given fields: _a0, _a1
func (_m *ORM) DeleteRunsOlderThan(_a0 context.Context, _a1 time.Duration) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Duration) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FindRun provides a mock function with given fields: id
func (_m *ORM) FindRun(id int64) (pipeline.Run, error) {
	ret := _m.Called(id)

	var r0 pipeline.Run
	if rf, ok := ret.Get(0).(func(int64) pipeline.Run); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(pipeline.Run)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int64) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAllRuns provides a mock function with given fields:
func (_m *ORM) GetAllRuns() ([]pipeline.Run, error) {
	ret := _m.Called()

	var r0 []pipeline.Run
	if rf, ok := ret.Get(0).(func() []pipeline.Run); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pipeline.Run)
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

// GetUnfinishedRuns provides a mock function with given fields: _a0, _a1, _a2
func (_m *ORM) GetUnfinishedRuns(_a0 context.Context, _a1 time.Time, _a2 func(pipeline.Run) error) error {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Time, func(pipeline.Run) error) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InsertFinishedRun provides a mock function with given fields: db, run, saveSuccessfulTaskRuns
func (_m *ORM) InsertFinishedRun(db postgres.Queryer, run pipeline.Run, saveSuccessfulTaskRuns bool) (int64, error) {
	ret := _m.Called(db, run, saveSuccessfulTaskRuns)

	var r0 int64
	if rf, ok := ret.Get(0).(func(postgres.Queryer, pipeline.Run, bool) int64); ok {
		r0 = rf(db, run, saveSuccessfulTaskRuns)
	} else {
		r0 = ret.Get(0).(int64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(postgres.Queryer, pipeline.Run, bool) error); ok {
		r1 = rf(db, run, saveSuccessfulTaskRuns)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreRun provides a mock function with given fields: db, run
func (_m *ORM) StoreRun(db postgres.Queryer, run *pipeline.Run) (bool, error) {
	ret := _m.Called(db, run)

	var r0 bool
	if rf, ok := ret.Get(0).(func(postgres.Queryer, *pipeline.Run) bool); ok {
		r0 = rf(db, run)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(postgres.Queryer, *pipeline.Run) error); ok {
		r1 = rf(db, run)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateTaskRunResult provides a mock function with given fields: taskID, result
func (_m *ORM) UpdateTaskRunResult(taskID uuid.UUID, result pipeline.Result) (pipeline.Run, bool, error) {
	ret := _m.Called(taskID, result)

	var r0 pipeline.Run
	if rf, ok := ret.Get(0).(func(uuid.UUID, pipeline.Result) pipeline.Run); ok {
		r0 = rf(taskID, result)
	} else {
		r0 = ret.Get(0).(pipeline.Run)
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(uuid.UUID, pipeline.Result) bool); ok {
		r1 = rf(taskID, result)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(uuid.UUID, pipeline.Result) error); ok {
		r2 = rf(taskID, result)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
