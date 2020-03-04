package pipe

import (
	"strings"

	"github.com/toggl/pipes-api/pkg/toggl"
)

func trimSpacesFromName(ps []*toggl.Project) []*toggl.Project {
	var trimmedPs []*toggl.Project
	for _, p := range ps {
		p.Name = strings.TrimSpace(p.Name)
		if len(p.Name) > 0 {
			trimmedPs = append(trimmedPs, p)
		}
	}
	return trimmedPs
}
