// package cron provides ability to parse and run a cron (https://en.wikipedia.org/wiki/Cron) like schedule
//
// expression parsing is inspired by https://github.com/robfig/cron
package cron

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// represent '*' in cron where all bit is set to 1
const star = field(^uint64(0))

type field uint64

// match check if the current value matches the field bitmap.
func (f field) match(value int) bool {
	return f&(1<<uint64(value)) != 0
}

func (f field) format() string {
	if f == star {
		return "*"
	}

	buffer := make([]string, 0)
	for i := 0; i < 64; i++ {
		if f.match(i) {
			buffer = append(buffer, strconv.Itoa(i))
		}
	}

	return strings.Join(buffer, ",")
}

// Entry represents a single cron entry
type Entry struct {
	Name     string
	Location *time.Location

	minute, hour, dom, month, dow field
}

func (e Entry) String() string {
	str := []string{e.minute.format(), e.hour.format(), e.dom.format(), e.month.format(), e.dow.format()}

	return fmt.Sprintf("{ name:%q schedule:%q, location:%q }", e.Name, strings.Join(str, " "), e.Location)
}

func (e Entry) Match(t time.Time) bool {
	t = t.In(e.Location)

	return e.minute.match(t.Minute()) &&
		e.hour.match(t.Hour()) &&
		e.dom.match(t.Day()) &&
		e.dow.match(int(t.Weekday())) &&
		e.month.match(int(t.Month()))
}

// Parse a cron expression on a location. If location is nil it uses system location
// it does not support macro (ex: @monthly)
//
// ex format:
//
//  +------------------ Minute (0-59)       : [5]
//  | +---------------- Hour (0-23)         : [0, 1, 2, ..., 23]
//  | |   +------------ Day of month (1-31) : [5, 10, 15, 20, 30]
//  | |   |    +------- Month (1-12)        : [1, 3, 5, ..., 11]
//  | |   |    |     +- Day of Week  (0-6)  : [Sun, Mon, Tue, Wed]
//  5 *  */5 1-12/2 0-3
func Parse(expression string, loc *time.Location, name string) (Entry, error) {
	if loc == nil {
		loc = time.Local
	}
	e := Entry{
		Name:     name,
		Location: loc,
	}
	fields := strings.Fields(expression)
	if len(fields) != 5 {
		return e, fmt.Errorf("got %d want %d expressions", len(fields), 5)
	}

	var err error
	e.minute, err = parseField(fields[0], 0, 59)
	if err != nil {
		return e, fmt.Errorf("failed parsing 'minute' field %q: %v", fields[0], err)
	}
	e.hour, err = parseField(fields[1], 0, 23)
	if err != nil {
		return e, fmt.Errorf("failed parsing 'hour' field %q: %v", fields[1], err)
	}
	e.dom, err = parseField(fields[2], 1, 31)
	if err != nil {
		return e, fmt.Errorf("failed parsing 'day of month' field %q: %v", fields[2], err)
	}
	e.month, err = parseField(fields[3], 1, 12)
	if err != nil {
		return e, fmt.Errorf("failed parsing 'month' field %q: %v", fields[3], err)
	}
	e.dow, err = parseField(fields[4], 0, 6)
	if err != nil {
		return e, fmt.Errorf("failed parsing 'day of week' field %q: %v", fields[4], err)
	}

	return e, nil
}

// parseField construct bitmap where position represents a value for that field
// ex: value of minutes `1,3,5`:
//   bit             7654 3210
//   possible value  6543 210
//   bit value       0010 1010  -> [0,2,4] will be represented as uint64 value 42 (0x2A)
func parseField(s string, min, max int) (field, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty field")
	}

	if s == "*" || s == "?" {
		return star, nil
	}

	var f field
	// parse single element or parse range (ex: '2' '1-5' '*/5' '1-30/2' )
	// determine start, end and interval. Construct bitmap by traversing from start-end with interval.
	for _, part := range strings.Split(s, ",") {
		var (
			err                        error
			interval                   = 1
			startInterval, endInterval = min, max
		)

		// parse interval (ex: '*/5' '1-30/2') if exists
		if i := strings.IndexByte(part, '/'); i >= 0 {
			if r := part[:i]; r != "*" && r != "?" && strings.IndexByte(r, '-') < 0 {
				return 0, fmt.Errorf("step given without range, expression %q", s)
			}

			step := part[i+1:]
			interval, err = strconv.Atoi(step)
			if err != nil {
				return 0, fmt.Errorf("failed parsing interval expression %q: %s", step, err)
			}
			part = part[:i]
		}

		start, end := part, part
		// parse range if exist
		if i := strings.IndexByte(part, '-'); i >= 0 {
			start, end = part[:i], part[i+1:]
		}

		// determine start & end, some cron format use '?' instead of '*'
		if start != "*" && start != "?" {
			startInterval, err = strconv.Atoi(start)
			if err != nil {
				return 0, fmt.Errorf("failed parsing expression %q: %s", s, err)
			}

			// parse end interval if exists, else it will be same as start (single value)
			if end != "" {
				endInterval, err = strconv.Atoi(end)
				if err != nil {
					return 0, fmt.Errorf("failed parsing expression %q: %s", s, err)
				}
			}
		}

		if startInterval < min || endInterval > max || startInterval > endInterval {
			return 0, fmt.Errorf("value out of range (%d - %d): %s", min, max, part)
		}

		// at this point we get the start, end, interval. Construct bitmap that represents possible values
		for i := startInterval; i <= endInterval; i += interval {
			f |= 1 << uint64(i)
		}
	}

	return f, nil
}
