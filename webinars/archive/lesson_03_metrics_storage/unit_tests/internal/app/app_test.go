package app

import (
	"example/internal/app/mocks"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestMetrics_DoStuff(t *testing.T) {
	testCases := []struct {
		name     string
		label1   string
		label2   string
		expCalls int
	}{

		{
			name:     "Increment with labels",
			label1:   "value1",
			label2:   "value2",
			expCalls: 1,
		},
		{
			name:     "Increment with different labels",
			label1:   "value3",
			label2:   "value4",
			expCalls: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPrometheus := mocks.NewMockPrometheusClient(ctrl)

			service := New(mockPrometheus)

			mockPrometheus.EXPECT().Inc(tc.label1, tc.label2).Times(tc.expCalls)

			service.DoStuff(tc.label1, tc.label2)
		})
	}
}
