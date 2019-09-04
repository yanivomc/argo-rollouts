# Kustomize Plugin for Argo Rollouts
Kustomize can not natively do StrategicMergePatches with non-native Kubernetes resources like Argo Rollout's Rollout or Experiment (see issue here for more info)[https://github.com/kubernetes-sigs/kustomize/issues/742]. Since the merge patch and JSON 6902 patches are not great, Argo Rollouts needed a better solution. Kustomize can not support StrategicMergePatches as the kustomize client does not have any understanding of the custom resource it is modifying.

Kustomize offers the ability for developers to provide their own plugins in order to get around the limitions of Kustomize.  These plugins can be in the form of a binary that Kustomize executes or a Golang Plugin that gets sideloaded into the Kustomize binary on start. Argo Rollouts decided to use the Exec plugin style since they are easier to bundle with the kustomize binary.

## Installation
`curl mv and chmod`

## Usage
In order to enable Plugins, the user will need to execute `kustomize build <path to dir> -enable_alpha_plugins` and add the Rollout-patch to transformer field of the Rollout.

