package main

import (
	"slices"
	"testing"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func Test_matchOrder(t *testing.T) {
	type item struct {
		name      string
		identifer string
	}

	testCases := []struct {
		name    string
		items   []item
		toMatch []item
		expect  []item
	}{
		{
			name: "no toMatch",
			items: []item{
				{name: "A"},
				{name: "B"},
			},
			toMatch: nil,
			expect: []item{
				{name: "A"},
				{name: "B"},
			},
		},
		{
			name: "no items",
			items: nil,
			toMatch: []item{
				{name: "A"},
				{name: "B"},
			},
			expect: nil,
		},
		{
			name: "mixed",
			items: []item{
				{name: "A", identifer: "first A in items"},
				{name: "A", identifer: "second A in items"},
				{name: "B", identifer: "first B in items"},
				{name: "C", identifer: "first C in items"},
				{name: "D", identifer: "first D in items"},
				{name: "C", identifer: "second C in items"},
				{name: "C", identifer: "third C in items"},
				{name: "B", identifer: "second B in items"},
			},
			toMatch: []item{
				{name: "A", identifer: "first A in toMatch"},
				{name: "B", identifer: "first B in toMatch"},
				{name: "C", identifer: "first C in toMatch"},
				// Only one A => the second one in items should be the first item at the end
				// No D => the one in items should be the second item at the end
				{name: "B", identifer: "second B in toMatch"},
				// Only two Bs in items, only two should appear in the result
				{name: "B", identifer: "third B in toMatch"},
				// No E in items, shouldn't appear in the result
				{name: "E", identifer: "first E in toMatch"},
				{name: "C", identifer: "second C in toMatch"},
				// Only two Cs => the third one in items should be the last item at the end
			},
			expect: []item{
				{name: "A", identifer: "first A in items"},
				{name: "B", identifer: "first B in items"},
				{name: "C", identifer: "first C in items"},
				{name: "B", identifer: "second B in items"},
				{name: "C", identifer: "second C in items"},
				{name: "A", identifer: "second A in items"},
				{name: "D", identifer: "first D in items"},
				{name: "C", identifer: "third C in items"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchOrder(tc.items, tc.toMatch, func(it item) string { return it.name })
			if !slices.Equal(result, tc.expect) {
				t.Error("expected", tc.expect, "got", result)
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

	t.Run("params, unmatched moved to end", func(t *testing.T) {
		// "D" has no counterpart in existing, so it moves to the end.
		source := pipelineWithParams("C", "D", "A", "B")
		existing := pipelineWithParams("A", "B", "C")

		p := NewPipelineEditor(source, &existing)
		p.ReorderArrays()

		expect := []string{"A", "B", "C", "D"}
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

		// "scan" is only in source, so it moves to the end.
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

		expectTasks := []string{"clone", "build", "scan"}
		if got := taskNames(tasks); !slices.Equal(got, expectTasks) {
			t.Error("tasks: expected", expectTasks, "got", got)
		}

		expectClone := []string{"REVISION", "URL"}
		if got := taskParamNames(tasks[0].Params); !slices.Equal(got, expectClone) {
			t.Error("clone params: expected", expectClone, "got", got)
		}

		expectBuild := []string{"CONTEXT", "DOCKERFILE", "IMAGE"}
		if got := taskParamNames(tasks[1].Params); !slices.Equal(got, expectBuild) {
			t.Error("build params: expected", expectBuild, "got", got)
		}
	})
}
