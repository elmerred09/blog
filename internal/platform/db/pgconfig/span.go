package pgconfig

import "strings"

// sqlcSpanName is the query name from sqlc's leading comment
// ("-- name: GetPostBySlug :one"). The span stays low-cardinality, and the
// name matches the method a reader finds in db/queries. A statement without
// that comment, such as a fixture's plain SQL, is just "query".
func sqlcSpanName(stmt string) string {
	const prefix = "-- name: "
	rest, ok := strings.CutPrefix(stmt, prefix)
	if !ok {
		return "query"
	}
	name, _, _ := strings.Cut(rest, " ")
	if name == "" {
		return "query"
	}
	return name
}
