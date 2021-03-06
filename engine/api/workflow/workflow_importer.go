package workflow

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Import is able to create a new workflow and all its components
func Import(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, u *sdk.User, force bool, msgChan chan<- sdk.Message) error {
	w.ProjectKey = proj.Key
	w.ProjectID = proj.ID

	mError := new(sdk.MultiError)

	var pipelineLoader = func(n *sdk.WorkflowNode) {
		pip, err := pipeline.LoadPipeline(db, proj.Key, n.Pipeline.Name, false)
		if err != nil {
			log.Warning("workflow.Import> %s > Pipeline %s not found", w.Name, n.Pipeline.Name)
			mError.Append(sdk.NewError(sdk.ErrPipelineNotFound, fmt.Errorf("pipeline %s/%s not found", proj.Key, n.Pipeline.Name)))
			return
		}
		n.Pipeline = *pip
	}
	w.Visit(pipelineLoader)

	var applicationLoader = func(n *sdk.WorkflowNode) {
		if n.Context == nil || n.Context.Application == nil || n.Context.Application.Name == "" {
			return
		}
		app, err := application.LoadByName(db, store, proj.Key, n.Context.Application.Name, u)
		if err != nil {
			log.Warning("workflow.Import> %s > Application %s not found", w.Name, n.Context.Application.Name)
			mError.Append(sdk.NewError(sdk.ErrApplicationNotFound, fmt.Errorf("application %s/%s not found", proj.Key, n.Context.Application.Name)))
			return
		}
		n.Context.Application = app
	}
	w.Visit(applicationLoader)

	var envLoader = func(n *sdk.WorkflowNode) {
		if n.Context == nil || n.Context.Environment == nil || n.Context.Environment.Name == "" {
			return
		}
		env, err := environment.LoadEnvironmentByName(db, proj.Key, n.Context.Environment.Name)
		if err != nil {
			log.Warning("workflow.Import> %s > Environment %s not found", w.Name, n.Context.Environment.Name)
			mError.Append(sdk.NewError(sdk.ErrNoEnvironment, fmt.Errorf("environment %s/%s not found", proj.Key, n.Context.Environment.Name)))
			return
		}
		n.Context.Environment = env
	}
	w.Visit(envLoader)

	if !mError.IsEmpty() {
		return mError
	}

	doUpdate, errE := Exists(db, proj.Key, w.Name)
	if errE != nil {
		return sdk.WrapError(errE, "Import")
	}

	if !doUpdate {
		if err := Insert(db, store, w, proj, u); err != nil {
			return sdk.WrapError(err, "Import> Unable to insert workflow")
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedInserted, w.Name)
		}
		return nil
	}

	if !force {
		return sdk.NewError(sdk.ErrConflict, fmt.Errorf("Workflow exists"))
	}

	oldW, errO := Load(db, store, proj.Key, w.Name, u, LoadOptions{})
	if errO != nil {
		return sdk.WrapError(errO, "Import> Unable to load old workflow")
	}

	w.ID = oldW.ID
	if err := Update(db, store, w, oldW, proj, u); err != nil {
		return sdk.WrapError(err, "Import> Unable to update workflow")
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedUpdated, w.Name)
	}
	return nil

}
