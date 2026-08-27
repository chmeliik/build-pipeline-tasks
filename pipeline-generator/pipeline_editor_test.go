package main

import (
	"slices"
	"testing"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func Test_matchOrder(t *testing.T) {
	type item struct {
		name       string
		identifier string
	}

	testCases := []struct {
		name          string
		newItems      []item
		existingItems []item
		expect        []item
	}{
		{
			name: "no existingItems",
			newItems: []item{
				{name: "A"},
				{name: "B"},
			},
			existingItems: nil,
			expect: []item{
				{name: "A"},
				{name: "B"},
			},
		},
		{
			name:     "no newItems",
			newItems: nil,
			existingItems: []item{
				{name: "A"},
				{name: "B"},
			},
			expect: nil,
		},
		{
			name:          "no items at all",
			newItems:      nil,
			existingItems: nil,
			expect:        nil,
		},
		{
			name: "disjoint sets keep original order",
			newItems: []item{
				{name: "C", identifier: "C-new"},
				{name: "A", identifier: "A-new"},
				{name: "B", identifier: "B-new"},
			},
			existingItems: []item{
				{name: "X", identifier: "X-existing"},
				{name: "Y", identifier: "Y-existing"},
			},
			expect: []item{
				{name: "C", identifier: "C-new"},
				{name: "A", identifier: "A-new"},
				{name: "B", identifier: "B-new"},
			},
		},
		{
			name: "reorder",
			newItems: []item{
				{name: "C", identifier: "C-new"},
				{name: "A", identifier: "A-new"},
				{name: "B", identifier: "B-new"},
			},
			existingItems: []item{
				{name: "B", identifier: "B-existing"},
				{name: "C", identifier: "C-existing"},
				{name: "A", identifier: "A-existing"},
			},
			expect: []item{
				{name: "B", identifier: "B-new"},
				{name: "C", identifier: "C-new"},
				{name: "A", identifier: "A-new"},
			},
		},
		{
			name: "unique items keep original position",
			newItems: []item{
				{name: "A", identifier: "A-new"},
				{name: "D", identifier: "D-new"},
				{name: "C", identifier: "C-new"},
				{name: "B", identifier: "B-new"},
				{name: "E", identifier: "E-new"},
			},
			existingItems: []item{
				{name: "B", identifier: "B-existing"},
				{name: "D", identifier: "D-existing"},
			},
			expect: []item{
				{name: "A", identifier: "A-new"},
				{name: "B", identifier: "B-new"},
				{name: "C", identifier: "C-new"},
				{name: "D", identifier: "D-new"},
				{name: "E", identifier: "E-new"},
			},
		},
		{
			name: "skip unknown items",
			newItems: []item{
				{name: "C", identifier: "C-new"},
				{name: "B", identifier: "B-new"},
				{name: "A", identifier: "A-new"},
			},
			existingItems: []item{
				{name: "D", identifier: "D-existing"},
				{name: "A", identifier: "A-existing"},
				{name: "E", identifier: "E-existing"},
				{name: "B", identifier: "B-existing"},
				{name: "F", identifier: "F-existing"},
				{name: "C", identifier: "C-existing"},
				{name: "G", identifier: "G-existing"},
			},
			expect: []item{
				{name: "A", identifier: "A-new"},
				{name: "B", identifier: "B-new"},
				{name: "C", identifier: "C-new"},
			},
		},
		{
			name: "duplicates in newItems",
			newItems: []item{
				{name: "A", identifier: "A1-new"},
				{name: "A", identifier: "A2-new"},
				{name: "B", identifier: "B1-new"},
				{name: "A", identifier: "A3-new"},
			},
			existingItems: []item{
				{name: "B", identifier: "B-existing"},
				{name: "A", identifier: "A-existing"},
			},
			expect: []item{
				{name: "B", identifier: "B1-new"},
				// Somewhat curiously, A2 and A1 swap their relative positions.
				// This is in line with what the function is supposed to do
				// (A2 is the unique item, so it gets to keep its position).
				{name: "A", identifier: "A2-new"},
				{name: "A", identifier: "A1-new"},
				{name: "A", identifier: "A3-new"},
			},
		},
		{
			name: "duplicates in existingItems",
			newItems: []item{
				{name: "B", identifier: "B-new"},
				{name: "A", identifier: "A-new"},
			},
			existingItems: []item{
				{name: "A", identifier: "A1-existing"},
				{name: "A", identifier: "A2-existing"},
				{name: "B", identifier: "B1-existing"},
				{name: "A", identifier: "A3-existing"},
			},
			expect: []item{
				{name: "A", identifier: "A-new"},
				{name: "B", identifier: "B-new"},
			},
		},
		{
			name: "mixed duplicates",
			newItems: []item{
				{name: "A", identifier: "A1-new"},
				{name: "A", identifier: "A2-new"},
				{name: "B", identifier: "B1-new"},
				{name: "A", identifier: "A3-new"},
				{name: "B", identifier: "B2-new"},
				{name: "B", identifier: "B3-new"},
				{name: "C", identifier: "C1-new"},
			},
			existingItems: []item{
				{name: "B", identifier: "B1-existing"},
				{name: "A", identifier: "A1-existing"},
				{name: "A", identifier: "A2-existing"},
				{name: "C", identifier: "C1-existing"},
				{name: "C", identifier: "C2-existing"},
				{name: "B", identifier: "B2-existing"},
			},
			expect: []item{
				// A1 A2 B1 => B1 A1 A2 (per existingItems order)
				{name: "B", identifier: "B1-new"},
				{name: "A", identifier: "A1-new"},
				{name: "A", identifier: "A2-new"},
				// A3 keeps position (unique item)
				{name: "A", identifier: "A3-new"},
				// C1 B2 per existingItems order...
				{name: "C", identifier: "C1-new"},
				// But B3 is unique so it keeps its position
				{name: "B", identifier: "B3-new"},
				{name: "B", identifier: "B2-new"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputBefore := slices.Clone(tc.newItems)
			result := matchOrder(tc.newItems, tc.existingItems, func(it item) string { return it.name })
			if !slices.Equal(result, tc.expect) {
				t.Error("expected", tc.expect, "got", result)
			}
			if !slices.Equal(tc.newItems, inputBefore) {
				t.Error("newItems was modified: before", inputBefore, "after", tc.newItems)
			}
		})
	}
}

// pipelineWithParams builds a pipeline whose spec.params have the given names.
func pipelineWithParams(names ...string) tektonapi.Pipeline {
	var params []tektonapi.ParamSpec
	for _, name := range names {
		params = append(params, tektonapi.ParamSpec{Name: name})
	}
	return tektonapi.Pipeline{Spec: tektonapi.PipelineSpec{Params: params}}
}

func paramNames(params []tektonapi.ParamSpec) []string {
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}
	return names
}

func taskParamNames(params []tektonapi.Param) []string {
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}
	return names
}

func taskNames(tasks []tektonapi.PipelineTask) []string {
	names := make([]string, len(tasks))
	for i, t := range tasks {
		names[i] = t.Name
	}
	return names
}

func Test_ReorderArrays(t *testing.T) {
	t.Run("nil existing leaves order unchanged", func(t *testing.T) {
		source := pipelineWithParams("A", "B", "C")

		p := NewPipelineEditor(source, nil)
		p.ReorderArrays()

		expect := []string{"A", "B", "C"}
		if got := paramNames(p.Pipeline.Spec.Params); !slices.Equal(got, expect) {
			t.Error("expected", expect, "got", got)
		}
	})

	t.Run("params, unique retains position", func(t *testing.T) {
		// "D" has no counterpart in existing, so it keeps its position.
		source := pipelineWithParams("C", "D", "A", "B")
		existing := pipelineWithParams("A", "B", "C")

		p := NewPipelineEditor(source, &existing)
		p.ReorderArrays()

		expect := []string{"A", "D", "B", "C"}
		if got := paramNames(p.Pipeline.Spec.Params); !slices.Equal(got, expect) {
			t.Error("expected", expect, "got", got)
		}
	})

	t.Run("tasks and task params", func(t *testing.T) {
		task := func(name string, paramNames ...string) tektonapi.PipelineTask {
			var params []tektonapi.Param
			for _, pn := range paramNames {
				params = append(params, tektonapi.Param{Name: pn})
			}
			return tektonapi.PipelineTask{Name: name, Params: params}
		}

		// "scan" is only in source, so it keeps its position.
		// "deploy" is only in existing, so it is ignored.
		source := tektonapi.Pipeline{Spec: tektonapi.PipelineSpec{
			Tasks: []tektonapi.PipelineTask{
				task("build", "IMAGE", "DOCKERFILE", "CONTEXT"),
				task("scan"),
				task("clone", "URL", "REVISION"),
			},
		}}
		existing := tektonapi.Pipeline{Spec: tektonapi.PipelineSpec{
			Tasks: []tektonapi.PipelineTask{
				task("clone", "REVISION", "URL"),
				task("deploy"),
				task("build", "CONTEXT", "DOCKERFILE", "IMAGE"),
			},
		}}

		p := NewPipelineEditor(source, &existing)
		p.ReorderArrays()

		tasks := p.Pipeline.Spec.Tasks

		expectTasks := []string{"clone", "scan", "build"}
		if got := taskNames(tasks); !slices.Equal(got, expectTasks) {
			t.Error("tasks: expected", expectTasks, "got", got)
		}

		expectClone := []string{"REVISION", "URL"}
		if got := taskParamNames(tasks[0].Params); !slices.Equal(got, expectClone) {
			t.Error("clone params: expected", expectClone, "got", got)
		}

		expectBuild := []string{"CONTEXT", "DOCKERFILE", "IMAGE"}
		if got := taskParamNames(tasks[2].Params); !slices.Equal(got, expectBuild) {
			t.Error("build params: expected", expectBuild, "got", got)
		}
	})
}
