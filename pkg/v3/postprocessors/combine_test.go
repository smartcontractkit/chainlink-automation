package postprocessors

import (
	"context"
	"fmt"
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCombinedPostprocessor(t *testing.T) {
	t.Run("no error returned from any processors", func(t *testing.T) {
		one := new(MockPostProcessor)
		two := new(MockPostProcessor)
		tre := new(MockPostProcessor)

		ctx := context.Background()
		rst := []ocr2keepers.CheckResult{}

		one.On("PostProcess", ctx, rst).Return(nil)
		two.On("PostProcess", ctx, rst).Return(nil)
		tre.On("PostProcess", ctx, rst).Return(nil)

		cmb := NewCombinedPostprocessor(one, two, tre)

		err := cmb.PostProcess(ctx, rst)

		assert.NoError(t, err, "no error expected from combined post processing")

		one.AssertExpectations(t)
		two.AssertExpectations(t)
		tre.AssertExpectations(t)
	})

	t.Run("error returned from single processor and all processors run", func(t *testing.T) {
		one := new(MockPostProcessor)
		two := new(MockPostProcessor)
		tre := new(MockPostProcessor)

		ctx := context.Background()
		rst := []ocr2keepers.CheckResult{}
		tst := fmt.Errorf("test")

		one.On("PostProcess", ctx, rst).Return(nil)
		two.On("PostProcess", ctx, rst).Return(tst)
		tre.On("PostProcess", ctx, rst).Return(nil)

		cmb := NewCombinedPostprocessor(one, two, tre)

		err := cmb.PostProcess(ctx, rst)

		// expect one error to be surfaced
		assert.ErrorIs(t, err, tst, "single error expected from combined post processing")

		// all post processors should still run
		one.AssertExpectations(t)
		two.AssertExpectations(t)
		tre.AssertExpectations(t)
	})

	t.Run("error returned from all processors", func(t *testing.T) {
		one := new(MockPostProcessor)
		two := new(MockPostProcessor)
		tre := new(MockPostProcessor)

		ctx := context.Background()
		rst := []ocr2keepers.CheckResult{}

		err1 := fmt.Errorf("test")
		err2 := fmt.Errorf("test")
		err3 := fmt.Errorf("test")

		one.On("PostProcess", ctx, rst).Return(err1)
		two.On("PostProcess", ctx, rst).Return(err2)
		tre.On("PostProcess", ctx, rst).Return(err3)

		cmb := NewCombinedPostprocessor(one, two, tre)

		err := cmb.PostProcess(ctx, rst)

		// expect one error to be surfaced
		assert.ErrorIs(t, err, err1, "joined error expected from combined postprocessor")
		assert.ErrorIs(t, err, err2, "joined error expected from combined postprocessor")
		assert.ErrorIs(t, err, err3, "joined error expected from combined postprocessor")

		// all post processors should still run
		one.AssertExpectations(t)
		two.AssertExpectations(t)
		tre.AssertExpectations(t)
	})
}

type MockPostProcessor struct {
	mock.Mock
}

func (_m *MockPostProcessor) PostProcess(ctx context.Context, r []ocr2keepers.CheckResult) error {
	ret := _m.Called(ctx, r)
	return ret.Error(0)
}
