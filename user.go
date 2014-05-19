package main

type (
	WorkspaceUser struct {
		ID       int64  `json:"id"`
		UID      int64  `json:"uid"`
		Wid      int64  `json:"wid"`
		Admin    bool   `json:"admin"`
		Active   bool   `json:"active"`
		Email    string `json:"email"`
		Inactive bool   `json:"inactive"`
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
