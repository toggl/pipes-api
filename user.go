package main

type (
	WorkspaceUser struct {
		ID        int    `json:"id"`
		Uid       int    `json:"uid"`
		Wid       int    `json:"wid"`
		Admin     bool   `json:"admin"`
		Active    bool   `json:"active"`
		Email     string `json:"email"`
		Inactive  bool   `json:"inactive"`
		ForeignID int    `json:"foreign_id,omitempty"`
	}

	usersRequest struct {
		Emails []string `json:"emails"`
	}

	UsersImport struct {
		WorkspaceUsers []*WorkspaceUser `json:"workspace_users"`
		Notifications  []string         `json:"notifications"`
	}
)

func (p *UsersImport) Count() int {
	return len(p.WorkspaceUsers)
}
