package response

type responseState string

const (
	stateStatusLine responseState = "status line"
	stateHeaders    responseState = "headers"
	stateBody       responseState = "body"
	stateDone       responseState = "done"
)

func newResponseState() responseState {
	return stateStatusLine
}

func (rs responseState) advance() responseState {
	switch rs {
	case stateStatusLine:
		return stateHeaders
	case stateHeaders:
		return stateBody
	case stateBody:
		return stateDone
	default:
		panic("invalid response state advance: " + rs)
	}
}
