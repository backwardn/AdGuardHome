// Module for managing statistics for DNS filtering server

package stats

import (
	"net"
)

// New - create object
// filename: DB file name
// limit: time limit (in days)
func New(filename string, limit int) Stats {
	return createObject(filename, limit)
}

// Stats - main interface
type Stats interface {
	// Close object.
	// This function is not thread safe
	//  (can't be called in parallel with any other function of this interface).
	Close()

	// Set new configuration at runtime.
	// limit: time limit (in days)
	Configurate(limit int)

	// Reset counters and clear database
	Clear()

	// Update counters
	Update(e Entry)

	// Get data
	GetData(timeUnit TimeUnit) map[string]interface{}
}

// TimeUnit - time unit
type TimeUnit int

// Supported time units
const (
	Hours TimeUnit = iota
	Days
)

// Result of DNS request processing
type Result int

// Supported result values
const (
	RNotFiltered Result = iota + 1
	RFiltered
	RSafeBrowsing
	RSafeSearch
	RParental
	rLast
)

// Entry - data to add
type Entry struct {
	Domain string
	Client net.IP
	Result Result
	Time   uint // processing time (msec)
}
