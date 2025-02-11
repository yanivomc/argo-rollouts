package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
)

func TestIsWorst(t *testing.T) {
	assert.False(t, IsWorse(v1alpha1.AnalysisStatusSuccessful, v1alpha1.AnalysisStatusSuccessful))
	assert.True(t, IsWorse(v1alpha1.AnalysisStatusSuccessful, v1alpha1.AnalysisStatusInconclusive))
	assert.True(t, IsWorse(v1alpha1.AnalysisStatusSuccessful, v1alpha1.AnalysisStatusError))
	assert.True(t, IsWorse(v1alpha1.AnalysisStatusSuccessful, v1alpha1.AnalysisStatusFailed))

	assert.False(t, IsWorse(v1alpha1.AnalysisStatusInconclusive, v1alpha1.AnalysisStatusSuccessful))
	assert.False(t, IsWorse(v1alpha1.AnalysisStatusInconclusive, v1alpha1.AnalysisStatusInconclusive))
	assert.True(t, IsWorse(v1alpha1.AnalysisStatusInconclusive, v1alpha1.AnalysisStatusError))
	assert.True(t, IsWorse(v1alpha1.AnalysisStatusInconclusive, v1alpha1.AnalysisStatusFailed))

	assert.False(t, IsWorse(v1alpha1.AnalysisStatusError, v1alpha1.AnalysisStatusError))
	assert.False(t, IsWorse(v1alpha1.AnalysisStatusError, v1alpha1.AnalysisStatusSuccessful))
	assert.False(t, IsWorse(v1alpha1.AnalysisStatusError, v1alpha1.AnalysisStatusInconclusive))
	assert.True(t, IsWorse(v1alpha1.AnalysisStatusError, v1alpha1.AnalysisStatusFailed))

	assert.False(t, IsWorse(v1alpha1.AnalysisStatusFailed, v1alpha1.AnalysisStatusSuccessful))
	assert.False(t, IsWorse(v1alpha1.AnalysisStatusFailed, v1alpha1.AnalysisStatusInconclusive))
	assert.False(t, IsWorse(v1alpha1.AnalysisStatusFailed, v1alpha1.AnalysisStatusError))
	assert.False(t, IsWorse(v1alpha1.AnalysisStatusFailed, v1alpha1.AnalysisStatusFailed))
}

func TestIsFastFailTerminating(t *testing.T) {
	run := &v1alpha1.AnalysisRun{
		Status: &v1alpha1.AnalysisRunStatus{
			Status: v1alpha1.AnalysisStatusRunning,
			MetricResults: []v1alpha1.MetricResult{
				{
					Name:   "other-metric",
					Status: v1alpha1.AnalysisStatusRunning,
				},
				{
					Name:   "success-rate",
					Status: v1alpha1.AnalysisStatusRunning,
				},
			},
		},
	}
	successRate := run.Status.MetricResults[1]
	assert.False(t, IsTerminating(run))
	successRate.Status = v1alpha1.AnalysisStatusError
	run.Status.MetricResults[1] = successRate
	assert.True(t, IsTerminating(run))
	successRate.Status = v1alpha1.AnalysisStatusFailed
	run.Status.MetricResults[1] = successRate
	assert.True(t, IsTerminating(run))
	successRate.Status = v1alpha1.AnalysisStatusInconclusive
	run.Status.MetricResults[1] = successRate
	assert.True(t, IsTerminating(run))
	run.Status.MetricResults = nil
	assert.False(t, IsTerminating(run))
	run.Status = nil
	assert.False(t, IsTerminating(run))
}

func TestGetResult(t *testing.T) {
	run := &v1alpha1.AnalysisRun{
		Status: &v1alpha1.AnalysisRunStatus{
			Status: v1alpha1.AnalysisStatusRunning,
			MetricResults: []v1alpha1.MetricResult{
				{
					Name:   "success-rate",
					Status: v1alpha1.AnalysisStatusRunning,
				},
			},
		},
	}
	assert.Nil(t, GetResult(run, "non-existent"))
	assert.Equal(t, run.Status.MetricResults[0], *GetResult(run, "success-rate"))
}

func TestSetResult(t *testing.T) {
	run := &v1alpha1.AnalysisRun{
		Status: &v1alpha1.AnalysisRunStatus{},
	}
	res := v1alpha1.MetricResult{
		Name:   "success-rate",
		Status: v1alpha1.AnalysisStatusRunning,
	}

	SetResult(run, res)
	assert.Equal(t, res, run.Status.MetricResults[0])
	res.Status = v1alpha1.AnalysisStatusFailed
	SetResult(run, res)
	assert.Equal(t, res, run.Status.MetricResults[0])
}

func TestMetricCompleted(t *testing.T) {
	run := &v1alpha1.AnalysisRun{
		Status: &v1alpha1.AnalysisRunStatus{
			Status: v1alpha1.AnalysisStatusRunning,
			MetricResults: []v1alpha1.MetricResult{
				{
					Name:   "success-rate",
					Status: v1alpha1.AnalysisStatusRunning,
				},
			},
		},
	}
	assert.False(t, MetricCompleted(run, "non-existent"))
	assert.False(t, MetricCompleted(run, "success-rate"))

	run.Status.MetricResults[0] = v1alpha1.MetricResult{
		Name:   "success-rate",
		Status: v1alpha1.AnalysisStatusError,
	}
	assert.True(t, MetricCompleted(run, "success-rate"))
}

func TestLastMeasurement(t *testing.T) {
	m1 := v1alpha1.Measurement{
		Status: v1alpha1.AnalysisStatusSuccessful,
		Value:  "99",
	}
	m2 := v1alpha1.Measurement{
		Status: v1alpha1.AnalysisStatusSuccessful,
		Value:  "98",
	}
	run := &v1alpha1.AnalysisRun{
		Status: &v1alpha1.AnalysisRunStatus{
			Status: v1alpha1.AnalysisStatusRunning,
			MetricResults: []v1alpha1.MetricResult{
				{
					Name:         "success-rate",
					Status:       v1alpha1.AnalysisStatusRunning,
					Measurements: []v1alpha1.Measurement{m1, m2},
				},
			},
		},
	}
	assert.Nil(t, LastMeasurement(run, "non-existent"))
	assert.Equal(t, m2, *LastMeasurement(run, "success-rate"))
	successRate := run.Status.MetricResults[0]
	successRate.Measurements = []v1alpha1.Measurement{}
	run.Status.MetricResults[0] = successRate
	assert.Nil(t, LastMeasurement(run, "success-rate"))
}

func TestIsTerminating(t *testing.T) {
	run := &v1alpha1.AnalysisRun{
		Status: &v1alpha1.AnalysisRunStatus{
			Status: v1alpha1.AnalysisStatusRunning,
			MetricResults: []v1alpha1.MetricResult{
				{
					Name:   "other-metric",
					Status: v1alpha1.AnalysisStatusRunning,
				},
				{
					Name:   "success-rate",
					Status: v1alpha1.AnalysisStatusRunning,
				},
			},
		},
	}
	assert.False(t, IsTerminating(run))
	run.Spec.Terminate = true
	assert.True(t, IsTerminating(run))
	run.Spec.Terminate = false
	successRate := run.Status.MetricResults[1]
	successRate.Status = v1alpha1.AnalysisStatusError
	run.Status.MetricResults[1] = successRate
	assert.True(t, IsTerminating(run))
}
