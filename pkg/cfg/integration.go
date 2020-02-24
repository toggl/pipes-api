package cfg

type (
	Integration struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Link       string  `json:"link"`
		Image      string  `json:"image"`
		Pipes      []*Pipe `json:"pipes"`
		AuthURL    string  `json:"auth_url,omitempty"`
		AuthType   string  `json:"auth_type,omitempty"`
		Authorized bool    `json:"authorized"`
	}
)
