package main

type (
	projectRequest struct {
		Projects []*Project `json:"projects"`
		SupportsClient bool `json:"supports_client"`
	}

	ProjectsImport struct {
		Projects      []*Project `json:"projects"`
		Notifications []string   `json:"notifications"`
	}
)

func (p *ProjectsImport) Count() int {
	return len(p.Projects)
}
