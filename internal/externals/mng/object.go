package mng

import (
	"fmt"
	"slices"
	"strings"

	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/pkgs/mir_v1"
)

func createIsObjectUniqueQuery(table, name, ns string) (sql string, vars map[string]any) {
	var q strings.Builder
	q.WriteString("SELECT * FROM " + table + " WHERE ")
	if name != "" && ns != "" {
		q.WriteString(fmt.Sprintf("(meta.name = \"%s\"", name))
		q.WriteString(" AND ")
		q.WriteString(fmt.Sprintf("meta.namespace = \"%s\")", ns))
	}
	q.WriteString(";")
	sql = q.String()
	return
}

func createListQueryForObjects(table string, t mir_v1.ObjectTarget) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("SELECT * FROM " + table)
	where := createTargetStatementForObjects(t)

	if len(where) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(where)
	}
	q.WriteString(";")
	sql = q.String()
	return
}

func createTargetStatementForObjects(t mir_v1.ObjectTarget) string {
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

func createDeleteQueryForObjects(table string, t mir_v1.ObjectTarget) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM " + table)
	where := createTargetStatementForObjects(t)
	if len(where) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(where)
	}

	q.WriteString(";")
	sql = q.String()
	return
}

func validateObjectMetaForUpdate(db *surreal.AutoReconnDB, table string, t mir_v1.ObjectTarget, obj mir_v1.Object) error {
	// if err := obj.Validate(); err != nil {
	// 	return err
	// }
	q, v := createListQueryForObjects(table, t)
	changingObjs, err := surreal.Query[[]mir_v1.Object](db, q, v)
	if err != nil {
		return fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
	}

	if obj.Meta.Name != "" && obj.Meta.Namespace != "" {
		if len(changingObjs) > 1 {
			return fmt.Errorf("cannot update multiple object as name/namespace '%s/%s' must be unique", obj.Meta.Name, obj.Meta.Namespace)
		} else if len(changingObjs) == 1 {
			// Check if name/ns is unique
			q, v := createIsObjectUniqueQuery(table, obj.Meta.Name, obj.Meta.Namespace)
			respCheck, err := surreal.Query[[]mir_v1.Object](db, q, v)
			if err != nil {
				return fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
			}
			if len(respCheck) > 0 && (respCheck[0].Meta.Name != changingObjs[0].Meta.Name || respCheck[0].Meta.Namespace != changingObjs[0].Meta.Namespace) {
				return fmt.Errorf("cannot update object has '%s/%s' is already in use", obj.Meta.Name, obj.Meta.Namespace)
			}
		}
	} else if obj.Meta.Namespace != "" {
		q, v := createListQueryForObjects(table, mir_v1.ObjectTarget{
			Namespaces: []string{obj.Meta.Namespace},
		})
		currentObjs, err := surreal.Query[[]mir_v1.Object](db, q, v)
		if err != nil {
			return fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
		}

		names := []string{}
		for _, d := range changingObjs {
			if slices.Contains(names, d.Meta.Name) {
				return fmt.Errorf("cannot update object as multiple device will have the same name '%s' in namespace '%s'", d.Meta.Name, d.Meta.Namespace)
			}
			names = append(names, d.Meta.Name)
		}
		names = []string{}
		for _, d := range currentObjs {
			names = append(names, d.Meta.Name)
		}
		for _, d := range changingObjs {
			if slices.Contains(names, d.Meta.Name) {
				return fmt.Errorf("cannot update object as name '%s' is already in use in namespace '%s'", d.Meta.Name, d.Meta.Namespace)
			}
		}
	} else if obj.Meta.Name != "" {
		q, v := createListQueryForObjects(table, mir_v1.ObjectTarget{
			Names: []string{obj.Meta.Name},
		})
		currentObjs, err := surreal.Query[[]mir_v1.Object](db, q, v)
		if err != nil {
			return fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
		}

		namespaces := []string{}
		for _, d := range changingObjs {
			if slices.Contains(namespaces, d.Meta.Namespace) {
				return fmt.Errorf("cannot update event as multiple object will have the same name '%s' in namespace '%s'", obj.Meta.Name, d.Meta.Namespace)
			}
			namespaces = append(namespaces, d.Meta.Namespace)
		}
		for _, newD := range changingObjs {
			for _, oldD := range currentObjs {
				if newD.Meta.Namespace == oldD.Meta.Namespace {
					return fmt.Errorf("cannot update object as name '%s' is already in use in namespace '%s'", oldD.Meta.Name, oldD.Meta.Namespace)
				}
			}
		}
	}
	return nil
}

func createUpdateQueryForMeta(t mir_v1.ObjectTarget, upd *mir_v1.MetaUpdate) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	if upd != nil {
		var sb strings.Builder
		if upd.Name != nil && *upd.Name != "" {
			sb.WriteString("name: $NAME,")
			vars["NAME"] = *upd.Name
		}
		if upd.Namespace != nil && *upd.Namespace != "" {
			sb.WriteString("namespace: $NS,")
			vars["NS"] = *upd.Namespace
		}
		if len(upd.Labels) > 0 {
			sb.WriteString("labels: {")
			for key, val := range upd.Labels {
				sb.WriteString("\"")
				sb.WriteString(key)
				sb.WriteString("\"")
				sb.WriteString(": ")
				if val == nil {
					sb.WriteString("NONE")
				} else {
					sb.WriteString(fmt.Sprintf("\"%s\"", *val))
				}
				sb.WriteString(",")
			}
			sb.WriteString("},")
		}
		if len(upd.Annotations) > 0 {
			sb.WriteString("annotations: {")
			for key, val := range upd.Annotations {
				sb.WriteString("\"")
				sb.WriteString(key)
				sb.WriteString("\"")
				sb.WriteString(": ")
				if val == nil {
					sb.WriteString("NONE")
				} else {
					sb.WriteString(fmt.Sprintf("\"%s\"", *val))
				}
				sb.WriteString(",")
			}
			sb.WriteString("},")
		}
		if sb.Len() > 0 {
			q.WriteString("meta: {")
			q.WriteString(sb.String())
			q.WriteString("},")
		}
	}
	sql = q.String()
	return
}
