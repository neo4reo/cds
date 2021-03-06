package bitbucket

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) Branches(fullname string) ([]sdk.VCSBranch, error) {
	branches := []sdk.VCSBranch{}

	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return branches, sdk.ErrRepoNotFound
	}

	stashBranches := []Branch{}

	path := fmt.Sprintf("/projects/%s/repos/%s/branches", t[0], t[1])
	params := url.Values{}

	nextPage := 0
	for {
		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response BranchResponse
		if err := b.do("GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "vcs> bitbucket> branches> Unable to get branches %s", path)
		}

		stashBranches = append(stashBranches, response.Values...)
		if response.IsLastPage {
			break
		} else {
			nextPage += response.Size
		}
	}

	for _, sb := range stashBranches {
		b := sdk.VCSBranch{
			ID:           sb.ID,
			DisplayID:    sb.DisplayID,
			LatestCommit: sb.LatestHash,
			Default:      sb.IsDefault,
		}
		branches = append(branches, b)
	}

	return branches, nil
}
func (b *bitbucketClient) Branch(fullname string, filter string) (*sdk.VCSBranch, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, sdk.ErrRepoNotFound
	}

	branches := BranchResponse{}
	path := fmt.Sprintf("/projects/%s/repos/%s/branches?filterText=%s", t[0], t[1], url.QueryEscape(filter))

	if err := b.do("GET", "core", path, nil, nil, &branches); err != nil {
		return nil, sdk.WrapError(err, "vcs> bitbucket> branches> Unable to get branch %s %s", filter, path)
	}

	if len(branches.Values) == 0 {
		return nil, sdk.ErrNotFound
	}

	branch := &sdk.VCSBranch{
		ID:           branches.Values[0].ID,
		DisplayID:    branches.Values[0].DisplayID,
		LatestCommit: branches.Values[0].LatestHash,
		Default:      branches.Values[0].IsDefault,
	}
	return branch, nil
}
