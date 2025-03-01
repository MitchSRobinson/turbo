// Package graph contains the CompleteGraph struct and some methods around it
package graph

import (
	gocontext "context"
	"fmt"

	"github.com/pyr-sh/dag"
	"github.com/vercel/turbo/cli/internal/fs"
	"github.com/vercel/turbo/cli/internal/nodes"
	"github.com/vercel/turbo/cli/internal/util"
)

// WorkspaceInfos holds information about each workspace in the monorepo.
type WorkspaceInfos map[string]*fs.PackageJSON

// CompleteGraph represents the common state inferred from the filesystem and pipeline.
// It is not intended to include information specific to a particular run.
type CompleteGraph struct {
	// WorkspaceGraph expresses the dependencies between packages
	WorkspaceGraph dag.AcyclicGraph

	// Pipeline is config from turbo.json
	Pipeline fs.Pipeline

	// WorkspaceInfos stores the package.json contents by package name
	WorkspaceInfos WorkspaceInfos

	// GlobalHash is the hash of all global dependencies
	GlobalHash string

	RootNode string
}

// GetPackageTaskVisitor wraps a `visitor` function that is used for walking the TaskGraph
// during execution (or dry-runs). The function returned here does not execute any tasks itself,
// but it helps curry some data from the Complete Graph and pass it into the visitor function.
func (g *CompleteGraph) GetPackageTaskVisitor(ctx gocontext.Context, visitor func(ctx gocontext.Context, packageTask *nodes.PackageTask) error) func(taskID string) error {
	return func(taskID string) error {
		packageName, taskName := util.GetPackageTaskFromId(taskID)

		pkg, ok := g.WorkspaceInfos[packageName]
		if !ok {
			return fmt.Errorf("cannot find package %v for task %v", packageName, taskID)
		}

		packageTask := &nodes.PackageTask{
			TaskID:      taskID,
			Task:        taskName,
			PackageName: packageName,
			Pkg:         pkg,
		}

		taskDefinition, err := g.getResolvedTaskDefinition(taskID, taskName)
		if err != nil {
			return err
		}

		packageTask.TaskDefinition = taskDefinition

		return visitor(ctx, packageTask)
	}
}

// getResolvedTaskDefinition currently just looks for the definition in the Pipeline
// defined in the Graph. Later, this will get Pipelines defined in the task's workspace as well.
func (g *CompleteGraph) getResolvedTaskDefinition(taskID string, taskName string) (*fs.TaskDefinition, error) {
	return getTaskFromPipeline(g.Pipeline, taskID, taskName)
}

func getTaskFromPipeline(pipeline fs.Pipeline, taskID string, taskName string) (*fs.TaskDefinition, error) {
	// first check for package-tasks
	taskDefinition, ok := pipeline[taskID]
	if !ok {
		// then check for regular tasks
		fallbackTaskDefinition, notcool := pipeline[taskName]
		// if neither, then bail
		if !notcool {
			// Return an empty fs.TaskDefinition
			return nil, fmt.Errorf("No task defined in pipeline")
		}

		// override if we need to...
		taskDefinition = fallbackTaskDefinition
	}

	return &taskDefinition, nil
}
