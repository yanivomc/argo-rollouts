package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AnalysisTemplate holds the template for performing canary analysis
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AnalysisTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AnalysisTemplateSpec `json:"spec"`
}

// AnalysisTemplateList is a list of AnalysisTemplate resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AnalysisTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AnalysisTemplate `json:"items"`
}

// AnalysisTemplateSpec is the specification for a AnalysisTemplate resource
type AnalysisTemplateSpec struct {
	// Metrics contains the list of metrics to query as part of an analysis run
	Metrics []Metric `json:"metrics"`
}

// Metric defines a metric in which to perform analysis
type Metric struct {
	// Name is the name of the metric
	Name string `json:"name"`
	// Interval defines the interval in seconds between each metric analysis
	// If omitted, will perform the metric analysis only once
	Interval *int32 `json:"interval,omitempty"`
	// Count is the number of times to run measurement. If both interval and count are omitted,
	// the effective count is 1. If only interval is specified, metric runs indefinitely.
	// A count > 1 must specify an interval.
	Count int32 `json:"count,omitempty"`
	// SuccessCondition is an expression which determines if a measurement is considered successful
	// Expression is a goevaluate expression. The keyword `result` is a variable reference to the
	// value of measurement. Results can be both structured data or primitive.
	// Examples:
	//   result > 10
	//   (result.requests_made * result.requests_succeeded / 100) >= 90
	//   result IN (red, yellow)
	SuccessCondition string `json:"successCondition,omitempty"`
	// FailureCondition is an expression which determines if a measurement is considered failed
	// If both success and failure conditions are specified, and the measurement does not fall into
	// either condition, the measurement is considered Inconclusive
	FailureCondition string `json:"failureCondition,omitempty"`
	// MaxFailures is the maximum number of times the measurement is allowed to fail, before the
	// entire metric is considered Failed (default: 0)
	MaxFailures int32 `json:"maxFailures,omitempty"`
	// MaxInconclusive is the maximum number of times the measurement is allowed to measure
	// Inconclusive, before the entire metric is considered Inconclusive (default: 0)
	MaxInconclusive int32 `json:"maxInconclusive,omitempty"`
	// MaxConsecutiveErrors is the maximum number of times the measurement is allowed to error in
	// succession, before the metric is considered error (default: 4)
	MaxConsecutiveErrors *int32 `json:"maxConsecutiveErrors,omitempty"`
	// FailFast will fail the entire analysis run prematurely
	FailFast bool `json:"failFast,omitempty"`
	// Provider configuration to the external system to use to verify the analysis
	Provider MetricProvider `json:"provider"`
}

// EffectiveCount is the effective count based on whether or not count/interval is specified
// If neither count or interval is specified, the effective count is 1
// If only interval is specified, metric runs indefinitely and there is no effective count (nil)
// Otherwise, it is the user specified value
func (m *Metric) EffectiveCount() *int32 {
	if m.Count == 0 {
		if m.Interval == nil {
			one := int32(1)
			return &one
		}
		return nil
	}
	return &m.Count
}

// MetricProvider which external system to use to verify the analysis
// Only one of the fields in this struct should be non-nil
type MetricProvider struct {
	// Prometheus specifies the prometheus metric to query
	Prometheus *PrometheusMetric `json:"prometheus,omitempty"`
	// Job specifies the job metric run
	Job *JobMetric `json:"job,omitempty"`
}

// AnalysisStatus is the overall status of an AnalysisRun, MetricResult, or Measurement
type AnalysisStatus string

// Possible AnalysisStatus values
const (
	AnalysisStatusPending      AnalysisStatus = "Pending"
	AnalysisStatusRunning      AnalysisStatus = "Running"
	AnalysisStatusSuccessful   AnalysisStatus = "Successful"
	AnalysisStatusFailed       AnalysisStatus = "Failed"
	AnalysisStatusError        AnalysisStatus = "Error"
	AnalysisStatusInconclusive AnalysisStatus = "Inconclusive"
)

// Completed returns whether or not the analysis status is considered completed
func (as AnalysisStatus) Completed() bool {
	switch as {
	case AnalysisStatusSuccessful, AnalysisStatusFailed, AnalysisStatusError, AnalysisStatusInconclusive:
		return true
	}
	return false
}

// PrometheusMetric defines the prometheus query to perform canary analysis
type PrometheusMetric struct {
	// Server is the address and port of the prometheus server
	Server string `json:"server,omitempty"`
	// Query is a raw prometheus query to perform
	Query string `json:"query,omitempty"`
}

// JobMetric defines a job to run which acts as a metric
type JobMetric struct {
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec     batchv1.JobSpec   `json:"spec"`
}

// AnalysisRun is an instantiation of an AnalysisTemplate
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AnalysisRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AnalysisRunSpec    `json:"spec"`
	Status            *AnalysisRunStatus `json:"status,omitempty"`
}

// AnalysisRunList is a list of AnalysisTemplate resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AnalysisRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AnalysisRun `json:"items"`
}

// AnalysisRunSpec is the spec for a AnalysisRun resource
type AnalysisRunSpec struct {
	// AnalysisSpec holds the AnalysisSpec definition for performing analysis
	AnalysisSpec AnalysisTemplateSpec `json:"analysisSpec"`
	// Arguments hold the arguments to the run to be used by metric providers
	Arguments []Argument `json:"arguments,omitempty"`
	// ReplicaSets identifies the ReplicaSets in which to monitor to decide when to begin analysis
	ReplicaSets []string `json:"replicaSets,omitempty"`
	// Terminate is used to prematurely stop the run (e.g. rollout completed and analysis is no longer desired)
	Terminate bool `json:"terminate,omitempty"`
}

// Argument is an argument to an AnalysisRun
type Argument struct {
	// Name is the name of the argument
	Name string `json:"name"`
	// Value is the value of the argument
	Value string `json:"value"`
}

// AnalysisRunStatus is the status for a AnalysisRun resource
type AnalysisRunStatus struct {
	// Status is the status of the analysis run
	Status AnalysisStatus `json:"status"`
	// Message is a message explaining current statuss
	Message string `json:"message,omitempty"`
	// MetricResults contains the metrics collected during the run
	MetricResults []MetricResult `json:"metricResults,omitempty"`
}

// MetricResult contain a list of the most recent measurements for a single metric along with
// counters on how often the measurement
type MetricResult struct {
	// Name is the name of the metric
	Name string `json:"name"`
	// Status is the overall aggregate status of the metric
	Status AnalysisStatus `json:"status"`
	// Measurements holds the most recent measurements collected for the metric
	Measurements []Measurement `json:"measurements,omitempty"`
	// Message contains a message describing current condition (e.g. error messages)
	Message string `json:"message,omitempty"`
	// Count is the number of times the metric was measured without Error
	// This is equal to the sum of Successful, Failed, Inconclusive
	Count int32 `json:"count,omitempty"`
	// Successful is the number of times the metric was measured Successful
	Successful int32 `json:"successful,omitempty"`
	// Failed is the number of times the metric was measured Failed
	Failed int32 `json:"failed,omitempty"`
	// Inconclusive is the number of times the metric was measured Inconclusive
	Inconclusive int32 `json:"inconclusive,omitempty"`
	// Error is the number of times an error was encountered during measurement
	Error int32 `json:"error,omitempty"`
	// ConsecutiveError is the number of times an error was encountered during measurement in succession
	// Resets to zero when non-errors are encountered
	ConsecutiveError int32 `json:"consecutiveError,omitempty"`
}

// Measurement is a point in time result value of a single metric, and the time it was measured
type Measurement struct {
	// Status is the status of this single measurement
	Status AnalysisStatus `json:"status"`
	// Message contains a message describing current condition (e.g. error messages)
	Message string `json:"message"`
	// StartedAt is the timestamp in which this measurement started to be measured
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// FinishedAt is the timestamp in which this measurement completed and value was collected
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
	// Value is the measured value of the metric
	Value string `json:"value,omitempty"`
	// Metadata stores additional metadata about this metric result, used by the different providers
	// (e.g. kayenta run ID, job name)
	Metadata map[string]string `json:"metadata,omitempty"`
}
