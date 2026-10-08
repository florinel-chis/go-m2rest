package magento2

import (
	"strings"
	"testing"
)

// wire converts the readable bracket form into what url.Values.Encode emits.
func wire(q string) string {
	return strings.NewReplacer("[", "%5B", "]", "%5D").Replace(q)
}

func TestSearchCriteriaValues(t *testing.T) {
	tests := []struct {
		name    string
		build   func() *SearchCriteria
		want    string // readable form; "" means no keys at all
		wantErr string
	}{
		{
			name:  "empty builder emits nothing",
			build: func() *SearchCriteria { return NewSearchCriteria().RestrictFields("state") },
			want:  "",
		},
		{
			name:  "empty builder without allowlist emits nothing",
			build: NewSearchCriteria,
			want:  "",
		},
		{
			name: "one filter",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").Filter("state", Eq, "holded")
			},
			want: "searchCriteria[filter_groups][0][filters][0][condition_type]=eq" +
				"&searchCriteria[filter_groups][0][filters][0][field]=state" +
				"&searchCriteria[filter_groups][0][filters][0][value]=holded",
		},
		{
			name: "two filters OR within one group",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").
					Filter("state", Eq, "holded").Or().Filter("state", Eq, "canceled")
			},
			want: "searchCriteria[filter_groups][0][filters][0][condition_type]=eq" +
				"&searchCriteria[filter_groups][0][filters][0][field]=state" +
				"&searchCriteria[filter_groups][0][filters][0][value]=holded" +
				"&searchCriteria[filter_groups][0][filters][1][condition_type]=eq" +
				"&searchCriteria[filter_groups][0][filters][1][field]=state" +
				"&searchCriteria[filter_groups][0][filters][1][value]=canceled",
		},
		{
			name: "two groups AND",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state", "created_at").
					Filter("state", Eq, "holded").And().Filter("created_at", Gteq, "2025-08-01 00:00:00")
			},
			want: "searchCriteria[filter_groups][0][filters][0][condition_type]=eq" +
				"&searchCriteria[filter_groups][0][filters][0][field]=state" +
				"&searchCriteria[filter_groups][0][filters][0][value]=holded" +
				"&searchCriteria[filter_groups][1][filters][0][condition_type]=gteq" +
				"&searchCriteria[filter_groups][1][filters][0][field]=created_at" +
				"&searchCriteria[filter_groups][1][filters][0][value]=2025-08-01+00%3A00%3A00",
		},
		{
			name: "consecutive filters without Or are AND (each in its own group)",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state", "store_id").
					Filter("state", Eq, "holded").Filter("store_id", Eq, "1")
			},
			want: "searchCriteria[filter_groups][0][filters][0][condition_type]=eq" +
				"&searchCriteria[filter_groups][0][filters][0][field]=state" +
				"&searchCriteria[filter_groups][0][filters][0][value]=holded" +
				"&searchCriteria[filter_groups][1][filters][0][condition_type]=eq" +
				"&searchCriteria[filter_groups][1][filters][0][field]=store_id" +
				"&searchCriteria[filter_groups][1][filters][0][value]=1",
		},
		{
			name: "sort",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("created_at").Sort("created_at", Desc)
			},
			want: "searchCriteria[sortOrders][0][direction]=DESC&searchCriteria[sortOrders][0][field]=created_at",
		},
		{
			name: "several sorts keep their order",
			build: func() *SearchCriteria {
				return NewSearchCriteria().Sort("created_at", Desc).Sort("entity_id", Asc)
			},
			want: "searchCriteria[sortOrders][0][direction]=DESC&searchCriteria[sortOrders][0][field]=created_at" +
				"&searchCriteria[sortOrders][1][direction]=ASC&searchCriteria[sortOrders][1][field]=entity_id",
		},
		{
			name: "paging",
			build: func() *SearchCriteria {
				return NewSearchCriteria().Page(50, 2)
			},
			want: "searchCriteria[currentPage]=2&searchCriteria[pageSize]=50",
		},
		{
			name: "paging is clamped to [1,300] and page >= 1",
			build: func() *SearchCriteria {
				return NewSearchCriteria().Page(5000, 0)
			},
			want: "searchCriteria[currentPage]=1&searchCriteria[pageSize]=300",
		},
		{
			name: "page size below one becomes one",
			build: func() *SearchCriteria {
				return NewSearchCriteria().Page(-3, 1)
			},
			want: "searchCriteria[currentPage]=1&searchCriteria[pageSize]=1",
		},
		{
			name: "without an allowlist any field passes",
			build: func() *SearchCriteria {
				return NewSearchCriteria().Filter("customer_email", Like, "%@example.com")
			},
			want: "searchCriteria[filter_groups][0][filters][0][condition_type]=like" +
				"&searchCriteria[filter_groups][0][filters][0][field]=customer_email" +
				"&searchCriteria[filter_groups][0][filters][0][value]=%25%40example.com",
		},
		{
			name: "unknown filter field",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").Filter("customer_email", Eq, "x")
			},
			wantErr: `field "customer_email"`,
		},
		{
			name: "allowlist set after the filter still applies",
			build: func() *SearchCriteria {
				return NewSearchCriteria().Filter("customer_email", Eq, "x").RestrictFields("state")
			},
			wantErr: `field "customer_email"`,
		},
		{
			name: "unknown sort field",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").Sort("entity_id", Asc)
			},
			wantErr: `field "entity_id"`,
		},
		{
			name: "empty field without an allowlist",
			build: func() *SearchCriteria {
				return NewSearchCriteria().Filter("", Eq, "x")
			},
			wantErr: "empty field",
		},
		{
			name: "unknown condition",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").Filter("state", Cond("equals"), "holded")
			},
			wantErr: `condition "equals"`,
		},
		{
			name: "injected field with searchCriteria syntax is not a field",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").Filter("state][0][value]=x&searchCriteria[filter_groups", Eq, "y")
			},
			wantErr: "field",
		},
		{
			name: "FilterIn joins with commas",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").FilterIn("state", "holded", "canceled")
			},
			want: "searchCriteria[filter_groups][0][filters][0][condition_type]=in" +
				"&searchCriteria[filter_groups][0][filters][0][field]=state" +
				"&searchCriteria[filter_groups][0][filters][0][value]=holded%2Ccanceled",
		},
		{
			name: "FilterIn rejects a value containing a comma",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").FilterIn("state", "holded", "a,b")
			},
			wantErr: "comma",
		},
		{
			name: "FilterIn rejects an empty list",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").FilterIn("state")
			},
			wantErr: "no values",
		},
		{
			name: "unknown sort direction",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").Sort("state", SortDir("sideways"))
			},
			wantErr: `direction "sideways"`,
		},
		{
			name: "Or before any filter",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").Or().Filter("state", Eq, "holded")
			},
			wantErr: "Or()",
		},
		{
			name: "every error is reported",
			build: func() *SearchCriteria {
				return NewSearchCriteria().RestrictFields("state").Filter("a", Eq, "x").Sort("b", Asc)
			},
			wantErr: `field "b"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.build().Values()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Values() error = %v, want one containing %q", err, tt.wantErr)
				}
				if got != nil {
					t.Fatalf("Values() returned %v alongside an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Values() error = %v", err)
			}
			if enc := got.Encode(); enc != wire(tt.want) {
				t.Fatalf("Values().Encode() =\n  %s\nwant\n  %s", enc, wire(tt.want))
			}
		})
	}
}

func TestSearchCriteriaEveryConditionIsAccepted(t *testing.T) {
	conds := map[Cond]string{
		Eq: "eq", Neq: "neq", Gt: "gt", Gteq: "gteq", Lt: "lt",
		Lteq: "lteq", Like: "like", Nlike: "nlike", In: "in", Nin: "nin",
		From: "from", To: "to", Finset: "finset", Null: "null",
		NotNull: "notnull", Moreq: "moreq",
	}
	if len(conds) != 16 {
		t.Fatalf("%d conditions, want 16", len(conds))
	}
	for cond, wire := range conds {
		t.Run(wire, func(t *testing.T) {
			v, err := NewSearchCriteria().RestrictFields("f").Filter("f", cond, "x").Values()
			if err != nil {
				t.Fatalf("Values() error = %v", err)
			}
			if got := v.Get("searchCriteria[filter_groups][0][filters][0][condition_type]"); got != wire {
				t.Fatalf("condition_type = %q, want %q", got, wire)
			}
		})
	}
}

// Values must not mutate the builder: two calls agree, and a caller cannot
// reach the internal state through the returned map.
func TestSearchCriteriaValuesIsRepeatable(t *testing.T) {
	sc := NewSearchCriteria().RestrictFields("state").Filter("state", Eq, "holded").Page(1, 1)
	a, err := sc.Values()
	if err != nil {
		t.Fatalf("first Values(): %v", err)
	}
	a.Set("searchCriteria[pageSize]", "999")
	b, err := sc.Values()
	if err != nil {
		t.Fatalf("second Values(): %v", err)
	}
	if b.Get("searchCriteria[pageSize]") != "1" {
		t.Fatalf("second Values() saw the caller's edit: %v", b)
	}
}

func TestPageSizeLimit(t *testing.T) {
	tests := []struct {
		message string
		want    int
		wantOK  bool
	}{
		{message: "Maximum SearchCriteria pageSize is 300", want: 300, wantOK: true},
		{message: "Invalid request: Maximum SearchCriteria pageSize is 50.", want: 50, wantOK: true},
		{message: "Maximum SearchCriteria pageSize is 0"},
		{message: "Maximum SearchCriteria pageSize is %max"},
		{message: "Maximum SearchCriteria pageSize is 99999999999999999999999"},
		{message: "The entity that was requested doesn't exist."},
		{message: ""},
	}
	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			n, ok := PageSizeLimit(tt.message)
			if n != tt.want || ok != tt.wantOK {
				t.Fatalf("PageSizeLimit(%q) = %d, %v; want %d, %v", tt.message, n, ok, tt.want, tt.wantOK)
			}
		})
	}
}
