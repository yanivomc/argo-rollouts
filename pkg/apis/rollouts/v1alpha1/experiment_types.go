package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Experiment is a specification for an Experiment resource
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Experiment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExperimentSpec   `json:"spec"`
	Status ExperimentStatus `json:"status,omitempty"`
}

// ExperimentSpec is the spec for a Experiment resource
type ExperimentSpec struct {
	// Templates A list of PodSpecs that define the ReplicaSets that should be run during an experiment.
	Templates []TemplateSpec `json:"templates"`
	// Duration the amount of time for the experiment to run. If not listed, the experiment will run for an
	// indefinite amount of time
	// +optional
	Duration *int32 `json:"duration,omitempty"`
	// ProgressDeadlineSeconds The maximum time in seconds for a experiment to
	// make progress before it is considered to be failed. Argo Rollouts will
	// continue to process failed experiments and a condition with a
	// ProgressDeadlineExceeded reason will be surfaced in the experiment status.
	// Note that progress will not be estimated during the time a experiment is paused.
	// Defaults to 600s.
	// +optional
	ProgressDeadlineSeconds *int32 `json:"progressDeadlineSeconds,omitempty"`
}

type TemplateSpec struct {
	// Name of the template used to identity replicaset running for this experiment
	Name string `json:"name"`
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Minimum number of seconds for which a newly created pod should be ready
	// without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready)
	// +optional
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`
	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this experiment.
	// It must match the pod template's labels. Each selector must be unique to the other selectors in the other templates
	Selector *metav1.LabelSelector `json:"selector"`
	// Template describes the pods that will be created.
	Template corev1.PodTemplateSpec `json:"template"`
}

// TemplateStatus is the status of a specific template of an Experiment
type TemplateStatus struct {
	// Name of the template used to identity which hash to compare to the hash
	Name string `json:"name"`
	// Total number of non-terminated pods targeted by this experiment (their labels match the selector).
	Replicas int32 `json:"replicas"`
	// Total number of non-terminated pods targeted by this experiment that have the desired template spec.
	UpdatedReplicas int32 `json:"updatedReplicas"`
	// Total number of ready pods targeted by this experiment.
	ReadyReplicas int32 `json:"readyReplicas"`
	// Total number of available pods (ready for at least minReadySeconds) targeted by this experiment.
	AvailableReplicas int32 `json:"availableReplicas"`
	// CollisionCount count of hash collisions for the Experiment. The Experiment controller uses this
	// field as a collision avoidance mechanism when it needs to create the name for the
	// newest ReplicaSet.
	// +optional
	CollisionCount *int32 `json:"collisionCount,omitempty"`
}

// ExperimentStatus is the status for a Experiment resource
type ExperimentStatus struct {
	// TemplateStatuses the hash of the list of environment spec that is used to prevent changes in spec.
	// +optional
	TemplateStatuses []TemplateStatus `json:"templateStatuses,omitempty"`
	// The generation observed by the experiment controller by taking a hash of the spec.
	// +optional
	ObservedGeneration string `json:"observedGeneration,omitempty"`
	// Running indicates if the experiment has started. If the experiment is not running, the controller will
	// scale down all RS. If the running field isn't set, that means that the experiment hasn't started yet.
	// +optional
	Running *bool `json:"running,omitempty"`
	// AvailableAt the time when all the templates become healthy and the experiment should start tracking the time to
	// run for the duration of specificed in the spec.
	// +optional
	AvailableAt *metav1.Time `json:"availableAt,omitempty"`
	// Conditions a list of conditions a experiment can have.
	// +optional
	Conditions []ExperimentCondition `json:"conditions,omitempty"`
}

// ExperimentConditionType defines the conditions of Experiment
type ExperimentConditionType string

// These are valid conditions of a experiment.
const (
	// InvalidExperimentSpec means the experiment has an invalid spec and will not progress until
	// the spec is fixed.
	InvalidExperimentSpec ExperimentConditionType = "InvalidSpec"
	// ExperimentConcluded means the experiment is available, ie. the active service is pointing at a
	// replicaset with the required replicas up and running for at least minReadySeconds.
	ExperimentCompleted ExperimentConditionType = "Completed"
	// ExperimentProgressing means the experiment is progressing. Progress for a experiment is
	// considered when a new replica set is created or adopted, when pods scale
	// up or old pods scale down, or when the services are updated. Progress is not estimated
	// for paused experiment.
	ExperimentProgressing ExperimentConditionType = "Progressing"
	// ExperimentRunning means that an experiment has reached the desired state and is running for the duration
	// specified in the spec
	ExperimentRunning ExperimentConditionType = "Running"
	// ExperimentReplicaFailure ReplicaFailure is added in a experiment when one of its pods
	// fails to be created or deleted.
	ExperimentReplicaFailure ExperimentConditionType = "ReplicaFailure"
)

// ExperimentCondition describes the state of a experiment at a certain point.
type ExperimentCondition struct {
	// Type of deployment condition.
	Type ExperimentConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExperimentList is a list of Experiment resources
type ExperimentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Experiment `json:"items"`
}
