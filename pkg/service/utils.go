package service

import (
	"strings"

	"github.com/toggl/pipes-api/pkg/domain"
)

func trimSpacesFromName(ps []*domain.Project) []*domain.Project {
	var trimmedPs []*domain.Project
	for _, p := range ps {
		p.Name = strings.TrimSpace(p.Name)
		if len(p.Name) > 0 {
			trimmedPs = append(trimmedPs, p)
		}
	}
	return trimmedPs
}
