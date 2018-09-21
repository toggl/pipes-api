package teamweek

import (
	"fmt"
)

type (
	Workspace struct {
		ID          int64  `json:"id,omitempty"`
		Name        string `json:"name,omitempty"`
		Active      bool   `json:"active,omitempty"`
		Role        string `json:"role,omitempty"`
		SuspendedAt string `json:"suspended_at,omitempty"`
		CreatedAt   string `json:"created_at,omitempty"`
		UpdatedAt   string `json:"updated_at,omitempty"`
	}

	UserProfile struct {
		ID         int64       `json:"id,omitempty"`
		Name       string      `json:"name,omitempty"`
		Email      string      `json:"email,omitempty"`
		Initials   string      `json:"initials,omitempty"`
		PictureUrl string      `json:"picture_url,omitempty"`
		HasPicture bool        `json:"has_picture,omitempty"`
		Workspaces []Workspace `json:"workspaces,omitempty"`
		ProjectIDs []int64     `json:"project_ids,omitempty"`
		CreatedAt  string      `json:"created_at,omitempty"`
		UpdatedAt  string      `json:"updated_at,omitempty"`
	}

	Member struct {
		Role              string `json:"role,omitempty"`
		Active            bool   `json:"active,omitempty"`
		HoursPerWorkDay   string `json:"hours_per_work_day,omitempty"`
		MinutesPerWorkDay string `json:"minutes_per_work_day,omitempty"`
		Dummy             bool   `json:"dummy,omitempty"`
		PictureUrl        string `json:"picture_url,omitempty"`
		HasPicture        bool   `json:"has_picture,omitempty"`
		ActivatedAt       string `json:"activated_at,omitempty"`
		DeactivatedAt     string `json:"deactivated_at,omitempty"`
		ID                int64  `json:"id,omitempty"`
		Name              string `json:"name,omitempty"`
		Email             string `json:"email,omitempty"`
		Initials          string `json:"initials,omitempty"`
		CreatedAt         string `json:"created_at,omitempty"`
		UpdatedAt         string `json:"updated_at,omitempty"`
	}

	Task struct {
		ID               int64    `json:"id,omitempty"`
		Name             string   `json:"name,omitempty"`
		Notes            string   `json:"notes,omitempty"`
		StartDate        string   `json:"start_date,omitempty"`
		EndDate          string   `json:"end_date,omitempty"`
		StartTime        string   `json:"start_time,omitempty"`
		EndTime          string   `json:"end_time,omitempty"`
		EstimatedMinutes int64    `json:"estimated_minutes,omitempty"`
		Done             bool     `json:"done,omitempty"`
		Pined            bool     `json:"pinned,omitempty"`
		UserID           int64    `json:"user_id,omitempty"`
		FolderID         int64    `json:"folder_id,omitempty"`
		Position         int64    `json:"position,omitempty"`
		Weight           float64  `json:"weight,omitempty"`
		ProjectID        int64    `json:"project_id,omitempty"`
		Project          *Project `json:"project,omitempty"`
		CreatedAt        string   `json:"created_at,omitempty"`
		UpdatedAt        string   `json:"updated_at,omitempty"`
	}

	Milestone struct {
		ID        int64  `json:"id,omitempty"`
		Name      string `json:"name,omitempty"`
		Date      string `json:"date,omitempty"`
		Done      bool   `json:"done,omitempty"`
		Holiday   bool   `json:"holiday,omitempty"`
		GroupID   int64  `json:"group_id,omitempty"`
		CreatedAt string `json:"created_at,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
	}

	Project struct {
		ID        int64  `json:"id,omitempty"`
		Name      string `json:"name,omitempty"`
		CreatedAt string `json:"created_at,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
	}

	Membership struct {
		ID        int64  `json:"id,omitempty"`
		UserID    int64  `json:"user_id,omitempty"`
		Position  int64  `json:"position,omitempty"`
		Weight    int64  `json:"weight,omitempty"`
		CreatedAt string `json:"created_at,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
	}

	Group struct {
		ID          int64        `json:"id,omitempty"`
		Name        string       `json:"name,omitempty"`
		Memberships []Membership `json:"memberships,omitempty"`
		CreatedAt   string       `json:"created_at,omitempty"`
		UpdatedAt   string       `json:"updated_at,omitempty"`
	}
)

func (c *Client) GetUserProfile() (*UserProfile, error) {
	profile := UserProfile{}

	if err := c.get("me", &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (c *Client) ListWorkspaceMembers(workspaceID int64) ([]Member, error) {
	members := []Member{}
	url := fmt.Sprintf("%d/members", workspaceID)

	if err := c.get(url, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (c *Client) ListWorkspaceProjects(workspaceID int64) ([]Project, error) {
	projects := []Project{}
	url := fmt.Sprintf("%d/projects", workspaceID)

	if err := c.get(url, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (c *Client) ListWorkspaceMilestones(workspaceID int64) ([]Milestone, error) {
	milestones := []Milestone{}
	url := fmt.Sprintf("%d/milestones", workspaceID)

	if err := c.get(url, &milestones); err != nil {
		return nil, err
	}
	return milestones, nil
}

func (c *Client) ListWorkspaceGroups(workspaceID int64) ([]Group, error) {
	groups := []Group{}
	url := fmt.Sprintf("%d/groups", workspaceID)

	if err := c.get(url, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (c *Client) ListWorkspaceTasks(workspaceID int64) ([]Task, error) {
	tasks := []Task{}
	url := fmt.Sprintf("%d/tasks", workspaceID)

	if err := c.get(url, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}
