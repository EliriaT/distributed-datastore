package cluster

type Command struct {
	Key         string      `json:"key"`
	Value       string      `json:"value"`
	CommandType commandList `json:"command_type"`
}
type commandList int

const (
	SET commandList = iota
	GET
)
