package stats

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	bolt "github.com/etcd-io/bbolt"
)

const (
	maxDomains = 100
	maxClients = 100
)

// Stats - global context
type Stats struct {
	limit    int
	filename string
	db       *bolt.DB

	unit     *unit
	unitLock sync.Mutex
}

// data for 1 time unit
type unit struct {
	id int

	nTotal  int
	nResult []int

	domains        map[string]int
	blockedDomains map[string]int
	clients        map[string]int

	timeSum int // usec
	timeAvg int // usec
}

type countPair struct {
	Name  string
	Count uint
}

// structure for storing data in file
type unitDB struct {
	NTotal  uint
	NResult []int

	Domains        []countPair
	BlockedDomains []countPair
	Clients        []countPair

	TimeAvg uint // usec
}

// New - create object
// filename: DB file name
// limit: time limit (in days)
func New(filename string, limit int) *Stats {
	s := Stats{}
	s.limit = limit
	s.filename = filename

	var err error
	log.Tracef("db.Open...")
	s.db, err = bolt.Open(s.filename, 0644, nil)
	if err != nil {
		log.Error("bolt.Open: %s: %s", s.filename, err)
		return nil
	}
	log.Tracef("db.Open")

	u := unit{}
	s.initUnit(&u)
	u.id = unitID()
	udb := s.loadUnitFromDB(u.id)
	if udb != nil {
		deserialize(&u, udb)
	}
	s.unit = &u

	go s.periodicFlush()
	return &s
}

// Close - close global object
func (s *Stats) Close() {
	u := s.swapUnit(nil)
	s.flushUnitToDB(u)

	if s.db != nil {
		log.Tracef("db.Close...")
		s.db.Close()
		log.Tracef("db.Close")
	}
}

func (s *Stats) swapUnit(new *unit) *unit {
	s.unitLock.Lock()
	u := s.unit
	s.unit = new
	s.unitLock.Unlock()
	return u
}

func unitID() int {
	return int(time.Now().Unix() / (60 * 60))
}

func (s *Stats) initUnit(u *unit) {
	u.nResult = make([]int, P+1)
	u.domains = make(map[string]int)
	u.blockedDomains = make(map[string]int)
	u.clients = make(map[string]int)
}

func (s *Stats) dbBeginTxn(wr bool) *bolt.Tx {
	if s.db == nil {
		return nil
	}

	log.Tracef("db.Begin...")
	tx, err := s.db.Begin(wr)
	if err != nil {
		log.Error("db.Begin: %s", err)
		return nil
	}
	log.Tracef("db.Begin")
	return tx
}

func unitName(id int) []byte {
	t := time.Unix(int64(id)*60*60, 0)
	s := fmt.Sprintf("%04d%02d%02d%02d", t.Year(), t.Month(), t.Day(), t.Hour())
	return []byte(s)
}

// itob returns an 8-byte big endian representation of v
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (s *Stats) periodicFlush() {
	for s.unit != nil {
		id := unitID()
		if s.unit.id == id {
			time.Sleep(time.Second)
			continue
		}

		nu := unit{}
		nu.id = id
		s.initUnit(&nu)
		u := s.swapUnit(&nu)
		s.flushUnitToDB(u)
	}
	log.Tracef("periodicFlush() exited")
}

func convertMapToArray(m map[string]int, max int) []countPair {
	a := []countPair{}
	for k, v := range m {
		pair := countPair{}
		pair.Name = k
		pair.Count = uint(v)
		a = append(a, pair)
	}
	less := func(i, j int) bool {
		if a[i].Count < a[j].Count {
			return true
		}
		return false
	}
	sort.Slice(a, less)
	if max > len(a) {
		max = len(a)
	}
	return a[:max]
}

func convertArrayToMap(a []countPair) map[string]int {
	m := map[string]int{}
	for _, it := range a {
		m[it.Name] = int(it.Count)
	}
	return m
}

func serialize(u *unit) *unitDB {
	u.timeAvg = u.timeSum / u.nTotal
	udb := unitDB{}
	udb.NTotal = uint(u.nTotal)
	udb.NResult = u.nResult
	udb.TimeAvg = uint(u.timeAvg)
	udb.Domains = convertMapToArray(u.domains, maxDomains)
	udb.BlockedDomains = convertMapToArray(u.blockedDomains, maxDomains)
	udb.Clients = convertMapToArray(u.clients, maxClients)
	return &udb
}

func deserialize(u *unit, udb *unitDB) {
	u.nTotal = int(udb.NTotal)
	u.nResult = udb.NResult
	u.domains = convertArrayToMap(udb.Domains)
	u.blockedDomains = convertArrayToMap(udb.BlockedDomains)
	u.clients = convertArrayToMap(udb.Clients)
	u.timeSum = int(udb.TimeAvg) * u.nTotal
}

func (s *Stats) flushUnitToDB(u *unit) {
	log.Tracef("Flushing unit %d", u.id)

	udb := serialize(u)

	tx := s.dbBeginTxn(true)
	if tx == nil {
		return
	}
	defer tx.Rollback()

	bkt, err := tx.CreateBucketIfNotExists(unitName(u.id))
	if err != nil {
		log.Error("tx.CreateBucketIfNotExists: %s", err)
		return
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(udb)
	if err != nil {
		log.Error("gob.Encode: %s", err)
		return
	}

	err = bkt.Put([]byte{0}, buf.Bytes())
	if err != nil {
		log.Error("bkt.Put: %s", err)
		return
	}

	log.Tracef("tx.Commit...")
	tx.Commit()
	log.Tracef("tx.Commit")
}

func (s *Stats) loadUnitFromDB(id int) *unitDB {
	log.Tracef("Loading unit %d", id)
	tx := s.dbBeginTxn(false)
	if tx == nil {
		return nil
	}
	defer tx.Rollback()

	bkt := tx.Bucket(unitName(id))
	if bkt == nil {
		return nil
	}

	var buf bytes.Buffer
	buf.Write(bkt.Get([]byte{0}))
	dec := gob.NewDecoder(&buf)
	udb := unitDB{}
	err := dec.Decode(&udb)
	if err != nil {
		log.Error("gob Decode: %s", err)
		return nil
	}

	return &udb
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
	client := e.Client.String()

	s.unitLock.Lock()
	u := s.unit

	u.nResult[e.Result]++

	if e.Result == NF {
		u.domains[e.Domain]++
	} else {
		u.blockedDomains[e.Domain]++
	}

	u.clients[client]++
	u.timeSum += int(e.Time)
	u.nTotal++
	s.unitLock.Unlock()
}

// Get - get data
func (s *Stats) Get() map[string]interface{} {
	d := map[string]interface{}{}

	d["time_units"] = "hours"

	u := s.unit
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
