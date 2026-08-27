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

// reorder reorders newItems to match the order in existingItems.
// Items that do not have a counterpart in existingItems keep their original position
// (as if "inserted into" the order defined by existingItems).
func reorder[T any, K comparable](newItems []T, existingItems []T, key func(T) K) {
	if len(newItems) > 0 && len(existingItems) > 0 {
		copy(newItems, matchOrder(newItems, existingItems, key))
	}
}

// matchOrder implements reorder, but returns a new slice instead of modifying the input.
func matchOrder[T any, K comparable](newItems []T, existingItems []T, key func(T) K) []T {
	newKeys := mapfn(key, newItems)
	existingKeys := mapfn(key, existingItems)

	uniqueExisting := multisetDifference(existingKeys, newKeys)
	uniqueNew := multisetDifference(newKeys, existingKeys)

	// matchIndices tracks where items of a given key should go in the reordered array.
	// The i-th occurence (0-based) of an item with key k should go to matchIndices[k][i].
	matchIndices := make(map[K][]int)
	matchIndex := 0
	for i, k := range existingKeys {
		if slices.Contains(uniqueExisting, i) {
			continue
		}
		for slices.Contains(uniqueNew, matchIndex) {
			// Find the next index not occupied by a unique new item (those keep their position)
			// Happens at most len(uniqueNew) times
			matchIndex++
		}
		// Happens exactly len(intersection of new and existing) times
		matchIndices[k] = append(matchIndices[k], matchIndex)
		matchIndex++
	}

	reordered := make([]T, len(newItems))
	for i, k := range newKeys {
		if indices := matchIndices[k]; len(indices) > 0 {
			// Item in the intersection, move to matching index
			j := indices[0]
			matchIndices[k] = indices[1:]
			reordered[j] = newItems[i]
		} else {
			// Unique item, keep original position
			reordered[i] = newItems[i]
		}
	}
	return reordered
}

func mapfn[A any, B any](fn func(A) B, items []A) []B {
	itemsB := make([]B, len(items))
	for i, item := range items {
		itemsB[i] = fn(item)
	}
	return itemsB
}

// Computes the multiset difference a \ b, returns indices of unique elements in a.
// Example:
// - a = {'a', 'b', 'c', 'a', 'b'}
// - b = {'a', 'b', 'a'}
// - multisetDifference(a, b) = {2, 4} (the 'c' element and the second 'b' element)
func multisetDifference[T comparable](a []T, b []T) []int {
	bCount := make(map[T]int)
	for _, item := range b {
		bCount[item]++
	}

	var unique []int
	for i, item := range a {
		if bCount[item] == 0 {
			unique = append(unique, i)
		} else {
			bCount[item]--
		}
	}
	return unique
}
