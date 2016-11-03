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

// Interface describes the methods a trimmer supports
type Interface interface {
	Trim(string, ...int) (string, bool)
	Counter
}

type nameAnalyzer struct {
	Counter
	defaultThreshhold uint8
	maxFreq           uint32
}

// New returns a trimmer with an unloaded probabalistic counter
// To do anything useful, it needs to be loaded with name shingles
func New(defaultThreshhold uint8, maxFreq uint32) Interface {
	cms := boom.NewCountMinSketch(0.0000001, 0.9999999)
	cms.SetHash(fnv.New64a())
	var ctr Counter = &cmsCounter{cms}
	return &nameAnalyzer{ctr, defaultThreshhold, maxFreq}
}

// NewFromFile initializes and loads a probabalistic name analyzer from serialized counter data
func NewFromFile(pathToCounterData string, defaultThreshhold uint8, maxFreq uint32) (i Interface, err error) {
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
	i = &nameAnalyzer{ctr, defaultThreshhold, maxFreq}
	return
}

func excludeNgram(ng string, df int, maxFreq uint32) bool {
	if !letters.MatchString(ng) {
		return true
	}
	if measurePattern.MatchString(ng) {
		return true
	}
	if df > int(maxFreq) {
		return true
	}
	return false
}

func (na nameAnalyzer) Trim(name string, threshhold ...int) (phrase string, ok bool) {
	tokens := stringy.MSAnalyze(name)
	switch len(tokens) {
	case 0:
		return "", false
	case 1:
		return tokens[0], true
	}
	phrase = strings.Join(tokens, "_")
	df := max(int(na.Count([]byte(phrase))), 1)
	if excludeNgram(phrase, df, na.maxFreq) {
		return "", false
	}
	for {
		tokens, ok = na.trimTokens(tokens)
		if !ok {
			return strings.Replace(phrase, "_", " ", -1), true
		}
		phrase2 := strings.Join(tokens, "_")
		df2 := max(int(na.Count([]byte(phrase2))), 1)
		if (df2 > int(na.defaultThreshhold)) || excludeNgram(phrase2, df2, na.maxFreq) {
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
	if na.Count([]byte(tokens[0])) > na.Count([]byte(tokens[len(tokens)-1])) {
		return tokens[1:], true
	}
	return tokens[0 : len(tokens)-1], true
}
