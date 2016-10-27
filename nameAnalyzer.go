package trimmer

import (
	"hash/fnv"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/nycmonkey/stringy"
	boom "github.com/tylertreat/BoomFilters"
)

var (
	letters        = regexp.MustCompile(`[a-zA-Z]`)
	measurePattern = regexp.MustCompile(`^\d+[dwmysxkcbt]$`)
)

const (
	MaxFreq           = 50
	DefaultThreshhold = 12
)

type Interface interface {
	Trim(string, ...int) (string, bool)
}

type nameAnalyzer struct {
	ctr Counter
}

func NewNameAnalyzer(pathToCounterData string) (na *nameAnalyzer, err error) {
	var f *os.File
	log.Println("Reading counter data from", pathToCounterData)
	f, err = os.Open(pathToCounterData)
	if err != nil {
		return
	}
	defer f.Close()
	var ctr Counter
	cms := boom.NewCountMinSketch(0.0000001, 0.9999999)
	cms.SetHash(fnv.New64a())
	ctr = &cmsCounter{cms}
	_, err = ctr.Import(f)
	if err != nil {
		return
	}
	na = &nameAnalyzer{ctr: ctr}
	return
}

func excludeNgram(ng string, df int) bool {
	if !letters.MatchString(ng) {
		return true
	}
	if measurePattern.MatchString(ng) {
		return true
	}
	if df > MaxFreq {
		return true
	}
	return false
}

func (na nameAnalyzer) Trim(name string, threshhold ...int) (phrase string, ok bool) {
	th := DefaultThreshhold
	if len(threshhold) > 0 {
		th = threshhold[0]
	}
	tokens := stringy.MSAnalyze(name)
	switch len(tokens) {
	case 0:
		return "", false
	case 1:
		return tokens[0], true
	}
	phrase = strings.Join(tokens, "_")
	df := max(int(na.ctr.Count([]byte(phrase))), 1)
	if excludeNgram(phrase, df) {
		return "", false
	}
	for {
		tokens, ok = na.trimTokens(tokens)
		if !ok {
			return strings.Replace(phrase, "_", " ", -1), true
		}
		phrase2 := strings.Join(tokens, "_")
		df2 := max(int(na.ctr.Count([]byte(phrase2))), 1)
		if (df2 > th) || excludeNgram(phrase2, df2) {
			return strings.Replace(phrase, "_", " ", -1), true
		}
		phrase = phrase2
		df = df2
	}
}

func (na nameAnalyzer) trimTokens(tokens []string) (trimmed []string, ok bool) {
	if len(tokens) < 2 {
		return tokens, false
	}
	// if the document frequency of the unigram token at the head is more common than the token at the tail,
	// remove it
	if na.ctr.Count([]byte(tokens[0])) > na.ctr.Count([]byte(tokens[len(tokens)-1])) {
		return tokens[1:], true
	}
	return tokens[0 : len(tokens)-1], true
}
