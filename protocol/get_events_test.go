package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
)

func TestEventTypeSetMatches(t *testing.T) {
	var defaultSet EventTypeSet
	all := EventTypeSet{}
	all[EventTypeContract] = nil
	all[EventTypeDiagnostic] = nil
	all[EventTypeSystem] = nil

	onlyContract := EventTypeSet{}
	onlyContract[EventTypeContract] = nil

	contractEvent := xdr.ContractEvent{Type: xdr.ContractEventTypeContract}
	diagnosticEvent := xdr.ContractEvent{Type: xdr.ContractEventTypeDiagnostic}
	systemEvent := xdr.ContractEvent{Type: xdr.ContractEventTypeSystem}

	for _, testCase := range []struct {
		name    string
		set     EventTypeSet
		event   xdr.ContractEvent
		matches bool
	}{
		{
			"all matches Contract events",
			all,
			contractEvent,
			true,
		},
		{
			"all matches System events",
			all,
			systemEvent,
			true,
		},
		{
			"all matches Diagnostic events",
			all,
			systemEvent,
			true,
		},
		{
			"defaultSet matches Contract events",
			defaultSet,
			contractEvent,
			true,
		},
		{
			"defaultSet matches System events",
			defaultSet,
			systemEvent,
			true,
		},
		{
			"defaultSet matches Diagnostic events",
			defaultSet,
			systemEvent,
			true,
		},
		{
			"onlyContract set matches Contract events",
			onlyContract,
			contractEvent,
			true,
		},
		{
			"onlyContract does not match System events",
			onlyContract,
			systemEvent,
			false,
		},
		{
			"onlyContract does not match Diagnostic events",
			defaultSet,
			diagnosticEvent,
			true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.matches, testCase.set.matches(testCase.event))
		})
	}
}

func TestEventTypeSetValid(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		keys          []string
		expectedError bool
	}{
		{
			"empty set",
			[]string{},
			false,
		},
		{
			"set with one valid element",
			[]string{EventTypeSystem},
			false,
		},
		{
			"set with two valid elements",
			[]string{EventTypeSystem, EventTypeContract},
			false,
		},
		{
			"set with three valid elements",
			[]string{EventTypeSystem, EventTypeContract, EventTypeDiagnostic},
			false,
		},
		{
			"set with one invalid element",
			[]string{"abc"},
			true,
		},
		{
			"set with multiple invalid elements",
			[]string{"abc", "def"},
			true,
		},
		{
			"set with valid elements mixed with invalid elements",
			[]string{EventTypeSystem, "abc"},
			true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			set := EventTypeSet{}
			for _, key := range testCase.keys {
				set[key] = nil
			}
			if testCase.expectedError {
				assert.Error(t, set.valid())
			} else {
				require.NoError(t, set.valid())
			}
		})
	}
}

func TestEventTypeSetMarshaling(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		input    string
		expected []string
	}{
		{
			"empty set",
			"",
			[]string{},
		},
		{
			"set with one element",
			"a",
			[]string{"a"},
		},
		{
			"set with more than one element",
			"a,b,c",
			[]string{"a", "b", "c"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var set EventTypeSet
			input, err := json.Marshal(testCase.input)
			require.NoError(t, err)
			err = set.UnmarshalJSON(input)
			require.NoError(t, err)
			assert.Equal(t, len(testCase.expected), len(set))
			for _, val := range testCase.expected {
				_, ok := set[val]
				assert.True(t, ok)
			}
		})
	}
}

//nolint:funlen
func TestTopicFilterMatches(t *testing.T) {
	transferSym := xdr.ScSymbol("transfer")
	transfer := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &transferSym,
	}
	sixtyfour := xdr.Uint64(64)
	number := xdr.ScVal{
		Type: xdr.ScValTypeScvU64,
		U64:  &sixtyfour,
	}
	star := "*"
	for _, tc := range []struct {
		name     string
		filter   TopicFilter
		includes []xdr.ScVec
		excludes []xdr.ScVec
	}{
		{
			name:   "<empty>",
			filter: nil,
			includes: []xdr.ScVec{
				{},
			},
			excludes: []xdr.ScVec{
				{transfer},
			},
		},

		// Exact matching
		{
			name: "ScSymbol(transfer)",
			filter: []SegmentFilter{
				{ScVal: &transfer},
			},
			includes: []xdr.ScVec{
				{transfer},
			},
			excludes: []xdr.ScVec{
				{number},
				{transfer, transfer},
			},
		},

		// Star
		{
			name: "*",
			filter: []SegmentFilter{
				{Wildcard: &star},
			},
			includes: []xdr.ScVec{
				{transfer},
			},
			excludes: []xdr.ScVec{
				{transfer, transfer},
			},
		},
		{
			name: "*/transfer",
			filter: []SegmentFilter{
				{Wildcard: &star},
				{ScVal: &transfer},
			},
			includes: []xdr.ScVec{
				{number, transfer},
				{transfer, transfer},
			},
			excludes: []xdr.ScVec{
				{number},
				{number, number},
				{number, transfer, number},
				{transfer},
				{transfer, number},
				{transfer, transfer, transfer},
			},
		},
		{
			name: "transfer/*",
			filter: []SegmentFilter{
				{ScVal: &transfer},
				{Wildcard: &star},
			},
			includes: []xdr.ScVec{
				{transfer, number},
				{transfer, transfer},
			},
			excludes: []xdr.ScVec{
				{number},
				{number, number},
				{number, transfer, number},
				{transfer},
				{number, transfer},
				{transfer, transfer, transfer},
			},
		},
		{
			name: "transfer/*/*",
			filter: []SegmentFilter{
				{ScVal: &transfer},
				{Wildcard: &star},
				{Wildcard: &star},
			},
			includes: []xdr.ScVec{
				{transfer, number, number},
				{transfer, transfer, transfer},
			},
			excludes: []xdr.ScVec{
				{number},
				{number, number},
				{number, transfer},
				{number, transfer, number, number},
				{transfer},
				{transfer, transfer, transfer, transfer},
			},
		},
		{
			name: "transfer/*/number",
			filter: []SegmentFilter{
				{ScVal: &transfer},
				{Wildcard: &star},
				{ScVal: &number},
			},
			includes: []xdr.ScVec{
				{transfer, number, number},
				{transfer, transfer, number},
			},
			excludes: []xdr.ScVec{
				{number},
				{number, number},
				{number, number, number},
				{number, transfer, number},
				{transfer},
				{number, transfer},
				{transfer, transfer, transfer},
				{transfer, number, transfer},
			},
		},
	} {
		name := tc.name
		if name == "" {
			name = topicFilterToString(tc.filter)
		}
		t.Run(name, func(t *testing.T) {
			for _, include := range tc.includes {
				assert.True(
					t,
					tc.filter.Matches(include),
					"Expected %v filter to include %v",
					name,
					include,
				)
			}
			for _, exclude := range tc.excludes {
				assert.False(
					t,
					tc.filter.Matches(exclude),
					"Expected %v filter to exclude %v",
					name,
					exclude,
				)
			}
		})
	}
}

func TestTopicFilterJSON(t *testing.T) {
	var got TopicFilter

	require.NoError(t, json.Unmarshal([]byte("[]"), &got))
	assert.Equal(t, TopicFilter{}, got)

	star := "*"
	require.NoError(t, json.Unmarshal([]byte("[\"*\"]"), &got))
	assert.Equal(t, TopicFilter{{Wildcard: &star}}, got)

	sixtyfour := xdr.Uint64(64)
	scval := xdr.ScVal{Type: xdr.ScValTypeScvU64, U64: &sixtyfour}
	scvalstr, err := xdr.MarshalBase64(scval)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(fmt.Appendf(nil, "[%q]", scvalstr), &got))
	assert.Equal(t, TopicFilter{{ScVal: &scval}}, got)
}

func topicFilterToString(t TopicFilter) string {
	var s []string
	for _, segment := range t {
		switch {
		case segment.Wildcard != nil:
			s = append(s, *segment.Wildcard)
		case segment.ScVal != nil:
			out, err := xdr.MarshalBase64(*segment.ScVal)
			if err != nil {
				panic(err)
			}
			s = append(s, out)
		default:
			panic("Invalid topic filter")
		}
	}
	if len(s) == 0 {
		s = append(s, "<empty>")
	}
	return strings.Join(s, "/")
}

//nolint:funlen
func TestGetEventsRequestValid(t *testing.T) {
	// omit startLedger but include cursor
	var request GetEventsRequest
	require.NoError(t, json.Unmarshal(
		[]byte("{ \"filters\": [], \"pagination\": { \"cursor\": \"0000000021474840576-0000000000\"} }"),
		&request,
	))
	assert.Equal(t, uint32(0), request.StartLedger)
	require.NoError(t, request.Valid(1000))

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters:     []EventFilter{},
		Pagination:  &PaginationOptions{Cursor: &Cursor{}},
	}).Valid(1000), "ledger ranges and cursor cannot both be set")

	require.NoError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters:     []EventFilter{},
		Pagination:  nil,
	}).Valid(1000))

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters:     []EventFilter{},
		Pagination:  &PaginationOptions{Limit: 1001},
	}).Valid(1000), "limit must not exceed 1000")

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 0,
		Filters:     []EventFilter{},
		Pagination:  nil,
	}).Valid(1000), "startLedger must be positive")

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters: []EventFilter{
			{}, {}, {}, {}, {}, {},
		},
		Pagination: nil,
	}).Valid(1000), "maximum 5 filters per request")

	err := (&GetEventsRequest{
		StartLedger: 1,
		Filters: []EventFilter{
			{EventType: map[string]interface{}{"foo": nil}},
		},
		Pagination: nil,
	}).Valid(1000)
	expectedErrStr := "filter 1 invalid: filter type invalid: if set, type must be either 'system', 'contract' or 'diagnostic'" //nolint:lll
	require.EqualError(t, err, expectedErrStr)

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters: []EventFilter{
			{ContractIDs: []string{
				"CCVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKUD2U",
				"CC53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO53WQD5",
				"CDGMZTGMZTGMZTGMZTGMZTGMZTGMZTGMZTGMZTGMZTGMZTGMZTGMZLND",
				"CDO53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO53YUK",
				"CDXO53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO53XO4M7R",
				"CD7777777777777777777777777777777777777777777777777767GY",
			}},
		},
		Pagination: nil,
	}).Valid(1000), "filter 1 invalid: maximum 5 contract IDs per filter")

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters: []EventFilter{
			{ContractIDs: []string{"a"}},
		},
		Pagination: nil,
	}).Valid(1000), "filter 1 invalid: contract ID 1 invalid")

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters: []EventFilter{
			{ContractIDs: []string{"CCVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVKVINVALID"}},
		},
		Pagination: nil,
	}).Valid(1000), "filter 1 invalid: contract ID 1 invalid")

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters: []EventFilter{
			{
				Topics: []TopicFilter{
					{}, {}, {}, {}, {}, {},
				},
			},
		},
		Pagination: nil,
	}).Valid(1000), "filter 1 invalid: maximum 5 topics per filter")

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters: []EventFilter{
			{Topics: []TopicFilter{
				{},
			}},
		},
		Pagination: nil,
	}).Valid(1000), "filter 1 invalid: topic 1 invalid: topic must have at least one segment")

	require.EqualError(t, (&GetEventsRequest{
		StartLedger: 1,
		Filters: []EventFilter{
			{Topics: []TopicFilter{
				{
					{},
					{},
					{},
					{},
					{},
				},
			}},
		},
		Pagination: nil,
	}).Valid(1000), "filter 1 invalid: topic 1 invalid: topic cannot have more than 4 segments")
}

func TestEventFilterSerialization(t *testing.T) {
	acct, err := xdr.AddressToAccountId(keypair.MustRandom().Address())
	require.NoError(t, err)

	scv := xdr.ScVal{
		Type: xdr.ScValTypeScvAddress,
		Address: &xdr.ScAddress{
			Type:      xdr.ScAddressTypeScAddressTypeAccount,
			AccountId: &acct,
		},
	}
	wc := "*"
	b64, err := xdr.MarshalBase64(scv)
	require.NoError(t, err)

	for _, testCase := range []struct {
		Filter  SegmentFilter
		Encoded string
	}{
		{SegmentFilter{Wildcard: &wc}, `"*"`},
		{SegmentFilter{ScVal: &scv}, fmt.Sprintf(`"%s"`, b64)},
	} {
		filter := EventFilter{Topics: []TopicFilter{{testCase.Filter}}}

		b, err := json.Marshal(testCase.Filter)
		require.NoError(t, err)
		require.JSONEq(t, testCase.Encoded, string(b))

		f, err := json.Marshal(filter)
		require.NoError(t, err)
		require.JSONEq(t, fmt.Sprintf(`{"topics":[[%s]]}`, string(b)), string(f))
	}

	_, err = json.Marshal(SegmentFilter{})
	require.Error(t, err)
}
