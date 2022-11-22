package cluster

type Command struct {
	Action  commandEnum `json:"action"`
	Payload interface{} `json:"payload"`
}

type DbCommand struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	//CommandType commandList `json:"command_type"`
}
type commandEnum int

const (
	SET commandEnum = iota
	GET
	AppendEntry
	RequestVote
)

type AppendEntriesArgs struct {
	Term     int `json:"term"`
	LeaderId int `json:"leader_id"`

	//PrevLogIndex int
	//PrevLogTerm  int
	//Entries      []LogEntry
	//LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int  `json:"term"`
	Success bool `json:"success"`
}

type RequestVoteArgs struct {
	Term        int `json:"term"`
	CandidateId int `json:"candidate_id"`
	//LastLogIndex int
	//LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int  `json:"term"`
	VoteGranted bool `json:"vote_granted"`
}

// convertFromMapToDbCommand converts from an interface which is map[string]interface {}, to cluster.DbCommand
func convertFromMapToDbCommand(payload interface{}) DbCommand {
	var dbCommand DbCommand
	m := payload.(map[string]interface{})

	if key, ok := m["key"].(string); ok {
		dbCommand.Key = key
	}
	if value, ok := m["value"].(string); ok {
		dbCommand.Value = value
	}
	return dbCommand
}

// convertFromMapToRequestVoteReply converts from an interface which is map[string]interface {}, to cluster.RequestVoteReply
func convertFromMapToRequestVoteReply(payload interface{}) RequestVoteReply {
	var voteReply RequestVoteReply
	m := payload.(map[string]interface{})

	if term, ok := m["term"].(float64); ok {
		//intTerm, _ := strconv.Atoi(term)
		voteReply.Term = int(term)
	}
	if value, ok := m["vote_granted"].(bool); ok {
		voteReply.VoteGranted = value
	}
	return voteReply
}

// convertFromMapToRequestVoteArgs converts from an interface which is map[string]interface {}, to cluster.RequestVoteArgs
func convertFromMapToRequestVoteArgs(payload interface{}) RequestVoteArgs {
	var voteArgs RequestVoteArgs
	m := payload.(map[string]interface{})

	if term, ok := m["term"].(float64); ok {
		//intTerm, _ := strconv.Atoi(term)
		voteArgs.Term = int(term)
	}
	if candidateId, ok := m["candidate_id"].(float64); ok {
		voteArgs.CandidateId = int(candidateId)
	}
	return voteArgs
}

// convertFromMapToAppendEntriesReply converts from an interface which is map[string]interface {}, to cluster.AppendEntriesReply
func convertFromMapToAppendEntriesReply(payload interface{}) AppendEntriesReply {
	var appendEntry AppendEntriesReply
	m := payload.(map[string]interface{})

	if term, ok := m["term"].(float64); ok {

		appendEntry.Term = int(term)
	}
	if success, ok := m["success"].(bool); ok {
		appendEntry.Success = success
	}
	return appendEntry
}

// convertFromMapToAppendEntriesReply converts from an interface which is map[string]interface {}, to cluster.AppendEntriesReply
func convertFromMapToAppendEntriesArgs(payload interface{}) AppendEntriesArgs {
	var appendEntry AppendEntriesArgs
	m := payload.(map[string]interface{})

	if term, ok := m["term"].(float64); ok {
		//intTerm, _ := strconv.Atoi(term)
		appendEntry.Term = int(term)
	}

	if leaderId, ok := m["leader_id"].(float64); ok {
		appendEntry.LeaderId = int(leaderId)
	}

	return appendEntry
}
