package common

type SMS struct {
	UUID    string `json:"uuid"`
	Mobile  string `json:"mobile"`
	Body    string `json:"body"`
	Status  string `json:"status"`
	Retries int    `json:"retries"`
}
