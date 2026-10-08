package magento2

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SortDir is a searchCriteria sort direction.
type SortDir string

// The two directions Magento accepts.
const (
	Asc  SortDir = "ASC"
	Desc SortDir = "DESC"
)

// Cond is a searchCriteria condition_type. Only the constants below pass
// Values; a Cond built from any other string is refused there.
type Cond string

// The condition types Magento's collection layer understands. Like/Nlike
// need explicit % wildcards in the value; In/Nin take a comma-separated
// value (see FilterIn).
const (
	Eq      Cond = "eq"
	Neq     Cond = "neq"
	Gt      Cond = "gt"
	Gteq    Cond = "gteq"
	Lt      Cond = "lt"
	Lteq    Cond = "lteq"
	Like    Cond = "like"
	Nlike   Cond = "nlike"
	In      Cond = "in"
	Nin     Cond = "nin"
	From    Cond = "from"
	To      Cond = "to"
	Finset  Cond = "finset"
	Null    Cond = "null"
	NotNull Cond = "notnull"
	Moreq   Cond = "moreq"
)

// Page size bounds applied by SearchCriteria.Page. Adobe Commerce 2.4.7+
// with webapi/validation/input_limit_enabled does not clamp: a pageSize
// above webapi/validation/maximum_page_size (default 300) is answered with
// HTTP 400 "Maximum SearchCriteria pageSize is N" (see PageSizeLimit).
const (
	MinPageSize = 1
	MaxPageSize = 300
)

// pageSizeLimitRe parses the message of Magento's
// Framework\Webapi\Validator\SearchCriteriaValidator after substitution.
var pageSizeLimitRe = regexp.MustCompile(`Maximum SearchCriteria pageSize is (\d+)`)

// PageSizeLimit reads N out of Magento's input-limit refusal "Maximum
// SearchCriteria pageSize is N" (pass APIError.Message). ok is false when
// message is not that refusal or N is not a positive number.
func PageSizeLimit(message string) (n int, ok bool) {
	m := pageSizeLimitRe.FindStringSubmatch(message)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

var conditions = map[Cond]bool{
	Eq: true, Neq: true, Gt: true, Gteq: true, Lt: true, Lteq: true, Like: true, Nlike: true,
	In: true, Nin: true, From: true, To: true, Finset: true, Null: true, NotNull: true, Moreq: true,
}

type scFilter struct {
	field, value string
	cond         Cond
}

type scSort struct {
	field string
	dir   SortDir
}

// SearchCriteria builds Magento's searchCriteria query parameters. Filters
// inside one group are OR; groups are AND. The API makes that explicit:
//
//	Filter(a).Filter(b)        → a AND b   (each Filter opens a new group by default)
//	Filter(a).Or().Filter(b)   → a OR b    (Or keeps the next filter in the current group)
//	Filter(a).And().Filter(b)  → a AND b   (And says so explicitly)
//
// Mistakes are collected and returned together by Values, so the chain
// never has to carry an error:
//
//	q, err := magento2.NewSearchCriteria().
//		Filter("status", magento2.Eq, "pending").
//		Sort("created_at", magento2.Desc).
//		Page(100, 1).
//		Values()
//	resp, err := client.Do(ctx, magento2.Request{Path: "/V1/orders", Query: q})
type SearchCriteria struct {
	allowed   map[string]bool // nil = every non-empty field
	groups    [][]scFilter
	sorts     []scSort
	pageSize  int
	page      int
	paged     bool
	sameGroup bool
	errs      []error
}

// NewSearchCriteria returns an empty builder.
func NewSearchCriteria() *SearchCriteria {
	return &SearchCriteria{}
}

// RestrictFields sets an allowlist: a filter or sort on any other field is
// an error at Values, whether it was added before or after this call.
// Calling it again adds to the list.
func (s *SearchCriteria) RestrictFields(fields ...string) *SearchCriteria {
	if s.allowed == nil {
		s.allowed = make(map[string]bool, len(fields))
	}
	for _, f := range fields {
		s.allowed[f] = true
	}
	return s
}

// Filter adds field <cond> value. Without a preceding Or it opens a new
// filter group, so consecutive filters narrow (AND).
func (s *SearchCriteria) Filter(field string, cond Cond, value string) *SearchCriteria {
	if !conditions[cond] {
		s.errs = append(s.errs, fmt.Errorf("searchCriteria: unknown condition %q", string(cond)))
	}
	if len(s.groups) == 0 || !s.sameGroup {
		s.groups = append(s.groups, nil)
	}
	last := len(s.groups) - 1
	s.groups[last] = append(s.groups[last], scFilter{field: field, cond: cond, value: value})
	s.sameGroup = false
	return s
}

// FilterIn adds field IN values, joined with the comma Magento splits on. A
// value that itself contains a comma cannot be expressed and is an error, as
// is an empty list.
func (s *SearchCriteria) FilterIn(field string, values ...string) *SearchCriteria {
	if len(values) == 0 {
		s.errs = append(s.errs, fmt.Errorf("searchCriteria: FilterIn(%q) has no values", field))
	}
	for _, v := range values {
		if strings.Contains(v, ",") {
			s.errs = append(s.errs, fmt.Errorf("searchCriteria: FilterIn(%q) value %q contains a comma", field, v))
		}
	}
	return s.Filter(field, In, strings.Join(values, ","))
}

// Or keeps the next Filter in the current group, making it an alternative
// (OR) to the filters already there. It must follow a Filter.
func (s *SearchCriteria) Or() *SearchCriteria {
	if len(s.groups) == 0 {
		s.errs = append(s.errs, errors.New("searchCriteria: Or() before any Filter"))
	}
	s.sameGroup = true
	return s
}

// And opens a new group for the next Filter, which is also what a bare
// Filter does; it exists so the intent reads at the call site.
func (s *SearchCriteria) And() *SearchCriteria {
	s.sameGroup = false
	return s
}

// Sort appends a sort order. Sorts apply in the order they were added.
func (s *SearchCriteria) Sort(field string, dir SortDir) *SearchCriteria {
	if dir != Asc && dir != Desc {
		s.errs = append(s.errs, fmt.Errorf("searchCriteria: unknown sort direction %q", string(dir)))
	}
	s.sorts = append(s.sorts, scSort{field: field, dir: dir})
	return s
}

// Page sets pageSize and currentPage. size is clamped to [MinPageSize,
// MaxPageSize] and current to >= 1. A store configured with a lower maximum
// answers 400 (see PageSizeLimit).
func (s *SearchCriteria) Page(size, current int) *SearchCriteria {
	s.pageSize = min(max(size, MinPageSize), MaxPageSize)
	s.page = max(current, 1)
	s.paged = true
	return s
}

// Values renders the wire format. An empty builder yields an empty map (no
// searchCriteria keys at all). An empty field, a field outside the
// allowlist, an unknown condition or direction, or Or() without a preceding
// Filter is returned here as an error (all of them, joined).
func (s *SearchCriteria) Values() (url.Values, error) {
	errs := append([]error(nil), s.errs...)
	for _, group := range s.groups {
		for _, f := range group {
			if err := s.checkField(f.field); err != nil {
				errs = append(errs, err)
			}
		}
	}
	for _, so := range s.sorts {
		if err := s.checkField(so.field); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	v := url.Values{}
	for g, group := range s.groups {
		for f, flt := range group {
			key := "searchCriteria[filter_groups][" + strconv.Itoa(g) + "][filters][" + strconv.Itoa(f) + "]"
			v.Set(key+"[field]", flt.field)
			v.Set(key+"[value]", flt.value)
			v.Set(key+"[condition_type]", string(flt.cond))
		}
	}
	for n, so := range s.sorts {
		key := "searchCriteria[sortOrders][" + strconv.Itoa(n) + "]"
		v.Set(key+"[field]", so.field)
		v.Set(key+"[direction]", string(so.dir))
	}
	if s.paged {
		v.Set("searchCriteria[pageSize]", strconv.Itoa(s.pageSize))
		v.Set("searchCriteria[currentPage]", strconv.Itoa(s.page))
	}
	return v, nil
}

func (s *SearchCriteria) checkField(field string) error {
	if s.allowed != nil {
		if !s.allowed[field] {
			return fmt.Errorf("searchCriteria: field %q is not in the allowlist", field)
		}
		return nil
	}
	if field == "" {
		return errors.New("searchCriteria: empty field")
	}
	return nil
}
