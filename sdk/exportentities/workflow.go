package exportentities

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

type Workflow struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// This will be filled for complex workflows
	Workflow map[string]NodeEntry   `json:"workflow,omitempty" yaml:"workflow,omitempty"`
	Hooks    map[string][]HookEntry `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	// This will be filled for simple workflows
	DependsOn       []string                    `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions      *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	When            []string                    `json:"when,omitempty" yaml:"when,omitempty"` //This is use only for manual and success condition
	PipelineName    string                      `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	ApplicationName string                      `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName string                      `json:"environment,omitempty" yaml:"environment,omitempty"`
	PipelineHooks   []HookEntry                 `json:"pipeline_hooks,omitempty" yaml:"pipeline_hooks,omitempty"`
	Permissions     map[string]int              `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

type NodeEntry struct {
	DependsOn       []string                    `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions      *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	When            []string                    `json:"when,omitempty" yaml:"when,omitempty"` //This is use only for manual and success condition
	PipelineName    string                      `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	ApplicationName string                      `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName string                      `json:"environment,omitempty" yaml:"environment,omitempty"`
	OneAtATime      *bool                       `json:"one_at_a_time,omitempty" yaml:"one_at_a_time,omitempty"`
}

type HookEntry struct {
	Model  string            `json:"type,omitempty" yaml:"type,omitempty"`
	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

type WorkflowVersion string

const WorkflowVersion1 = "v1.0"

//NewWorkflow creates a new exportable workflow
func NewWorkflow(w sdk.Workflow, withPermission bool) (Workflow, error) {
	exportedWorkflow := Workflow{}
	exportedWorkflow.Name = w.Name
	exportedWorkflow.Version = WorkflowVersion1
	exportedWorkflow.Workflow = map[string]NodeEntry{}
	exportedWorkflow.Hooks = map[string][]HookEntry{}
	nodes := w.Nodes(false)

	if withPermission {
		exportedWorkflow.Permissions = make(map[string]int, len(w.Groups))
		for _, p := range w.Groups {
			exportedWorkflow.Permissions[p.Group.Name] = p.Permission
		}
	}

	var craftNodeEntry = func(n *sdk.WorkflowNode) (NodeEntry, error) {
		entry := NodeEntry{}

		ancestorIDs := n.Ancestors(&w, false)
		ancestors := []string{}
		for _, aID := range ancestorIDs {
			a := w.GetNode(aID)
			if a == nil {
				return entry, sdk.ErrWorkflowNodeNotFound
			}
			ancestors = append(ancestors, a.Name)
		}

		entry.DependsOn = ancestors
		entry.PipelineName = n.Pipeline.Name
		conditions := []sdk.WorkflowNodeCondition{}
		for _, c := range n.Context.Conditions.PlainConditions {
			if c.Operator == sdk.WorkflowConditionsOperatorEquals &&
				c.Value == "Success" &&
				c.Variable == "cds.status" {
				entry.When = append(entry.When, "success")
			} else if c.Operator == sdk.WorkflowConditionsOperatorEquals &&
				c.Value == "true" &&
				c.Variable == "cds.manual" {
				entry.When = append(entry.When, "manual")
			} else {
				conditions = append(conditions, c)
			}
		}

		if len(conditions) > 0 || n.Context.Conditions.LuaScript != "" {
			entry.Conditions = &sdk.WorkflowNodeConditions{
				PlainConditions: conditions,
				LuaScript:       n.Context.Conditions.LuaScript,
			}
		}

		if n.Context.Application != nil {
			entry.ApplicationName = n.Context.Application.Name
		}
		if n.Context.Environment != nil {
			entry.EnvironmentName = n.Context.Environment.Name
		}

		if n.Context.Mutex {
			entry.OneAtATime = &n.Context.Mutex
		}

		return entry, nil
	}

	hooks := w.GetHooks()

	if len(nodes) == 0 {
		n := w.Root
		if n == nil {
			return exportedWorkflow, sdk.ErrWorkflowNodeNotFound
		}
		entry, err := craftNodeEntry(n)
		if err != nil {
			return exportedWorkflow, err
		}
		exportedWorkflow.ApplicationName = entry.ApplicationName
		exportedWorkflow.PipelineName = entry.PipelineName
		exportedWorkflow.EnvironmentName = entry.EnvironmentName
		exportedWorkflow.DependsOn = entry.DependsOn
		if entry.Conditions != nil && (len(entry.Conditions.PlainConditions) > 0 || entry.Conditions.LuaScript != "") {
			exportedWorkflow.When = entry.When
			exportedWorkflow.Conditions = entry.Conditions
		}
		for _, h := range hooks {
			if exportedWorkflow.Hooks == nil {
				exportedWorkflow.Hooks = make(map[string][]HookEntry)
			}
			exportedWorkflow.PipelineHooks = append(exportedWorkflow.PipelineHooks, HookEntry{
				Model:  h.WorkflowHookModel.Name,
				Config: h.Config.Values(),
			})
		}
	} else {
		nodes = append(nodes, *w.Root)
		for i := range nodes {
			n := &nodes[i]
			if n == nil {
				return exportedWorkflow, sdk.ErrWorkflowNodeNotFound
			}
			entry, err := craftNodeEntry(n)
			if err != nil {
				return exportedWorkflow, err
			}
			exportedWorkflow.Workflow[n.Name] = entry

		}

		for _, h := range hooks {
			if exportedWorkflow.Hooks == nil {
				exportedWorkflow.Hooks = make(map[string][]HookEntry)
			}
			exportedWorkflow.Hooks[w.GetNode(h.WorkflowNodeID).Name] = append(exportedWorkflow.Hooks[w.GetNode(h.WorkflowNodeID).Name], HookEntry{
				Model:  h.WorkflowHookModel.Name,
				Config: h.Config.Values(),
			})
		}
	}

	return exportedWorkflow, nil
}

// Entries returns the map of all workflow entries
func (w Workflow) Entries() map[string]NodeEntry {
	if len(w.Workflow) != 0 {
		return w.Workflow
	}

	singleEntry := NodeEntry{
		ApplicationName: w.ApplicationName,
		EnvironmentName: w.EnvironmentName,
		PipelineName:    w.PipelineName,
		Conditions:      w.Conditions,
		DependsOn:       w.DependsOn,
		When:            w.When,
	}
	return map[string]NodeEntry{
		w.PipelineName: singleEntry,
	}

}

func (e NodeEntry) checkValidity() error {
	return nil
}

func (w Workflow) checkValidity() error {
	mError := new(sdk.MultiError)

	if len(w.Workflow) != 0 {
		if w.ApplicationName != "" {
			mError.Append(fmt.Errorf("Error: wrong usage: application %s not allowed here", w.ApplicationName))
		}
		if w.EnvironmentName != "" {
			mError.Append(fmt.Errorf("Error: wrong usage: environment %s not allowed here", w.EnvironmentName))
		}
		if w.PipelineName != "" {
			mError.Append(fmt.Errorf("Error: wrong usage: pipeline %s not allowed here", w.PipelineName))
		}
		if w.Conditions != nil {
			mError.Append(fmt.Errorf("Error: wrong usage: conditions not allowed here"))
		}
		if len(w.When) != 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: when not allowed here"))
		}
		if len(w.DependsOn) != 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: depends_on not allowed here"))
		}
		if len(w.PipelineHooks) != 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: pipeline_hooks not allowed here"))
		}
	} else {
		if len(w.Hooks) > 0 {
			mError.Append(fmt.Errorf("Error: wrong usage: hooks not allowed here"))
		}
	}

	for name := range w.Hooks {
		if _, ok := w.Workflow[name]; !ok {
			mError.Append(fmt.Errorf("Error: wrong usage: invalid hook on %s", name))
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func (w Workflow) checkDependencies() error {
	mError := new(sdk.MultiError)
	for s, e := range w.Entries() {
		if err := e.checkDependencies(w); err != nil {
			mError.Append(fmt.Errorf("Error: %s invalid: %v", s, err))
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func (e NodeEntry) checkDependencies(w Workflow) error {
	mError := new(sdk.MultiError)
nextDep:
	for _, d := range e.DependsOn {
		for s := range w.Workflow {
			if s == d {
				continue nextDep
			}
		}
		mError.Append(fmt.Errorf("%s not found", d))
	}
	if mError.IsEmpty() {
		return nil
	}
	return mError
}

//GetWorkflow returns a fresh sdk.Workflow
func (w Workflow) GetWorkflow() (*sdk.Workflow, error) {
	var wf = new(sdk.Workflow)
	wf.Name = w.Name
	if err := w.checkValidity(); err != nil {
		return nil, err
	}
	if err := w.checkDependencies(); err != nil {
		return nil, err
	}

	entries := w.Entries()
	var attempt int
	// attempt is there to avoid infinit loop, but it should not happend becase we check validty and dependencies earlier
	for len(entries) != 0 && attempt < 1000 {
		for name, entry := range entries {
			ok, err := entry.processNode(name, wf)
			if err != nil {
				return nil, err
			}
			if ok {
				delete(entries, name)
			}
		}
		attempt++
	}
	if len(entries) > 0 {
		return nil, fmt.Errorf("Unable to process %+v", entries)
	}

	//Process hooks
	wf.Visit(w.processHooks)

	return wf, nil
}

func (e *NodeEntry) getNode(name string) (*sdk.WorkflowNode, error) {
	node := &sdk.WorkflowNode{
		Name: name,
		Ref:  name,
		Pipeline: sdk.Pipeline{
			Name: e.PipelineName,
		},
	}

	if e.ApplicationName != "" {
		node.Context = new(sdk.WorkflowNodeContext)
		node.Context.Application = &sdk.Application{
			Name: e.ApplicationName,
		}
	}

	if e.EnvironmentName != "" {
		if node.Context == nil {
			node.Context = new(sdk.WorkflowNodeContext)
		}

		node.Context.Environment = &sdk.Environment{
			Name: e.EnvironmentName,
		}
	}

	if e.Conditions != nil {
		if node.Context == nil {
			node.Context = new(sdk.WorkflowNodeContext)
		}
		node.Context.Conditions = *e.Conditions
	}

	for _, w := range e.When {
		if node.Context == nil {
			node.Context = new(sdk.WorkflowNodeContext)
		}

		switch w {
		case "success":
			node.Context.Conditions.PlainConditions = append(node.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
				Operator: sdk.WorkflowConditionsOperatorEquals,
				Value:    "Success",
				Variable: "cds.status",
			})
		case "manual":
			node.Context.Conditions.PlainConditions = append(node.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
				Operator: sdk.WorkflowConditionsOperatorEquals,
				Value:    "true",
				Variable: "cds.manual",
			})
		default:
			return nil, fmt.Errorf("Unsupported when condition %s", w)
		}
	}

	if e.OneAtATime != nil {
		if node.Context == nil {
			node.Context = &sdk.WorkflowNodeContext{}
		}
		node.Context.Mutex = *e.OneAtATime
	}

	return node, nil
}

func (w *Workflow) processHooks(n *sdk.WorkflowNode) {
	var addHooks = func(hooks []HookEntry) {
		for _, h := range hooks {
			cfg := make(sdk.WorkflowNodeHookConfig, len(h.Config))
			for k, v := range h.Config {
				cfg[k] = sdk.WorkflowNodeHookConfigValue{
					Value:        v,
					Configurable: true,
				}
			}
			n.Hooks = append(n.Hooks, sdk.WorkflowNodeHook{
				WorkflowHookModel: sdk.WorkflowHookModel{
					Name: h.Model,
				},
				Config: cfg,
			})
		}
	}

	if len(w.PipelineHooks) > 0 {
		//Only one node workflow
		addHooks(w.PipelineHooks)
		return
	}

	addHooks(w.Hooks[n.Name])
}

func (e *NodeEntry) processNode(name string, w *sdk.Workflow) (bool, error) {
	if err := e.checkValidity(); err != nil {
		return false, err
	}

	var ancestorsExist = true
	var ancestors []*sdk.WorkflowNode

	if len(e.DependsOn) == 1 {
		a := e.DependsOn[0]
		//Looking for the ancestor
		ancestor := w.GetNodeByName(a)
		if ancestor == nil {
			ancestorsExist = false
		}
		ancestors = append(ancestors, ancestor)
	} else {
		for _, a := range e.DependsOn {
			//Looking for the ancestor
			ancestor := w.GetNodeByName(a)
			if ancestor == nil {
				ancestorsExist = false
				break
			}
			ancestors = append(ancestors, ancestor)
		}
	}

	if !ancestorsExist {
		return false, nil
	}

	n, err := e.getNode(name)
	if err != nil {
		return false, err
	}

	switch len(ancestors) {
	case 0:
		w.Root = n
		return true, nil
	case 1:
		w.AddTrigger(ancestors[0].Name, *n)
		return true, nil
	}

	//Try to find an existing join with the same references
	var join *sdk.WorkflowNodeJoin
	for i := range w.Joins {
		j := &w.Joins[i]
		var joinFound = true

		for _, ref := range j.SourceNodeRefs {
			var refFound bool
			for _, a := range e.DependsOn {
				if ref == a {
					refFound = true
					break
				}
			}
			if !refFound {
				joinFound = false
				break
			}
		}

		if joinFound {
			join = j
		}
	}

	var appendJoin bool
	if join == nil {
		join = &sdk.WorkflowNodeJoin{
			SourceNodeRefs: e.DependsOn,
		}
		appendJoin = true
	}

	join.Triggers = append(join.Triggers, sdk.WorkflowNodeJoinTrigger{
		WorkflowDestNode: *n,
	})

	if appendJoin {
		w.Joins = append(w.Joins, *join)
	}
	return true, nil

}
