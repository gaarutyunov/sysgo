package salsa

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// Revision is a monotonic counter bumped each time an input actually changes.
type Revision uint64

// Db is an incremental query database. The zero value is not usable; create one
// with [New]. A Db is safe for concurrent use.
type Db struct {
	mu       sync.Mutex
	revision Revision
	slots    map[string]*slot
}

// New returns an empty database at revision 0.
func New() *Db {
	return &Db{slots: make(map[string]*slot)}
}

// slot is the stored state for one input or one memoized query invocation.
type slot struct {
	isInput    bool
	hasValue   bool
	value      any
	changedAt  Revision // revision at which value last changed
	verifiedAt Revision // revision at which value was last confirmed up to date
	computing  bool     // on the active stack — used for cycle detection
	deps       []string // dependency keys (derived slots only)
	compute    func(*Ctx) any
}

// Ctx is the handle passed to query compute functions and to top-level reads
// via [Db.Read]. Reads performed through a Ctx are dependency-tracked.
type Ctx struct {
	db    *Db
	ctx   context.Context
	frame *frame
}

// Context returns the context governing this read.
func (c *Ctx) Context() context.Context { return c.ctx }

// frame records, in order, the dependency keys read while computing one query.
type frame struct {
	order []string
	seen  map[string]bool
}

func newFrame() *frame { return &frame{seen: make(map[string]bool)} }

func (f *frame) add(key string) {
	if !f.seen[key] {
		f.seen[key] = true
		f.order = append(f.order, key)
	}
}

// CycleError reports that a query transitively depended on itself.
type CycleError struct{ Key string }

func (e CycleError) Error() string { return "salsa: dependency cycle at " + e.Key }

// canceled unwinds a cancelled read to the enclosing [Db.Read].
type canceled struct{ err error }

// Read runs fn under the database lock with a root [Ctx] and reports any error
// raised during it: the context's error if ctx was cancelled, or a [CycleError]
// if a dependency cycle was hit. If ctx is nil, context.Background is used.
func (db *Db) Read(ctx context.Context, fn func(*Ctx)) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	defer func() {
		switch r := recover().(type) {
		case nil:
		case CycleError:
			err = r
		case canceled:
			err = r.err
		default:
			panic(r)
		}
	}()
	fn(&Ctx{db: db, ctx: ctx})
	return nil
}

// Revision returns the current global revision.
func (db *Db) Revision() Revision {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.revision
}

// getDerived resolves a memoized query slot, computing or re-verifying it as
// needed, and returns its value. The caller must hold db.mu (via Read).
func (db *Db) getDerived(c *Ctx, key string, compute func(*Ctx) any) any {
	s := db.slots[key]
	if s == nil {
		s = &slot{compute: compute}
		db.slots[key] = s
	} else {
		s.compute = compute
	}
	db.ensureFresh(c, s, key)
	return s.value
}

// ensureFresh makes s up to date for the current revision, recomputing it only
// if a dependency changed since it was last verified.
func (db *Db) ensureFresh(c *Ctx, s *slot, key string) {
	if s.isInput {
		return
	}
	if s.hasValue && s.verifiedAt == db.revision {
		return
	}
	if s.computing {
		panic(CycleError{Key: key})
	}
	if s.hasValue && !db.anyDepChanged(c, s) {
		s.verifiedAt = db.revision
		return
	}

	s.computing = true
	child := &Ctx{db: db, ctx: c.ctx, frame: newFrame()}
	newVal := s.compute(child)
	s.computing = false

	s.deps = child.frame.order
	s.verifiedAt = db.revision
	if !s.hasValue || !reflect.DeepEqual(s.value, newVal) {
		s.changedAt = db.revision
	}
	s.value = newVal
	s.hasValue = true
}

// anyDepChanged reports whether any dependency of s changed after s was last
// verified, freshening each derived dependency first.
func (db *Db) anyDepChanged(c *Ctx, s *slot) bool {
	for _, dep := range s.deps {
		ds := db.slots[dep]
		if ds == nil {
			return true // dependency vanished
		}
		db.ensureFresh(c, ds, dep)
		if ds.changedAt > s.verifiedAt {
			return true
		}
	}
	return false
}

func checkCancel(c *Ctx) {
	if err := c.ctx.Err(); err != nil {
		panic(canceled{err: err})
	}
}

var idCounter int64

func freshID(name string) string {
	return fmt.Sprintf("%s#%d", name, atomic.AddInt64(&idCounter, 1))
}

func keyStr[K comparable](k K) string { return fmt.Sprintf("%v", k) }

// Input is a durable, settable base fact keyed by K with value V.
type Input[K comparable, V any] struct{ id string }

// NewInput creates a fresh input group. name is for diagnostics only; distinct
// inputs never collide even with the same name.
func NewInput[K comparable, V any](name string) *Input[K, V] {
	return &Input[K, V]{id: freshID(name)}
}

func (in *Input[K, V]) key(k K) string { return in.id + "\x00" + keyStr(k) }

// Set stores v for key k, bumping the revision only if the value actually
// changed.
func (in *Input[K, V]) Set(db *Db, k K, v V) {
	db.mu.Lock()
	defer db.mu.Unlock()
	key := in.key(k)
	s := db.slots[key]
	if s != nil && s.hasValue && reflect.DeepEqual(s.value, v) {
		return
	}
	db.revision++
	if s == nil {
		s = &slot{isInput: true}
		db.slots[key] = s
	}
	s.isInput = true
	s.value = v
	s.hasValue = true
	s.changedAt = db.revision
}

// Get returns the value set for k, or the zero value if unset. The read is
// recorded as a dependency of the enclosing query so a later Set invalidates it.
func (in *Input[K, V]) Get(c *Ctx, k K) V {
	checkCancel(c)
	key := in.key(k)
	if c.frame != nil {
		c.frame.add(key)
	}
	s := c.db.slots[key]
	if s == nil {
		// Materialize an empty input slot so the dependency exists and a future
		// Set can invalidate readers.
		s = &slot{isInput: true}
		c.db.slots[key] = s
	}
	if !s.hasValue {
		var zero V
		return zero
	}
	return s.value.(V)
}

// Query is a memoized, dependency-tracked function from K to V.
type Query[K comparable, V any] struct {
	id      string
	compute func(*Ctx, K) V
}

// NewQuery creates a tracked query. compute must be a pure function of the data
// it reads through the [Ctx]; it may read inputs and other queries.
func NewQuery[K comparable, V any](name string, compute func(*Ctx, K) V) *Query[K, V] {
	return &Query[K, V]{id: freshID(name), compute: compute}
}

func (q *Query[K, V]) key(k K) string { return q.id + "\x00" + keyStr(k) }

// Get returns the query's value for k, computing it on first use and reusing
// the memoized result while its dependencies are unchanged.
func (q *Query[K, V]) Get(c *Ctx, k K) V {
	checkCancel(c)
	key := q.key(k)
	if c.frame != nil {
		c.frame.add(key)
	}
	v := c.db.getDerived(c, key, func(child *Ctx) any { return q.compute(child, k) })
	return v.(V)
}
