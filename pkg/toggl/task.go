package toggl

type (
	TaskRequest struct {
		Tasks []*Task `json:"tasks"`
	}
	TasksImport struct {
		Tasks         []*Task  `json:"tasks"`
		Notifications []string `json:"notifications"`
	}
)

func (p *TasksImport) Count() int {
	return len(p.Tasks)
}
