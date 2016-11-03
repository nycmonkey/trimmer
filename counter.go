package trimmer

import (
	"io"
	"strings"

	boom "github.com/tylertreat/BoomFilters"
)

// Counter is implemented to track how often a string occurs in a corpus / set
type Counter interface {
	Add([]byte)
	Count([]byte) uint64
	Export(io.Writer) (int, error)
	Import(io.Reader) (int, error)
}

type cmsCounter struct {
	cms *boom.CountMinSketch
}

func (ctr *cmsCounter) Add(t []byte) {
	ctr.cms.Add(t)
}

func (ctr *cmsCounter) Count(t []byte) uint64 {
	return ctr.cms.Count(t)
}

func (ctr *cmsCounter) Export(w io.Writer) (int, error) {
	return ctr.cms.WriteDataTo(w)
}

func (ctr *cmsCounter) Import(r io.Reader) (int, error) {
	return ctr.cms.ReadDataFrom(r)
}

// nonRedundant filters out strings that have shorter prefixes in the
// set.  There's no use using both "foo" and "foo bar" as search terms
// when "foo bar" results are a strict subset of "foo" results.
// The input 'list' must be sorted.
func nonRedundant(list []string) (result []string) {
	if len(list) < 2 {
		return list
	}
	prev := list[0]
	result = append(result, prev)
	for _, item := range list[1:] {
		if strings.HasPrefix(item, prev) {
			continue
		}
		result = append(result, item)
		prev = item
	}
	return
}

func max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}
