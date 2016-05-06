package sql

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	gwerr "gateway/errors"
	"gateway/stats"
)

var allSamples = append(stats.AllSamples(), "ms")
var allMeasurements = stats.AllMeasurements()
var rowLength = len(allSamples)

func logQuery(paramVals func(int) []string, num int) string {
	fixedSamples := make([]string, len(allSamples))
	for i, s := range allSamples {
		fixedSamples[i] = strings.Replace(s, ".", "_", -1)
	}

	paramVs := paramVals(num * rowLength)
	params := make([]string, num)

	for i := 0; i < num; i++ {
		start, end := i*rowLength, (i+1)*rowLength
		params[i] = "(" + strings.Join(paramVs[start:end], ", ") + ")"
	}

	return fmt.Sprintf(`
INSERT INTO stats (
  %s
) VALUES
  %s
`[1:],
		strings.Join(fixedSamples, "\n  , "),
		strings.Join(params, "\n  , "),
	)
}

func getArgs(node string, ps ...stats.Point) ([]interface{}, error) {
	if len(ps) < 1 {
		return nil, errors.New("must pass at least one stats.Point")
	}

	args := make([]interface{}, len(ps)*rowLength)
	errs := make([]error, len(ps))

	var wg sync.WaitGroup

	for i, p := range ps {
		wg.Add(1)
		// concurrent safe slice mutation via re-slice
		go func(n int, p stats.Point, args []interface{}) {
			defer wg.Done()
			errs[n] = setPointArgs(p, node, args)
		}(i, p, args[i*rowLength:(i+1)*rowLength])
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return args, nil
}

func setPointArgs(p stats.Point, node string, args []interface{}) error {
	ts := p.Timestamp.UTC()
	for i, m := range allSamples {
		switch m {
		case "timestamp":
			args[i] = ts
		case "node":
			args[i] = node
		case "ms":
			args[i] = dayMillis(ts)
		default:
			if v, ok := p.Values[m]; ok {
				args[i] = v
			} else {
				// All Points must have the full set of Measurements.
				return fmt.Errorf("point missing measurement %q", m)
			}
		}
	}

	return nil
}

// Log implements stats.Logger on SQL.  Note that all Points passed must have
// all measurement values populated, or an error will be returned.
func (s *SQL) Log(ps ...stats.Point) error {
	node := "global"
	if s.ID != "" {
		node = s.ID
	}

	args, err := getArgs(node, ps...)
	if err != nil {
		return gwerr.NewWrapped(
			"failed to get args for stats query", err,
		)
	}

	query := logQuery(
		s.Parameters,
		len(ps),
	)

	_, err = s.Exec(query, args...)
	if err != nil {
		log.Println("\n" + query)
		return gwerr.NewWrapped("failed to exec stats query", err)
	}

	return nil
}
