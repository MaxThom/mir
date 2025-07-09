package mng

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/surrealdb/surrealdb.go"
)

func natskvCreateIsObjectUniqueQuery(scope, version, kind, ns, name string) (sql string, vars map[string]any) {
	return "", nil
}

func natskvExecutQueryForType[T any](db *surrealdb.DB, query string, vars map[string]any) (T, error) {
	var empty T
	result, err := db.Query(query, vars)
	if err != nil {
		return empty, err
	}

	res, err := surrealdb.SmartUnmarshal[T](result, err)
	if err != nil {
		return empty, err
	}

	return res, nil
}

func natskvCreateListQueryForObjects(table string, t mir_v1.ObjectTarget) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("SELECT * FROM " + table)
	where := createSurrealTargetStatementForObjects(t)

	if len(where) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(where)
	}
	q.WriteString(";")
	sql = q.String()
	return
}

func natskvCreateTargetStatementForObjects(t mir_v1.ObjectTarget) string {
	var q strings.Builder

	cond := []string{}
	if len(t.Names) > 0 {
		var i []string
		for _, n := range t.Names {
			i = append(i, fmt.Sprintf("meta.name CONTAINS \"%s\"", n))
		}
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Namespaces) > 0 {
		var i []string
		for _, ns := range t.Namespaces {
			i = append(i, fmt.Sprintf("meta.namespace = \"%s\"", ns))
		}
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Labels) > 0 {
		var i []string
		for k, v := range t.Labels {
			i = append(i, fmt.Sprintf("meta.labels.%s CONTAINS \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(i, " AND ")+")")
	}
	q.WriteString(strings.Join(cond, " AND "))
	ti := q.String()
	return ti
}

func natskvCreateDeleteQueryForObjects(table string, t mir_v1.ObjectTarget) (sql string, vars map[string]any) {
	return "", nil
}

func (s *natskvMirStore) validateObjectMetaForUpdate(table string, t mir_v1.ObjectTarget, obj mir_v1.Object) error {
	return nil
}

// generateLabelsHash creates a highly unique hash
func generateLabelsHash(labels map[string]string) string {
	if labels == nil || len(labels) == 0 {
		return ""
	}

	// Sort keys for consistent hashing
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sorted key=value string
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(labels[k])
	}

	// Generate SHA256 hash and use first 12 bytes as base64 (16 chars)
	hash := sha256.Sum256([]byte(b.String()))
	return base64.URLEncoding.EncodeToString(hash[:12])
}
