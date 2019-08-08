package stats

import "net"

// Stats - global context
type Stats struct {
	limit int

	unit unit
}

type unit struct {
	nTotal  int
	nResult []int

	domains        map[string]int
	blockedDomains map[string]int
	clients        map[string]int

	timeSum int // usec
	timeAvg int // usec
}

// New - create object
// limit: time limit (in days)
func New(limit int) *Stats {
	s := Stats{}
	s.initUnit(&s.unit)
	s.limit = limit
	return &s
}

// Close - close global object
func (s *Stats) Close() {

}

func (s *Stats) initUnit(u *unit) {
	u.nResult = make([]int, P+1)
	u.domains = make(map[string]int)
	u.blockedDomains = make(map[string]int)
	u.clients = make(map[string]int)
}

type Result int

const (
	NF Result = iota + 1
	F
	SB
	SS
	P
)

// Entry - data to add
type Entry struct {
	Domain string
	Client net.IP
	Result Result
	Time   uint // processing time (msec)
}

// Update - update counters
func (s *Stats) Update(e Entry) {
	if e.Result == 0 ||
		len(e.Domain) == 0 ||
		!(len(e.Client) == 4 || len(e.Client) == 16) {
		return
	}
	u := &s.unit

	u.nResult[e.Result]++

	if e.Result == NF {
		u.domains[e.Domain]++
	} else {
		u.blockedDomains[e.Domain]++
	}

	u.clients[e.Client.String()]++
	u.timeSum += int(e.Time)
	u.nTotal++
}

// Get - get data
func (s *Stats) Get() map[string]interface{} {
	d := map[string]interface{}{}

	d["time_units"] = "hours"

	u := &s.unit
	d["num_dns_queries"] = u.nTotal
	d["num_blocked_filtering"] = u.nResult[F]
	d["num_replaced_safebrowsing"] = u.nResult[SB]
	d["num_replaced_safesearch"] = u.nResult[SS]
	d["num_replaced_parental"] = u.nResult[P]

	avg := u.timeAvg
	if u.timeAvg == 0 && u.nTotal != 0 {
		avg = u.timeSum / u.nTotal
	}
	d["avg_processing_time"] = float64(avg) / 1000000

	a := []int{}
	a = append(a, u.nTotal)
	d["dns_queries"] = a

	a = []int{}
	a = append(a, u.nResult[F])
	d["blocked_filtering"] = a

	a = []int{}
	a = append(a, u.nResult[SB])
	d["replaced_safebrowsing"] = a

	a = []int{}
	a = append(a, u.nResult[P])
	d["replaced_parental"] = a

	m := []map[string]interface{}{}
	for k, v := range u.domains {
		ent := map[string]interface{}{}
		ent[k] = v
		m = append(m, ent)
	}
	d["top_queried_domains"] = m

	m = []map[string]interface{}{}
	for k, v := range u.blockedDomains {
		ent := map[string]interface{}{}
		ent[k] = v
		m = append(m, ent)
	}
	d["top_blocked_domains"] = m

	m = []map[string]interface{}{}
	for k, v := range u.clients {
		ent := map[string]interface{}{}
		ent[k] = v
		m = append(m, ent)
	}
	d["top_clients"] = m

	return d
}
