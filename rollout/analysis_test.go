package rollout

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/pkg/controller"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	analysisutil "github.com/argoproj/argo-rollouts/utils/analysis"
	"github.com/argoproj/argo-rollouts/utils/conditions"
)

func analysisTemplate(name string) *v1alpha1.AnalysisTemplate {
	return &v1alpha1.AnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.AnalysisTemplateSpec{
			Metrics: []v1alpha1.Metric{{
				Name: "example",
			}},
		},
	}
}

func analysisRun(at *v1alpha1.AnalysisTemplate, analysisRunType string, r *v1alpha1.Rollout) *v1alpha1.AnalysisRun {
	labels := map[string]string{}
	podHash := controller.ComputeHash(&r.Spec.Template, r.Status.CollisionCount)
	if analysisRunType == v1alpha1.RolloutTypeStepLabel {
		labels = analysisutil.StepLabels(r, *r.Status.CurrentStepIndex, podHash)
	}
	return &v1alpha1.AnalysisRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-%s-%s-%s", r.Name, at.Name, podHash, MockGeneratedNameSuffix),
			Namespace:       metav1.NamespaceDefault,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(r, controllerKind)},
		},
		Spec: v1alpha1.AnalysisRunSpec{
			AnalysisSpec: at.Spec,
		},
	}
}

func TestCreateAnalysisRunOnAnalysisStep(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	at := analysisTemplate("bar")
	steps := []v1alpha1.CanaryStep{{
		Analysis: &v1alpha1.RolloutAnalysisStep{
			TemplateName: at.Name,
		},
	}}

	r1 := newCanaryRollout("foo", 1, nil, steps, pointer.Int32Ptr(0), intstr.FromInt(0), intstr.FromInt(1))
	r2 := bumpVersion(r1)
	ar := analysisRun(at, v1alpha1.RolloutTypeStepLabel, r2)

	rs1 := newReplicaSetWithStatus(r1, 1, 1)
	rs2 := newReplicaSetWithStatus(r2, 0, 0)
	f.kubeobjects = append(f.kubeobjects, rs1, rs2)
	f.replicaSetLister = append(f.replicaSetLister, rs1, rs2)
	rs1PodHash := rs1.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]
	rs2PodHash := rs2.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]

	r2 = updateCanaryRolloutStatus(r2, rs1PodHash, 1, 0, 1, false)
	progressingCondition, _ := newProgressingCondition(conditions.ReplicaSetUpdatedReason, rs2)
	conditions.SetRolloutCondition(&r2.Status, progressingCondition)
	availableCondition, _ := newAvailableCondition(true)
	conditions.SetRolloutCondition(&r2.Status, availableCondition)

	f.rolloutLister = append(f.rolloutLister, r2)
	f.analysisTemplateLister = append(f.analysisTemplateLister, at)
	f.objects = append(f.objects, r2, at)

	createdIndex := f.expectCreateAnalysisRunAction(ar)
	index := f.expectPatchRolloutAction(r1)

	f.run(getKey(r2, t))
	createdAr := f.getCreatedAnalysisRun(createdIndex)
	expectedArGeneratedName := fmt.Sprintf("%s-%s-%s", r2.Name, at.Name, rs2PodHash)
	expectedArName := fmt.Sprintf("%s-%s", expectedArGeneratedName, MockGeneratedNameSuffix)
	assert.Equal(t, expectedArGeneratedName, createdAr.GenerateName)

	patch := f.getPatchedRollout(index)
	expectedPatch := `{
		"status": {
			"canary": {
				"currentStepAnalysisRun": "%s"
			}
		}
	}`
	assert.Equal(t, calculatePatch(r2, fmt.Sprintf(expectedPatch, expectedArName)), patch)
}

func TestFailCreateAnalysisRunIfInvalidTemplateRef(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	steps := []v1alpha1.CanaryStep{{
		Analysis: &v1alpha1.RolloutAnalysisStep{
			TemplateName: "invalid-template-ref",
		},
	}}

	r1 := newCanaryRollout("foo", 1, nil, steps, pointer.Int32Ptr(0), intstr.FromInt(0), intstr.FromInt(1))
	r2 := bumpVersion(r1)

	rs1 := newReplicaSetWithStatus(r1, 1, 1)
	rs2 := newReplicaSetWithStatus(r2, 0, 0)
	f.kubeobjects = append(f.kubeobjects, rs1, rs2)
	f.replicaSetLister = append(f.replicaSetLister, rs1, rs2)
	rs1PodHash := rs1.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]

	r2 = updateCanaryRolloutStatus(r2, rs1PodHash, 1, 0, 1, false)
	progressingCondition, _ := newProgressingCondition(conditions.ReplicaSetUpdatedReason, rs2)
	conditions.SetRolloutCondition(&r2.Status, progressingCondition)
	availableCondition, _ := newAvailableCondition(true)
	conditions.SetRolloutCondition(&r2.Status, availableCondition)

	f.rolloutLister = append(f.rolloutLister, r2)
	f.objects = append(f.objects, r2)

	f.runExpectError(getKey(r2, t), true)
}

func TestDoNothingWhileAnalysisRunRunning(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	at := analysisTemplate("bar")
	steps := []v1alpha1.CanaryStep{{
		Analysis: &v1alpha1.RolloutAnalysisStep{
			TemplateName: at.Name,
		},
	}}

	r1 := newCanaryRollout("foo", 1, nil, steps, pointer.Int32Ptr(0), intstr.FromInt(0), intstr.FromInt(1))
	r2 := bumpVersion(r1)
	ar := analysisRun(at, v1alpha1.RolloutTypeStepLabel, r2)

	rs1 := newReplicaSetWithStatus(r1, 1, 1)
	rs2 := newReplicaSetWithStatus(r2, 0, 0)
	f.kubeobjects = append(f.kubeobjects, rs1, rs2)
	f.replicaSetLister = append(f.replicaSetLister, rs1, rs2)
	rs1PodHash := rs1.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]

	r2 = updateCanaryRolloutStatus(r2, rs1PodHash, 1, 0, 1, false)
	progressingCondition, _ := newProgressingCondition(conditions.ReplicaSetUpdatedReason, rs2)
	conditions.SetRolloutCondition(&r2.Status, progressingCondition)
	availableCondition, _ := newAvailableCondition(true)
	conditions.SetRolloutCondition(&r2.Status, availableCondition)
	r2.Status.Canary.CurrentStepAnalysisRun = ar.Name

	f.rolloutLister = append(f.rolloutLister, r2)
	f.analysisTemplateLister = append(f.analysisTemplateLister, at)
	f.analysisRunLister = append(f.analysisRunLister, ar)
	f.objects = append(f.objects, r2, at, ar)

	patchIndex := f.expectPatchRolloutAction(r2)
	f.run(getKey(r2, t))
	patch := f.getPatchedRollout(patchIndex)
	assert.Equal(t, calculatePatch(r2, OnlyObservedGenerationPatch), patch)
}

func TestCancelOlderAnalysisRuns(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	at := analysisTemplate("bar")
	steps := []v1alpha1.CanaryStep{{
		Analysis: &v1alpha1.RolloutAnalysisStep{
			TemplateName: at.Name,
		},
	}}

	r1 := newCanaryRollout("foo", 1, nil, steps, pointer.Int32Ptr(0), intstr.FromInt(0), intstr.FromInt(1))
	r2 := bumpVersion(r1)
	ar := analysisRun(at, v1alpha1.RolloutTypeStepLabel, r2)
	olderAr := ar.DeepCopy()
	olderAr.Name = "older-analysis-run"

	rs1 := newReplicaSetWithStatus(r1, 1, 1)
	rs2 := newReplicaSetWithStatus(r2, 0, 0)
	f.kubeobjects = append(f.kubeobjects, rs1, rs2)
	f.replicaSetLister = append(f.replicaSetLister, rs1, rs2)
	rs1PodHash := rs1.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]

	r2 = updateCanaryRolloutStatus(r2, rs1PodHash, 1, 0, 1, false)
	progressingCondition, _ := newProgressingCondition(conditions.ReplicaSetUpdatedReason, rs2)
	conditions.SetRolloutCondition(&r2.Status, progressingCondition)
	availableCondition, _ := newAvailableCondition(true)
	conditions.SetRolloutCondition(&r2.Status, availableCondition)
	r2.Status.Canary.CurrentStepAnalysisRun = ar.Name

	f.rolloutLister = append(f.rolloutLister, r2)
	f.analysisTemplateLister = append(f.analysisTemplateLister, at)
	f.analysisRunLister = append(f.analysisRunLister, ar, olderAr)
	f.objects = append(f.objects, r2, at, ar, olderAr)

	cancelOldAr := f.expectPatchAnalysisRunAction(olderAr)
	patchIndex := f.expectPatchRolloutAction(r2)
	f.run(getKey(r2, t))

	assert.True(t, f.verifyPatchedAnalysisRun(cancelOldAr, olderAr))
	patch := f.getPatchedRollout(patchIndex)
	assert.Equal(t, calculatePatch(r2, OnlyObservedGenerationPatch), patch)
}

func TestIncrementStepAfterSuccessfulAnalysisRun(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	at := analysisTemplate("bar")
	steps := []v1alpha1.CanaryStep{{
		Analysis: &v1alpha1.RolloutAnalysisStep{
			TemplateName: at.Name,
		},
	}}

	r1 := newCanaryRollout("foo", 1, nil, steps, pointer.Int32Ptr(0), intstr.FromInt(0), intstr.FromInt(1))
	r2 := bumpVersion(r1)
	ar := analysisRun(at, v1alpha1.RolloutTypeStepLabel, r2)
	ar.Status = &v1alpha1.AnalysisRunStatus{
		Status: v1alpha1.AnalysisStatusSuccessful,
	}

	rs1 := newReplicaSetWithStatus(r1, 1, 1)
	rs2 := newReplicaSetWithStatus(r2, 0, 0)
	f.kubeobjects = append(f.kubeobjects, rs1, rs2)
	f.replicaSetLister = append(f.replicaSetLister, rs1, rs2)
	rs1PodHash := rs1.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]

	r2 = updateCanaryRolloutStatus(r2, rs1PodHash, 1, 0, 1, false)
	r2.Status.Canary.CurrentStepAnalysisRun = ar.Name

	f.rolloutLister = append(f.rolloutLister, r2)
	f.analysisTemplateLister = append(f.analysisTemplateLister, at)
	f.analysisRunLister = append(f.analysisRunLister, ar)
	f.objects = append(f.objects, r2, at, ar)

	patchIndex := f.expectPatchRolloutAction(r2)
	f.run(getKey(r2, t))
	patch := f.getPatchedRollout(patchIndex)
	expectedPatch := `{
		"status": {
			"canary": {
				"currentStepAnalysisRun": null
			},
			"currentStepIndex": 1,
			"conditions": %s
		}
	}`
	condition := generateConditionsPatch(true, conditions.ReplicaSetUpdatedReason, rs2, false)

	assert.Equal(t, calculatePatch(r2, fmt.Sprintf(expectedPatch, condition)), patch)
}

func TestPausedStepAfterInconclusiveAnalysisRun(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	at := analysisTemplate("bar")
	steps := []v1alpha1.CanaryStep{{
		Analysis: &v1alpha1.RolloutAnalysisStep{
			TemplateName: at.Name,
		},
	}}

	r1 := newCanaryRollout("foo", 1, nil, steps, pointer.Int32Ptr(0), intstr.FromInt(0), intstr.FromInt(1))
	r2 := bumpVersion(r1)
	ar := analysisRun(at, v1alpha1.RolloutTypeStepLabel, r2)
	ar.Status = &v1alpha1.AnalysisRunStatus{
		Status: v1alpha1.AnalysisStatusInconclusive,
	}

	rs1 := newReplicaSetWithStatus(r1, 1, 1)
	rs2 := newReplicaSetWithStatus(r2, 0, 0)
	f.kubeobjects = append(f.kubeobjects, rs1, rs2)
	f.replicaSetLister = append(f.replicaSetLister, rs1, rs2)
	rs1PodHash := rs1.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]

	r2 = updateCanaryRolloutStatus(r2, rs1PodHash, 1, 0, 1, false)
	r2.Status.Canary.CurrentStepAnalysisRun = ar.Name

	f.rolloutLister = append(f.rolloutLister, r2)
	f.analysisTemplateLister = append(f.analysisTemplateLister, at)
	f.analysisRunLister = append(f.analysisRunLister, ar)
	f.objects = append(f.objects, r2, at, ar)

	patchIndex := f.expectPatchRolloutAction(r2)
	f.run(getKey(r2, t))
	patch := f.getPatchedRollout(patchIndex)
	now := metav1.Now().UTC().Format(time.RFC3339)
	expectedPatch := `{
		"spec":{
			"paused": true
		},
		"status": {
			"conditions": %s,
			"canary": {
				"currentStepAnalysisRun": null
			},
			"pauseStartTime": "%s"
		}
	}`
	condition := generateConditionsPatch(true, conditions.ReplicaSetUpdatedReason, r2, false)

	assert.Equal(t, calculatePatch(r2, fmt.Sprintf(expectedPatch, condition, now)), patch)
}

func TestErrorConditionAfterErrorAnalysisRun(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	at := analysisTemplate("bar")
	steps := []v1alpha1.CanaryStep{{
		Analysis: &v1alpha1.RolloutAnalysisStep{
			TemplateName: at.Name,
		},
	}}

	r1 := newCanaryRollout("foo", 1, nil, steps, pointer.Int32Ptr(0), intstr.FromInt(0), intstr.FromInt(1))
	r2 := bumpVersion(r1)
	ar := analysisRun(at, v1alpha1.RolloutTypeStepLabel, r2)
	ar.Status = &v1alpha1.AnalysisRunStatus{
		Status: v1alpha1.AnalysisStatusError,
		MetricResults: []v1alpha1.MetricResult{{
			Status: v1alpha1.AnalysisStatusError,
		}},
	}

	rs1 := newReplicaSetWithStatus(r1, 1, 1)
	rs2 := newReplicaSetWithStatus(r2, 0, 0)
	f.kubeobjects = append(f.kubeobjects, rs1, rs2)
	f.replicaSetLister = append(f.replicaSetLister, rs1, rs2)
	rs1PodHash := rs1.Labels[v1alpha1.DefaultRolloutUniqueLabelKey]

	r2 = updateCanaryRolloutStatus(r2, rs1PodHash, 1, 0, 1, false)
	r2.Status.Canary.CurrentStepAnalysisRun = ar.Name

	f.rolloutLister = append(f.rolloutLister, r2)
	f.analysisTemplateLister = append(f.analysisTemplateLister, at)
	f.analysisRunLister = append(f.analysisRunLister, ar)
	f.objects = append(f.objects, r2, at, ar)

	patchIndex := f.expectPatchRolloutAction(r2)
	f.run(getKey(r2, t))
	patch := f.getPatchedRollout(patchIndex)
	expectedPatch := `{
		"status": {
			"conditions": %s
		}
	}`
	condition := generateConditionsPatch(true, conditions.RolloutAnalysisRunFailedReason, r2, false)

	assert.Equal(t, calculatePatch(r2, fmt.Sprintf(expectedPatch, condition)), patch)
}
