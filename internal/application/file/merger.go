package file

type Merger interface {
	MergeBlock(state, delta []byte) (newState []byte, content string, err error)
}
