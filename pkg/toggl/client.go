package toggl

type (
	ClientRequest struct {
		Clients []*Client `json:"clients"`
	}

	ClientsImport struct {
		Clients       []*Client `json:"clients"`
		Notifications []string  `json:"notifications"`
	}
)

func (p *ClientsImport) Count() int {
	return len(p.Clients)
}
