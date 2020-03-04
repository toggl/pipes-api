package pipe

type Integration struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Link       string  `json:"link"`
	Image      string  `json:"image"`
	AuthURL    string  `json:"auth_url,omitempty"`
	AuthType   string  `json:"auth_type,omitempty"`
	Authorized bool    `json:"authorized"`
	Pipes      []*Pipe `json:"pipes"`
}
