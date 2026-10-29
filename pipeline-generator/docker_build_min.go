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
		"quay.io/konflux-ci/tekton-catalog/task-git-clone-oci-ta-min:0.2.6@sha256:953ca2e294946f6e100550169000eb0893f6a025dc8778bdf837cbbea476a4f7",
	)
	p.SetTaskRef(
		"prefetch-dependencies",
		"prefetch-dependencies-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-prefetch-dependencies-oci-ta-min:0.11.0@sha256:e2a857029641c89d990ff2dbfe4b9c01c23c0caa4e9ca99a6dedb27e08edf6c9",
	)
	p.SetTaskRef(
		"build-container",
		"buildah-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-buildah-oci-ta-min:0.14.1@sha256:34adff1c6c28c7ebd8057cf1d6ed2fd16177175a6a04923b651f50d97fefeaa2",
	)
	p.SetTaskRef(
		"build-image-index",
		"build-image-index-min",
		"quay.io/konflux-ci/tekton-catalog/task-build-image-index-min:0.3.1@sha256:ee8a340c2b82ab4bafdf9125a44d8dd87fe47fccc9599ba6db1695bdc919f56a",
	)
	p.SetTaskRef(
		"clamav-scan",
		"clamav-scan-min",
		"quay.io/konflux-ci/tekton-catalog/task-clamav-scan-min:0.3@sha256:c5f39a64fe5d4a5fd439c349979a6737f0f9870a9a843d70e021428f28b2ecf0",
	)
	p.SetTaskRef(
		"sast-shell-check",
		"sast-shell-check-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-sast-shell-check-oci-ta-min:0.1@sha256:e48587ffa34224ab113ea62f1237cf3c5f80d97a7ac732ad354f417d17ef9099",
	)
	p.SetTaskRef(
		"sast-unicode-check",
		"sast-unicode-check-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-sast-unicode-check-oci-ta-min:0.4@sha256:f4c3381a2e86c7e2c46f3556ddf3a829086582c438ecfb78100e37e68395a8a3",
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
		"quay.io/konflux-ci/tekton-catalog/task-tpa-scan:0.1@sha256:9f75fae64eb9c334da6823c004d2339a826cabe191fdc1ad1eff64d9b2e53f38",
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
