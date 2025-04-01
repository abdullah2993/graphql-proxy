package graphql

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

const (
	Query        Operation = "query"
	Mutation     Operation = "mutation"
	Subscription Operation = "subscription"
)

// Request represents a GraphQL request.
type Request struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// Operation represents a GraphQL operation. It can be one of Query, Mutation or Subscription.
type Operation = ast.Operation

// ParseQuery parses a GraphQL request. Return the type of the operation.
// if OperationName is present in query it returns the same name as Request.OperationName
// if OperationName is not preset in query it returns an error
// if OperationName is not specified and there are multiple operations in query it retruns an error
// if OperationName is not specified and there is only one operation in query it returns the name of that operation
// if the query has multiple operations with same name it will return the first one
func (r Request) Parse() (Operation, string, error) {
	doc, err := parser.ParseQuery(&ast.Source{Input: r.Query})
	if err != nil {
		return "", "", err
	}

	if len(doc.Operations) == 0 {
		return "", "", errors.New("no operations found")

	}

	if len(doc.Operations) == 1 && r.OperationName == "" {
		return doc.Operations[0].Operation, doc.Operations[0].Name, nil
	}

	if len(doc.Operations) > 1 && r.OperationName == "" {
		return "", "", errors.New("query contains multiple operations and no operation name was specified")
	}

	for _, op := range doc.Operations {
		if op.Name == r.OperationName {
			return op.Operation, op.Name, nil
		}
	}

	return "", "", fmt.Errorf("no operation found with name %s", r.OperationName)
}

// Response represents a GraphQL response.
type Response struct {
	Data   map[string]interface{} `json:"data,omitempty"`
	Errors []interface{}          `json:"errors,omitempty"`
}

// ParseGraphQLRequest parses a GraphQL request from an http.Request.
// It supports GET and POST requests.
// If the request is a GET request, it expects the query parameter to be present.
// In case of POST request the Content-Type header must be application/json or application/graphql.
func ParseGraphQLRequest(r *http.Request) (*Request, error) {
	req := new(Request)
	switch r.Method {
	case http.MethodGet:
		vars := r.URL.Query()
		if !vars.Has("query") {
			return nil, errors.New("query parameter is required")
		}

		req.Query = vars.Get("query")

		if vars.Has("variables") {
			json.Unmarshal([]byte(vars.Get("variables")), &req.Variables)
		}

		if vars.Has("operationName") {
			req.OperationName = vars.Get("operationName")
		}
	case http.MethodPost:

		switch r.Header.Get("Content-Type") {

		case "application/graphql":
			// TODO: Size limit
			query, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("error reading request body: %w", err)
			}
			req.Query = string(query)
		case "application/json":
			if err := json.NewDecoder(r.Body).Decode(req); err != nil {
				return nil, fmt.Errorf("error parsing request body: %w", err)
			}
		default:
			return nil, errors.New("unsupported content type")
		}
	default:
		return nil, errors.New("method not allowed")
	}
	return req, nil
}
