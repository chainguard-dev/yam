package path

import "strconv"

const root = "root"

type PartKind int

const (
	RootKind PartKind = iota
	MapKind
	SeqKind
)

const (
	anyKey   = "//any-key//"
	anyIndex = -1
)

type Part interface {
	id() string
	Kind() PartKind
}

type mapPart struct {
	key string
}

func (p mapPart) id() string {
	return p.key
}

func (p mapPart) Kind() PartKind {
	return MapKind
}

type seqPart struct {
	index int
}

func (p seqPart) id() string {
	return strconv.Itoa(p.index)
}

func (p seqPart) Kind() PartKind {
	return SeqKind
}

type rootPart struct{}

func (p rootPart) id() string {
	return root
}

func (p rootPart) Kind() PartKind {
	return RootKind
}
