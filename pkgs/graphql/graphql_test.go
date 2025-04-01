package graphql

import (
	"testing"
)

func TestParseGraphQLRequest(t *testing.T) {

}

func TestRequestParse1(t *testing.T) {
	testCases := []struct {
		desc string
		req  *Request
		op   Operation
		name string
		err  bool
	}{
		{
			desc: "Parse single unamed operation",
			req: &Request{
				Query: "query { hello }",
			},
			op:   Query,
			name: "",
			err:  false,
		},
		{
			desc: "Parse single named operation",
			req: &Request{
				Query:         "query yo{ hello }",
				OperationName: "yo",
			},
			op:   Query,
			name: "yo",
			err:  false,
		},
		{
			desc: "Parse single unamed operation with operation name",
			req: &Request{
				Query:         "query { hello }",
				OperationName: "yo",
			},
			err: true,
		},
		{
			desc: "Parse single named operation with wrong operation name",
			req: &Request{
				Query:         "query yo1{ hello }",
				OperationName: "yo",
			},
			err: true,
		},
		{
			desc: "Parse multiple named operation without operation name",
			req: &Request{
				Query: "query yo1{ hello } query yo12{ hello }",
			},
			err: true,
		},
		{
			desc: "Parse multiple named operation with operation name",
			req: &Request{
				Query:         "query yo1{ hello } query yo12{ hello }",
				OperationName: "yo12",
			},
			op:   Query,
			name: "yo12",
			err:  false,
		},
		{
			desc: "Parse multiple named operation with operation name",
			req: &Request{
				Query:         "query yo1{ hello } query yo12{ hello } mutation yo123{ hello }",
				OperationName: "yo123",
			},
			op:   Mutation,
			name: "yo123",
			err:  false,
		},
		{
			desc: "Parse multiple named operation with wrong operation name",
			req: &Request{
				Query:         "query yo1{ hello } query yo12{ hello } mutation yo123{ hello }",
				OperationName: "yo1234",
			},
			err: true,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {

			op, name, err := tC.req.Parse()
			if (tC.err && err == nil) || (!tC.err && err != nil) {
				t.Fatalf("expected error: %v, got: %v", tC.err, err)
			}
			if err != nil && !tC.err {
				t.Errorf("expected error, got %v", err)
			}

			if op != tC.op {
				t.Errorf("expected %v, got %v", tC.op, op)
			}

			if name != tC.name {
				t.Errorf("expected %v, got %v", tC.name, name)
			}
		})
	}
}
