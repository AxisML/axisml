package openapigen

// OpenAPI 3.0.3 document subset. We emit only the fields the per-service
// generators use; adding new fields here is intentional (and will need
// matching coverage in the cmd/openapi-gen integrity tests).

// Document is the OpenAPI 3.0.3 root.
type Document struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Servers    []ServerEntry       `json:"servers,omitempty"`
	Tags       []TagEntry          `json:"tags,omitempty"`
	Paths      map[string]PathItem `json:"paths"`
	Components ComponentsBlock     `json:"components"`
}

// Info is the spec's identification block.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// ServerEntry is one entry of `servers:`.
type ServerEntry struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// TagEntry groups operations under a name.
type TagEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PathItem holds the methods registered at one URL.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Put    *Operation `json:"put,omitempty"`
}

// Operation is one HTTP method on a path.
type Operation struct {
	Tags        []string            `json:"tags,omitempty"`
	Summary     string              `json:"summary,omitempty"`
	OperationID string              `json:"operationId,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

// Parameter describes a path/query/header parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody is an operation's body description.
type RequestBody struct {
	Required bool                 `json:"required,omitempty"`
	Content  map[string]MediaType `json:"content"`
}

// Response is one entry under operation.responses.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// MediaType wraps a content-type's schema.
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// ComponentsBlock holds reusable schemas referenced via $ref.
type ComponentsBlock struct {
	Schemas map[string]*Schema `json:"schemas"`
}

// Schema is the subset of OpenAPI 3.0 schema we emit. Only the fields the
// generator actually populates are present — adding a new constraint means
// adding the field here AND walking it in walkSchema for $ref integrity.
type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"`
	Enum                 []string           `json:"enum,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	MinItems             *int               `json:"minItems,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
}

// Ref returns a #/components/schemas/<name> reference Schema.
func Ref(name string) *Schema { return &Schema{Ref: "#/components/schemas/" + name} }

// PathParam returns a required string-typed path parameter.
func PathParam(name, desc string) Parameter {
	return Parameter{Name: name, In: "path", Required: true, Description: desc, Schema: &Schema{Type: "string"}}
}

// QueryParam returns an optional query parameter with the given schema.
func QueryParam(name, desc string, sch *Schema) Parameter {
	return Parameter{Name: name, In: "query", Description: desc, Schema: sch}
}

// JSONBody returns a required application/json request body referencing the
// given component schema name.
func JSONBody(refName string) *RequestBody {
	return &RequestBody{
		Required: true,
		Content:  map[string]MediaType{"application/json": {Schema: Ref(refName)}},
	}
}

// JSONResp returns a Response with an application/json body referencing the
// given component schema name.
func JSONResp(desc, refName string) Response {
	return Response{
		Description: desc,
		Content:     map[string]MediaType{"application/json": {Schema: Ref(refName)}},
	}
}

// StringResp returns a Response with a text/plain body.
func StringResp(desc string) Response {
	return Response{
		Description: desc,
		Content:     map[string]MediaType{"text/plain": {Schema: &Schema{Type: "string"}}},
	}
}

// NoContentResp is the canonical 204 response.
var NoContentResp = Response{Description: "Deleted (idempotent)."}

// IntFormat32Param is a shared int32 schema used for limit/offset-style query
// parameters.
func IntFormat32Param() *Schema { return &Schema{Type: "integer", Format: "int32"} }
