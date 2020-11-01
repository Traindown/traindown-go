package traindown

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

// Metadata is key value pairs.
type Metadata map[string]interface{}

// Performance is an expression of a movement.
type Performance struct {
	Fails        int     `json:"fails"`
	Load         float32 `json:"load"`
	PercentOfMax float32 `json:"percentOfMax,omitempty"`
	Reps         int     `json:"reps"`
	Sequence     int     `json:"sequence"`
	Sets         int     `json:"sets"`
	Unit         string  `json:"unit"`

	Metadata Metadata `json:"metadata"`
	Notes    []string `json:"notes"`
}

// NewPerformance spits out a new Performance
func NewPerformance() *Performance {
	return &Performance{
		Metadata: make(Metadata),
		Notes:    make([]string, 0),
		Reps:     1,
		Sets:     1,
	}
}

func (p Performance) String() string {
	ps, _ := json.Marshal(p)
	return string(ps)
}

// Movement is an thing you do, you know?
type Movement struct {
	Name     string `json:"name"`
	Sequence int    `json:"sequence"`
	SuperSet bool   `json:"superSet"`

	Performances []*Performance `json:"performances"`

	Metadata Metadata `json:"metadata"`
	Notes    []string `json:"notes"`
}

// NewMovement spits out a new Movement
func NewMovement() *Movement {
	return &Movement{
		Metadata:     make(Metadata),
		Notes:        make([]string, 0),
		Performances: make([]*Performance, 0),
	}
}

// Session is a collection of Movements that occurred.
type Session struct {
	Date      time.Time   `json:"date"`
	Errors    []error     `json:"errors"`
	Movements []*Movement `json:"movements"`

	Metadata Metadata `json:"metadata"`
	Notes    []string `json:"notes"`
}

// NewSession spits out a new Session
func NewSession() *Session {
	return &Session{
		Metadata:  make(Metadata),
		Movements: make([]*Movement, 0),
		Notes:     make([]string, 0),
	}
}

// ParseByte takes in a Traindown byte slice and returns a pointer to a Session.
func ParseByte(txt []byte) (*Session, error) {
	lexer, err := NewLexer()

	if err != nil {
		return &Session{}, err
	}

	tokens, err := lexer.Scan(txt)

	if err != nil {
		return &Session{}, fmt.Errorf("Failed to parse: %q", err)
	}

	s, err := parse(tokens)

	if err != nil {
		return &Session{}, fmt.Errorf("Failed to parse: %q", err)
	}

	return s, nil
}

// ParseString takes in a Traindown string and returns a pointer to a Session.
func ParseString(txt string) (*Session, error) {
	lexer, err := NewLexer()

	if err != nil {
		return &Session{}, err
	}

	tokens, err := lexer.Scan([]byte(txt))

	if err != nil {
		return &Session{}, fmt.Errorf("Failed to parse: %q", err)
	}

	s, err := parse(tokens)

	if err != nil {
		return &Session{}, fmt.Errorf("Failed to parse: %q", err)
	}

	return s, nil
}

func floatValue(s string, t string) (float32, error) {
	f, err := strconv.ParseFloat(s, 32)

	if err != nil {
		return 0.0, fmt.Errorf("Failed to parse %q: %q", t, s)
	}

	return float32(f), nil
}

func intValue(s string, t string) (int, error) {
	i, err := strconv.Atoi(s)

	if err != nil {
		return 0, fmt.Errorf("Failed to parse %q: %q", t, s)
	}

	return i, nil
}

func parse(tokens []*Token) (*Session, error) {
	m := NewMovement()
	p := NewPerformance()
	s := NewSession()

	inSession := true
	inPerformance := false

	for _, tok := range tokens {
		switch tok.Name() {
		case "DATE":
			d, err := dateparse.ParseAny(tok.Value())

			if err != nil {
				s.Errors = append(s.Errors, fmt.Errorf("Failed to parse date: %q. Using today UTC", err))
				s.Date = time.Now()
			} else {
				s.Date = d
			}
		case "FAILS":
			i, err := intValue(tok.Value(), "fails")

			if err != nil {
				s.Errors = append(s.Errors, err)
			}

			p.Fails = i
		case "LOAD":
			if inPerformance {
				m.Performances = append(m.Performances, p)
				p = NewPerformance()
			}
			f, err := floatValue(tok.Value(), "load")

			if err != nil {
				s.Errors = append(s.Errors, err)
			}

			p.Load = f
			inPerformance = true
		case "METADATA":
			pair := strings.Split(tok.Value(), ":")
			key := strings.Trim(pair[0], " ")
			value := strings.Trim(pair[1], " ")

			if inSession {
				s.Metadata[key] = value
			} else if inPerformance {
				p.Metadata[key] = value
			} else {
				m.Metadata[key] = value
			}
		case "MOVEMENT", "MOVEMENT_SS":
			inSession = false

			if inPerformance {
				m.Performances = append(m.Performances, p)
				p = NewPerformance()
			}
			inPerformance = false

			if m.Name != "" {
				s.Movements = append(s.Movements, m)
				m = NewMovement()
			}

			m.Name = tok.Value()

			if tok.Name() == "MOVEMENT_SS" {
				m.SuperSet = true
			}
		case "NOTE":
			if inSession {
				s.Notes = append(s.Notes, tok.Value())
			} else if inPerformance {
				p.Notes = append(p.Notes, tok.Value())
			} else {
				m.Notes = append(m.Notes, tok.Value())
			}
		case "REPS":
			i, err := intValue(tok.Value(), "reps")

			if err != nil {
				s.Errors = append(s.Errors, err)
			}

			p.Reps = i
		case "SETS":
			i, err := intValue(tok.Value(), "sets")

			if err != nil {
				s.Errors = append(s.Errors, err)
			}

			p.Sets = i
		}
	}

	if p.Load != 0.0 {
		m.Performances = append(m.Performances, p)
	}

	if m.Name != "" {
		s.Movements = append(s.Movements, m)
	}

	return s, nil
}