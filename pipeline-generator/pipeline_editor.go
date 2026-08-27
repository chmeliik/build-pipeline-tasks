package main

import (
	"log"
	"slices"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

type PipelineEditor struct {
	Pipeline tektonapi.Pipeline
	existing *tektonapi.Pipeline
}

// NewPipelineEditor returns a helper that edits the source pipeline
// while taking into account the existing target pipeline, if any.
// The source pipeline is deep-copied so edits never mutate the caller's copy.
func NewPipelineEditor(pipeline tektonapi.Pipeline, existing *tektonapi.Pipeline) PipelineEditor {
	return PipelineEditor{*pipeline.DeepCopy(), existing}
}

// StringValue returns a string-type tekton param.
func StringValue(val string) *tektonapi.ParamValue {
	return &tektonapi.ParamValue{Type: tektonapi.ParamTypeString, StringVal: val}
}

// ArrayValue returns an array-type tekton param.
func ArrayValue(vals ...string) *tektonapi.ParamValue {
	return &tektonapi.ParamValue{Type: tektonapi.ParamTypeArray, ArrayVal: vals}
}

// RemovePipelineWorkspace removes a pipeline-level workspace declaration
// and strips every task binding that references it.
func (p *PipelineEditor) RemovePipelineWorkspace(name string) {
	spec := &p.Pipeline.Spec

	prevLength := len(spec.Workspaces)
	spec.Workspaces = slices.DeleteFunc(spec.Workspaces,
		func(ws tektonapi.PipelineWorkspaceDeclaration) bool { return ws.Name == name })

	if prevLength == len(spec.Workspaces) {
		log.Printf("WARNING: no pipeline workspace named %q in spec.workspaces\n", name)
	}

	for i := range spec.Tasks {
		task := &spec.Tasks[i]
		task.Workspaces = slices.DeleteFunc(task.Workspaces,
			func(ws tektonapi.WorkspacePipelineTaskBinding) bool { return ws.Workspace == name })
	}
}

// SetParamDefault sets the default value of an existing pipeline param.
func (p *PipelineEditor) SetParamDefault(paramName string, def string) {
	for i := range p.Pipeline.Spec.Params {
		if p.Pipeline.Spec.Params[i].Name == paramName {
			p.Pipeline.Spec.Params[i].Default = StringValue(def)
			return
		}
	}
	log.Printf("WARNING: no pipeline param named %q in spec.params\n", paramName)
}

// RemoveParam removes a pipeline param by name.
func (p *PipelineEditor) RemoveParam(paramName string) {
	spec := &p.Pipeline.Spec

	prevLength := len(spec.Params)
	spec.Params = slices.DeleteFunc(spec.Params,
		func(param tektonapi.ParamSpec) bool { return param.Name == paramName })

	if prevLength == len(spec.Params) {
		log.Printf("WARNING: no pipeline param named %q in spec.params\n", paramName)
	}
}

// RemoveTask removes a task by its in-pipeline name.
func (p *PipelineEditor) RemoveTask(pipelineTaskName string) {
	spec := &p.Pipeline.Spec

	prevLength := len(spec.Tasks)
	spec.Tasks = slices.DeleteFunc(spec.Tasks,
		func(task tektonapi.PipelineTask) bool { return task.Name == pipelineTaskName })

	if prevLength == len(spec.Tasks) {
		log.Printf("WARNING: no pipeline task named %q in spec.tasks\n", pipelineTaskName)
	}
}

// RenameTask changes the in-pipeline name of a task.
func (p *PipelineEditor) RenameTask(oldName string, newName string) {
	if task := p.findTask(oldName); task != nil {
		task.Name = newName
	}
}

// SetTaskRef sets the taskRef for a task. Specifically:
// - "name" param => taskName
// - "bundle" param => defaultBundle or the existing bundle, if the target pipeline already has one
func (p *PipelineEditor) SetTaskRef(pipelineTaskName string, taskName string, defaultBundle string) {
	pipelineTask := p.findTask(pipelineTaskName)
	if pipelineTask == nil {
		return
	}

	bundle := defaultBundle
	if v := p.getExistingBundleRef(taskName); v != "" {
		bundle = v
	}

	pipelineTask.TaskRef = &tektonapi.TaskRef{ResolverRef: tektonapi.ResolverRef{
		Resolver: "bundles",
		Params: []tektonapi.Param{
			{Name: "name", Value: *StringValue(taskName)},
			{Name: "bundle", Value: *StringValue(bundle)},
			{Name: "kind", Value: *StringValue("task")},
		},
	}}
}

// AddTaskParam appends a param to a task.
func (p *PipelineEditor) AddTaskParam(pipelineTaskName string, paramName string, value *tektonapi.ParamValue) {
	if task := p.findTask(pipelineTaskName); task != nil {
		task.Params = append(task.Params, tektonapi.Param{Name: paramName, Value: *value})
	}
}

// SetTaskParam replaces the value of an existing task param.
func (p *PipelineEditor) SetTaskParam(pipelineTaskName string, paramName string, value *tektonapi.ParamValue) {
	task := p.findTask(pipelineTaskName)
	if task == nil {
		return
	}
	for i := range task.Params {
		if task.Params[i].Name == paramName {
			task.Params[i].Value = *value
			return
		}
	}
	log.Printf("WARNING: no param named %q in task %q\n", paramName, pipelineTaskName)
}

// SetTaskRunAfter replaces a task's runAfter list.
func (p *PipelineEditor) SetTaskRunAfter(pipelineTaskName string, runAfter []string) {
	if task := p.findTask(pipelineTaskName); task != nil {
		task.RunAfter = runAfter
	}
}

// AddTaskMatrixParam appends a matrix param to a task, creating the matrix if necessary.
func (p *PipelineEditor) AddTaskMatrixParam(pipelineTaskName string, paramName string, value *tektonapi.ParamValue) {
	task := p.findTask(pipelineTaskName)
	if task == nil {
		return
	}
	if task.Matrix == nil {
		task.Matrix = &tektonapi.Matrix{}
	}
	task.Matrix.Params = append(task.Matrix.Params, tektonapi.Param{Name: paramName, Value: *value})
}

// ReorderArrays reorders relevant arrays in p.Pipeline
// to better match the corresponding arrays in p.existing, if any.
//
// The "relevant" arrays are the ones that we expect to be edited by migration scripts.
func (p *PipelineEditor) ReorderArrays() {
	if p.existing == nil {
		return
	}

	reorder(p.Pipeline.Spec.Params, p.existing.Spec.Params,
		func(p tektonapi.ParamSpec) string { return p.Name })

	reorder(p.Pipeline.Spec.Results, p.existing.Spec.Results,
		func(r tektonapi.PipelineResult) string { return r.Name })

	reorder(p.Pipeline.Spec.Tasks, p.existing.Spec.Tasks,
		func(t tektonapi.PipelineTask) string { return t.Name })

	existingTasks := make(map[string]*tektonapi.PipelineTask)
	for _, et := range p.existing.Spec.Tasks {
		existingTasks[et.Name] = &et
	}

	for _, task := range p.Pipeline.Spec.Tasks {
		exTask := existingTasks[task.Name]
		if exTask == nil {
			continue
		}

		reorder(task.Params, exTask.Params,
			func(p tektonapi.Param) string { return p.Name })

		if task.Matrix != nil && exTask.Matrix != nil {
			reorder(task.Matrix.Params, exTask.Matrix.Params,
				func(p tektonapi.Param) string { return p.Name })
		}

		reorder(task.When, exTask.When,
			func(w tektonapi.WhenExpression) string { return w.Input })

		reorder(task.RunAfter, exTask.RunAfter, func(s string) string { return s })
	}
}

func (p *PipelineEditor) findTask(pipelineTaskName string) *tektonapi.PipelineTask {
	for i := range p.Pipeline.Spec.Tasks {
		pt := &p.Pipeline.Spec.Tasks[i]
		if pt.Name == pipelineTaskName {
			return pt
		}
	}

	log.Printf("WARNING: no pipeline task named %q in spec.tasks\n", pipelineTaskName)
	return nil
}

func (p *PipelineEditor) getExistingBundleRef(taskName string) string {
	if p.existing == nil {
		return ""
	}

	for _, pipelineTask := range p.existing.Spec.Tasks {
		ref := pipelineTask.TaskRef
		if ref == nil {
			continue
		}

		var name string
		var bundle string

		for _, param := range ref.Params {
			switch param.Name {
			case "name":
				name = param.Value.StringVal
			case "bundle":
				bundle = param.Value.StringVal
			}
		}

		if name == taskName {
			return bundle
		}
	}

	return ""
}

// reorder reorders items to match the order in toMatch.
// Items that do not have a counterpart in toMatch move to the end.
func reorder[T any, K comparable](items []T, toMatch []T, key func(T) K) {
	if len(items) > 0 && len(toMatch) > 0 {
		copy(items, matchOrder(items, toMatch, key))
	}
}

// matchOrder implements reorder, but returns a new slice instead of modifying the input.
func matchOrder[T any, K comparable](items []T, toMatch []T, key func(T) K) []T {
	itemCounts := make(map[K]int)
	for _, item := range items {
		itemCounts[key(item)]++
	}

	matchIndices := make(map[K][]int)
	// matchIndex is the index of the next item in toMatch that also appears in items.
	// (More accurately: an item whose key is the same as the key of an item in items,
	//  and whose key hasn't been seen more times than the number of its occurrences in items.)
	// The index is counted as if toMatch didn't contain any non-matching items.
	// If there are no more matching items, matchIndex equals the size of the intersection.
	matchIndex := 0
	for _, item := range toMatch {
		k := key(item)
		if itemCounts[k] > 0 {
			matchIndices[k] = append(matchIndices[k], matchIndex)
			matchIndex++
			itemCounts[k]--
		}
	}

	reordered := make([]T, len(items))
	for _, item := range items {
		k := key(item)
		if indices := matchIndices[k]; len(indices) > 0 {
			i := indices[0]
			matchIndices[k] = indices[1:]
			reordered[i] = item
		} else {
			// matchIndex is the size of the intersection (or was, before we incremented it),
			// so the section for unique items starts at matchIndex
			reordered[matchIndex] = item
			matchIndex++
		}
	}

	return reordered
}
