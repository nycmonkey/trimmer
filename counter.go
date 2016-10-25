package trimmer

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	boom "github.com/tylertreat/BoomFilters"
)

type Counter interface {
	Add([]byte)
	Count([]byte) uint64
	Export(io.Writer) (int, error)
	Import(io.Reader) (int, error)
}

type cmsCounter struct {
	cms *boom.CountMinSketch
}

type mapCounter struct {
	m map[string]uint64
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

func (ctr *mapCounter) Add(t []byte) {
	ctr.m[string(t)]++
}

func (ctr *mapCounter) Count(t []byte) uint64 {
	return ctr.m[string(t)]
}

func (ctr *mapCounter) Import(r io.Reader) (n int, err error) {
	scanner := bufio.NewScanner(r)
	var token string
	var count uint64
	var fields int
	for scanner.Scan() {
		fields, err = fmt.Sscanf(scanner.Text(), "%d %s", &count, &token)
		if err != nil {
			return
		}
		if fields != 2 {
			err = fmt.Errorf("expected to parse an integer and a string, but got %d fields", fields)
			return
		}
		ctr.m[token] = count
		n++
	}
	err = scanner.Err()
	return
}

func (ctr *mapCounter) Export(w io.Writer) (n int, err error) {
	for k, v := range ctr.m {
		var n1 int
		n1, err = fmt.Fprintln(w, k, v)
		n += n1
		if err != nil {
			return
		}
	}
	return
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

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
