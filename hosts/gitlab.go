package hosts

import (
	"context"
	"sync"

	"github.com/zricethezav/gitleaks/audit"
	"github.com/zricethezav/gitleaks/manager"
	"github.com/zricethezav/gitleaks/options"

	log "github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

// Gitlab wraps a gitlab client and manager. This struct implements what the Host interface defines.
type Gitlab struct {
	client  *gitlab.Client
	manager manager.Manager
	ctx     context.Context
	wg      sync.WaitGroup
}

// NewGitlabClient accepts a manager struct and returns a Gitlab host pointer which will be used to
// perform a gitlab audit on an group or user.
func NewGitlabClient(m manager.Manager) *Gitlab {
	return &Gitlab{
		manager: m,
		ctx:     context.Background(),
		client:  gitlab.NewClient(nil, options.GetAccessToken(m.Opts)),
	}
}

// Audit will audit a github user or organization's repos.
func (g *Gitlab) Audit() {
	var (
		projects []*gitlab.Project
		resp     *gitlab.Response
		err      error
	)

	page := 1
	listOpts := gitlab.ListOptions{
		PerPage: 100,
		Page:    page,
	}
	for {
		var _projects []*gitlab.Project
		if g.manager.Opts.User != "" {
			glOpts := &gitlab.ListProjectsOptions{
				ListOptions: listOpts,
			}
			projects, resp, err = g.client.Projects.ListUserProjects(g.manager.Opts.User, glOpts)

		} else if g.manager.Opts.Organization != "" {
			glOpts := &gitlab.ListGroupProjectsOptions{
				ListOptions: listOpts,
			}
			projects, resp, err = g.client.Groups.ListGroupProjects(g.manager.Opts.Organization, glOpts)
		}
		if err != nil {
			log.Error(err)
		}

		projects = append(projects, _projects...)
		if resp == nil {
			break
		}
		if page >= resp.TotalPages {
			// exit when we've seen all pages
			break
		}
		page = resp.NextPage
	}

	// iterate of gitlab projects
	for _, p := range projects {
		r := audit.NewRepo(&g.manager)
		cloneOpts := g.manager.CloneOptions
		cloneOpts.URL = p.HTTPURLToRepo
		err := r.Clone(cloneOpts)
		// TODO handle clone retry with ssh like github host
		r.Name = p.Name

		if err = r.Audit(); err != nil {
			log.Error(err)
		}
	}
}

// AuditPR TODO not implemented
func (g *Gitlab) AuditPR() {
	log.Error("AuditPR is not implemented in Gitlab host yet...")
}
