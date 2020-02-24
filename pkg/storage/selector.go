package storage

type Selector struct {
	IDs         []int `json:"ids"`
	SendInvites bool  `json:"send_invites"`
}
