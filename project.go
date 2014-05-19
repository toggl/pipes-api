package main

type (
	projectRequest struct {
		Projects []*Project `json:"projects"`
	}

	ProjectsImport struct {
		Projects      []*Project `json:"projects"`
		Notifications []string   `json:"notifications"`
	}
)

func (p *ProjectsImport) Count() int {
	return len(p.Projects)
}
