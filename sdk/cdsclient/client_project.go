package cdsclient

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectCreate(p *sdk.Project, groupName string) error {
	if groupName != "" {
		gr := sdk.Group{}
		if _, err := c.GetJSON("/group/"+groupName, &gr); err != nil {
			return err
		}
		p.ProjectGroups = []sdk.GroupPermission{
			sdk.GroupPermission{
				Group:      gr,
				Permission: permission.PermissionReadWriteExecute,
			},
		}
	}

	if _, err := c.PostJSON("/project", p, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectDelete(key string) error {
	_, err := c.DeleteJSON("/project/"+key, nil, nil)
	return err
}

func (c *client) ProjectGet(key string, mods ...RequestModifier) (*sdk.Project, error) {
	p := &sdk.Project{}
	if _, err := c.GetJSON("/project/"+key, p, mods...); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ProjectList() ([]sdk.Project, error) {
	p := []sdk.Project{}
	if _, err := c.GetJSON("/project", &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ProjectGroupsImport(projectKey string, content io.Reader, format string, force bool) (sdk.Project, error) {
	var proj sdk.Project
	url := fmt.Sprintf("/project/%s/group/import?format=%s", projectKey, format)

	if force {
		url += "&forceUpdate=true"
	}

	btes, _, errReq := c.Request("POST", url, content)
	if errReq != nil {
		return proj, errReq
	}

	if err := json.Unmarshal(btes, &proj); err != nil {
		return proj, err
	}

	return proj, nil
}
