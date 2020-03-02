package pipes

type Selector struct {
	IDs         []int `json:"ids"`
	SendInvites bool  `json:"send_invites"`
}
