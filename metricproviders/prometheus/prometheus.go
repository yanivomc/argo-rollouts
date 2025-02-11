package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/utils/evaluate"
	metricutil "github.com/argoproj/argo-rollouts/utils/metric"
	"github.com/argoproj/argo-rollouts/utils/query"
)

const (
	//ProviderType indicates the provider is prometheus
	ProviderType = "Prometheus"
)

// Provider contains all the required components to run a prometheus query
type Provider struct {
	api    v1.API
	logCtx log.Entry
}

// Type incidates provider is a prometheus provider
func (p *Provider) Type() string {
	return ProviderType
}

// Run queries prometheus for the metric
func (p *Provider) Run(run *v1alpha1.AnalysisRun, metric v1alpha1.Metric, args []v1alpha1.Argument) v1alpha1.Measurement {
	startTime := metav1.Now()
	newMeasurement := v1alpha1.Measurement{
		StartedAt: &startTime,
	}

	//TODO(dthomson) make timeout configuriable
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query, err := query.BuildQuery(metric.Provider.Prometheus.Query, args)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	response, err := p.api.Query(ctx, query, time.Now())
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	newValue, newStatus, err := p.processResponse(metric, response)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)

	}
	newMeasurement.Value = newValue

	newMeasurement.Status = newStatus
	finishedTime := metav1.Now()
	newMeasurement.FinishedAt = &finishedTime
	return newMeasurement
}

// Resume should not be used the prometheus provider since all the work should occur in the Run method
func (p *Provider) Resume(run *v1alpha1.AnalysisRun, metric v1alpha1.Metric, args []v1alpha1.Argument, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	p.logCtx.Warn("Prometheus provider should not execute the Resume method")
	return measurement
}

// Terminate should not be used the prometheus provider since all the work should occur in the Run method
func (p *Provider) Terminate(run *v1alpha1.AnalysisRun, metric v1alpha1.Metric, args []v1alpha1.Argument, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	p.logCtx.Warn("Prometheus provider should not execute the Terminate method")
	return measurement
}

func (p *Provider) evaluateResult(result interface{}, metric v1alpha1.Metric) v1alpha1.AnalysisStatus {
	successCondition, err := evaluate.EvalCondition(result, metric.SuccessCondition)
	if err != nil {
		p.logCtx.Warning(err.Error())
		return v1alpha1.AnalysisStatusError
	}

	failCondition, err := evaluate.EvalCondition(result, metric.FailureCondition)
	if err != nil {
		return v1alpha1.AnalysisStatusError
	}
	if failCondition {
		return v1alpha1.AnalysisStatusFailed
	}

	if !failCondition && !successCondition {
		return v1alpha1.AnalysisStatusInconclusive
	}

	// If we reach this code path, failCondition is false and successCondition is true
	return v1alpha1.AnalysisStatusSuccessful
}

func (p *Provider) processResponse(metric v1alpha1.Metric, response model.Value) (string, v1alpha1.AnalysisStatus, error) {
	switch value := response.(type) {
	case *model.Scalar:
		valueStr := value.Value.String()
		result := float64(value.Value)
		newStatus := p.evaluateResult(result, metric)
		return valueStr, newStatus, nil
	case model.Vector:
		result := make([]float64, len(value))
		valueStr := "["
		for _, s := range value {
			if s != nil {
				valueStr = valueStr + s.Value.String() + ","
				result = append(result, float64(s.Value))
			}
		}
		// if we appended to the string, we should remove the last comma on the string
		if len(valueStr) > 1 {
			valueStr = valueStr[:len(valueStr)-1]
		}
		valueStr = valueStr + "]"
		newStatus := p.evaluateResult(result, metric)
		return valueStr, newStatus, nil
	//TODO(dthomson) add other response types
	default:
		return "", v1alpha1.AnalysisStatusError, fmt.Errorf("Prometheus metric type not supported")
	}
}

// NewPrometheusProvider Creates a new Prometheus client
func NewPrometheusProvider(api v1.API, logCtx log.Entry) *Provider {
	return &Provider{
		logCtx: logCtx,
		api:    api,
	}
}

// NewPrometheusAPI generates a prometheus API from the metric configuration
func NewPrometheusAPI(metric v1alpha1.Metric) (v1.API, error) {
	client, err := api.NewClient(api.Config{
		Address: metric.Provider.Prometheus.Server,
	})
	if err != nil {
		return nil, err
	}

	return v1.NewAPI(client), nil
}
