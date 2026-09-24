package main

import (
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const dockerBuildMinDescription = `This pipeline is ideal for building demo container images from a Containerfile while maintaining trust after pipeline customization.

This version of pipeline has minimal resource requests set, it's good for demonstrating and testing.

_Uses ` + "`buildah`" + ` to create a container image leveraging [trusted artifacts](https://konflux-ci.dev/architecture/ADR/0036-trusted-artifacts.html). It also optionally creates a source image and runs some build-time tests. Information is shared between tasks using OCI artifacts instead of PVCs. EC will pass the [` + "`trusted_task.trusted`" + `](https://conforma.dev/docs/policy/packages/release_trusted_task.html#trusted_task__trusted) policy as long as all data used to build the artifact is generated from trusted tasks.
This pipeline is pushed as a Tekton bundle to [quay.io](https://quay.io/repository/konflux-ci/tekton-catalog/pipeline-docker-build-oci-ta?tab=tags)_
`

func GenerateDockerBuildMin(dockerBuildOciTa tektonapi.Pipeline, existing *tektonapi.Pipeline) (tektonapi.Pipeline, error) {
	p := NewPipelineEditor(dockerBuildOciTa, existing)

	p.Pipeline.Spec.Description = dockerBuildMinDescription
	p.Pipeline.Name = "docker-build-oci-ta-min"
	p.Pipeline.Labels = map[string]string{
		"pipelines.openshift.io/used-by":  "build-cloud",
		"pipelines.openshift.io/runtime":  "generic",
		"pipelines.openshift.io/strategy": "docker",
	}

	// Swap the resource-heavy tasks for their minimal variants.
	p.SetTaskRef(
		"clone-repository",
		"git-clone-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-git-clone-oci-ta-min:0.2.6@sha256:5a4bed5a11cf834f67d237104085bc83ea51d2dc51af974137fb61f11fc2e7e3",
	)
	p.SetTaskRef(
		"prefetch-dependencies",
		"prefetch-dependencies-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-prefetch-dependencies-oci-ta-min:0.10.3@sha256:b21f1fe2f549ed78914ad5369b055fcbf235f966e2bab873b34c469a4d14e8b7",
	)
	p.SetTaskRef(
		"build-container",
		"buildah-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-buildah-oci-ta-min:0.12.2@sha256:4cd3ffd311a2b3a198a8beffc2aa732e325b66f045f98591cf8a3ddf9d511f0e",
	)
	p.SetTaskRef(
		"build-image-index",
		"build-image-index-min",
		"quay.io/konflux-ci/tekton-catalog/task-build-image-index-min:0.3.1@sha256:ee8a340c2b82ab4bafdf9125a44d8dd87fe47fccc9599ba6db1695bdc919f56a",
	)
	p.SetTaskRef(
		"clamav-scan",
		"clamav-scan-min",
		"quay.io/konflux-ci/tekton-catalog/task-clamav-scan-min:0.3@sha256:ec3afb0a8726612b8f4450c7dd6511a2ba9594ed8b3851f9e3be50544e92c640",
	)
	p.SetTaskRef(
		"sast-shell-check",
		"sast-shell-check-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-sast-shell-check-oci-ta-min:0.1@sha256:25716c9e82387c5606f3979dbea501a4fed033c435e4bd81ecbd9116690da551",
	)
	p.SetTaskRef(
		"sast-unicode-check",
		"sast-unicode-check-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-sast-unicode-check-oci-ta-min:0.4@sha256:f0bd053a37fecf56586b2426ec9d3bfc9ddaa8960c70c277ea897258ab4e7ce9",
	)

	// Add the TPA scan task.
	// Append without a taskRef, then SetTaskRef so its bundle is renovate-preserved like the others.
	p.Pipeline.Spec.Tasks = append(p.Pipeline.Spec.Tasks, tektonapi.PipelineTask{
		Name: "tpa-scan",
		Params: tektonapi.Params{
			{Name: "image-digest", Value: *StringValue("$(tasks.build-image-index.results.IMAGE_DIGEST)")},
			{Name: "image-url", Value: *StringValue("$(tasks.build-image-index.results.IMAGE_URL)")},
		},
		RunAfter: []string{"build-image-index"},
		When: tektonapi.WhenExpressions{
			{Input: "$(params.skip-checks)", Operator: "in", Values: []string{"false"}},
		},
	})
	p.SetTaskRef(
		"tpa-scan",
		"tpa-scan",
		"quay.io/konflux-ci/tekton-catalog/task-tpa-scan:0.1@sha256:c15dbb2faf110947dbc2644a6a289cd1cbb858bddf212486a7deac65a1506f87",
	)

	// Drop the checks and steps not wanted in the minimal pipeline.
	p.RemoveTask("push-dockerfile")
	p.RemoveTask("apply-tags")
	p.RemoveTask("sast-snyk-check")
	p.RemoveTask("ecosystem-cert-preflight-checks")
	p.RemoveTask("clair-scan")
	p.RemoveTask("build-source-image")

	// build-source-image param is only used by the removed build-source-image task.
	p.RemoveParam("build-source-image")

	p.ReorderArrays()
	return p.Pipeline, nil
}
