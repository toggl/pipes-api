package toggl

type (
	UsersRequest struct {
		Users []*User `json:"users"`
	}

	UsersImport struct {
		WorkspaceUsers []*User  `json:"users"`
		Notifications  []string `json:"notifications"`
	}
)

func (p *UsersImport) Count() int {
	return len(p.WorkspaceUsers)
}
