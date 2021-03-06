package sql

import (
	"errors"
	"fmt"
	"strings"

	gwerr "gateway/errors"
	"gateway/stats"
)

var allSamples = append(stats.AllSamples(), "ms")
var allMeasurements = stats.AllMeasurements()
var rowLength = len(allSamples)

// logQuery generates the INSERT statement for the given vals.
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

// getArgs retrieves the args for the INSERT given the slice of stats.Point's.
func getArgs(node string, points ...stats.Point) ([]interface{}, error) {
	if len(points) < 1 {
		return nil, errors.New("must pass at least one stats.Point")
	}

	args := make([]interface{}, len(points)*rowLength)
	errs := make([]error, len(points))

	for i, point := range points {
		errs[i] = setPointArgs(point, node, args[i*rowLength:(i+1)*rowLength])
	}

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return args, nil
}

// setPointArgs assigns the args which will be passed to the INSERT as
// interpolation values.
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
	if s.NAME != "" {
		node = s.NAME
	}

	// get the args we'll use as interpolation values in the INSERT.
	args, err := getArgs(node, ps...)
	if err != nil {
		return gwerr.NewWrapped(
			"failed to get args for stats query", err,
		)
	}

	// generate the INSERT query we'll use.
	query := logQuery(
		s.Parameters,
		len(ps),
	)

	// Execute the query.
	tx, txerr := s.Begin()
	if txerr != nil {
		return gwerr.NewWrapped("failed to start transaction", txerr)
	}
	_, err = tx.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return gwerr.NewWrapped("failed to exec stats query", err)
	}
	tx.Commit()
	return nil
}
