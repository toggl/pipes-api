package errnotifier

import (
	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/toggl"
)

// DummyNotifier doing nothing, just stub implementation for test purposes.
type DummyNotifier struct{}

// NewDummyNotifier creates new dummy notifier for test purposes.
func NewDummyNotifier() *DummyNotifier {
	return &DummyNotifier{}
}

func (n *DummyNotifier) NotifyPipeError(p *environment.PipeConfig, err error)                {}
func (n *DummyNotifier) NotifyError(err error)                                               {}
func (n *DummyNotifier) NotifyTimeEntryError(workspaceID int, te toggl.TimeEntry, err error) {}
