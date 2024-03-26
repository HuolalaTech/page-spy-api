package data

type Tag struct {
	Model
	Key   string `json:"key"`
	Value string `json:"value"`
}
