package errnotifier

import (
	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type BugSnagNotifier struct {
	apiKey  string
	envType string
}

func NewBugSnagNotifier(APIKey, envType string) *BugSnagNotifier {
	bn := &BugSnagNotifier{
		apiKey:  APIKey,
		envType: envType,
	}

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       bn.apiKey,
		ReleaseStage: bn.envType,
		NotifyReleaseStages: []string{
			environment.EnvTypeProduction,
			environment.EnvTypeStaging,
		},
		// more configuration options
	})

	return bn
}

// NotifyPipeError notifies bugsnag with metadata for the given pipe.
func (n *BugSnagNotifier) NotifyPipeError(p *environment.PipeConfig, err error) {
	bugsnag.Notify(err, bugsnag.MetaData{
		"pipe": {
			"ID":            p.ID,
			"Name":          p.Name,
			"ServiceParams": string(p.ServiceParams),
			"WorkspaceID":   p.WorkspaceID,
			"ServiceID":     p.ServiceID,
		},
	})
}

// NotifyError notifies bugsnag with error.
func (n *BugSnagNotifier) NotifyError(err error) {
	bugsnag.Notify(err)
}

// NotifyTimeEntryError notifies bugsnag with TimeEntry integration error.
func (n *BugSnagNotifier) NotifyTimeEntryError(workspaceID int, te toggl.TimeEntry, err error) {
	bugsnag.Notify(err, bugsnag.MetaData{
		"Workspace": {
			"ID": workspaceID,
		},
		"Entry": {
			"ID":        te.ID,
			"TaskID":    te.TaskID,
			"UserID":    te.UserID,
			"ProjectID": te.ProjectID,
		},
		"Foreign Entry": {
			"ForeignID":        te.ForeignID,
			"ForeignTaskID":    te.ForeignTaskID,
			"ForeignUserID":    te.ForeignUserID,
			"ForeignProjectID": te.ForeignProjectID,
		},
	})
}
