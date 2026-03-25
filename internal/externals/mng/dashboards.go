package mng

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/pkgs/mir_v1"
)

var (
	ErrorListingDashboards = errors.New("error listing dashboards from database")
	ErrorDashboardNotFound = errors.New("dashboard not found")
)

const (
	surrealDashboardTable string = "dashboards"
)

func (s *surrealMirStore) ListDashboards(t mir_v1.ObjectTarget) ([]mir_v1.Dashboard, error) {
	q, v := createListQueryForObjects(surrealDashboardTable, t)
	dash, err := surreal.Query[[]mir_v1.Dashboard](s.db, q, v)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrorListingDashboards, err)
	}
	return dash, nil
}

func (s *surrealMirStore) CreateDashboard(d mir_v1.Dashboard) (mir_v1.Dashboard, error) {
	// Uniqueness
	if err := d.Validate(); err != nil {
		return mir_v1.Dashboard{}, err
	}
	q, v := createIsObjectUniqueQuery(surrealDashboardTable, d.Meta.Name, d.Meta.Namespace)
	respCheck, err := surreal.Query[[]mir_v1.Event](s.db, q, v)
	if err != nil {
		return mir_v1.Dashboard{}, fmt.Errorf("%w for dashboard %s/%s: %w", mir_v1.ErrorDbExecutingQuery, d.Meta.Name, d.Meta.Namespace, err)
	}
	if len(respCheck) > 0 {
		return mir_v1.Dashboard{}, fmt.Errorf("dashboard %s/%s already exist", d.Meta.Name, d.Meta.Namespace)
	}

	// Validate
	now := time.Now().UTC()
	if d.Status.CreatedAt.IsZero() {
		d.Status.CreatedAt = now
	}
	if d.Status.UpdatedAt.IsZero() {
		d.Status.UpdatedAt = now
	}
	if d.Spec.Widgets == nil {
		d.Spec.Widgets = []mir_v1.DashboardWidget{}
	}

	rec, err := surreal.Create[mir_v1.Dashboard](s.db, surrealDashboardTable, d)
	if err != nil {
		return mir_v1.Dashboard{}, fmt.Errorf("error creating dashboard: %w", err)
	}
	return *rec, nil
}

func (s *surrealMirStore) UpdateDashboard(t mir_v1.ObjectTarget, upd mir_v1.DashboardUpdate) ([]mir_v1.Dashboard, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}

	obj := mir_v1.Object{
		Meta: mir_v1.Meta{
			Name:      "",
			Namespace: "",
		},
	}
	if upd.Meta != nil && upd.Meta.Name != nil {
		obj.Meta.Name = *upd.Meta.Name
	}
	if upd.Meta != nil && upd.Meta.Namespace != nil {
		obj.Meta.Namespace = *upd.Meta.Namespace
	}
	if err := validateObjectMetaForUpdate(s.db, surrealEventTable, t, obj); err != nil {
		return nil, err
	}

	q, v := createUpdateQueryForDashboards(t, upd)
	if q == "" {
		return s.ListDashboards(t)
	}
	respDb, err := surreal.Query[[]mir_v1.Dashboard](s.db, q, v)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", mir_v1.ErrorDbExecutingQuery.Error(), err)
	}
	return respDb, nil
}

func (s *surrealMirStore) MergeDashboard(t mir_v1.ObjectTarget, patch json.RawMessage, op UpdateType) ([]mir_v1.Dashboard, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}
	if op == MergePatch {
		// Validate json
		dash := mir_v1.Dashboard{}
		d := json.NewDecoder(bytes.NewReader(patch))
		d.DisallowUnknownFields()
		if err := d.Decode(&dash); err != nil {
			return nil, fmt.Errorf("unknown fields in json patch: %w", err)
		}

		if err := validateObjectMetaForUpdate(s.db, surrealDashboardTable, t, dash.Object); err != nil {
			return nil, err
		}

		var qSb strings.Builder
		if len(patch) > 0 {
			qSb.WriteString("UPDATE " + surrealDashboardTable + " MERGE ")
			// NONE is a special value for null for SurrealDB
			patch = nullRegEx.ReplaceAll(patch, []byte("${1}NONE"))
			qSb.Write(patch)
			qSb.WriteString(" WHERE ")
			qSb.WriteString(createTargetStatementForObjects(t))
			qSb.WriteString(";")
		}
		sql := qSb.String()

		respDb, err := surreal.Query[[]mir_v1.Dashboard](s.db, sql, map[string]any{})
		if err != nil {
			return nil, fmt.Errorf("%v: %v", mir_v1.ErrorDbExecutingQuery.Error(), err)
		}
		return respDb, nil
	}
	return nil, errors.New("only MergePatch operation is implemented")
}

func (s *surrealMirStore) DeleteDashboard(t mir_v1.ObjectTarget) ([]mir_v1.Dashboard, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}

	dash, err := s.ListDashboards(t)
	if err != nil {
		return []mir_v1.Dashboard{}, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
	}

	q, v := createDeleteQueryForObjects(surrealDashboardTable, t)
	_, err = surreal.Query[any](s.db, q, v)
	if err != nil {
		return []mir_v1.Dashboard{}, fmt.Errorf("error deleting dashboard: %w", err)
	}
	return dash, nil
}

func createUpdateQueryForDashboards(t mir_v1.ObjectTarget, upd mir_v1.DashboardUpdate) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	sMeta, varsMeta := createUpdateQueryForMeta(t, upd.Meta)
	if sMeta != "" {
		q.WriteString(sMeta)
		maps.Copy(vars, varsMeta)
	}

	var sbSpec strings.Builder
	if upd.Spec != nil {
		if upd.Spec.Description != nil && *upd.Spec.Description != "" {
			sbSpec.WriteString("description: $DESC,")
			vars["DESC"] = *upd.Spec.Description
		}
		if upd.Spec.RefreshInterval != nil {
			sbSpec.WriteString("refreshInterval: $RI,")
			vars["RI"] = *upd.Spec.RefreshInterval
		}
		if upd.Spec.TimeMinutes != nil {
			sbSpec.WriteString("timeMinutes: $TM,")
			vars["TM"] = *upd.Spec.TimeMinutes
		}
		if upd.Spec.Widgets != nil {
			sbSpec.WriteString("widgets: $WID,")
			vars["WID"] = upd.Spec.Widgets
		}
	}
	if sbSpec.Len() > 0 {
		q.WriteString("spec: {")
		q.WriteString(sbSpec.String())
		q.WriteString("},")
	}

	var sbStatus strings.Builder
	if upd.Status != nil {
		if upd.Status.CreatedAt != nil && !upd.Status.CreatedAt.IsZero() {
			sbStatus.WriteString("createdAt: $CREATEDAT,")
			vars["CREATEDAT"] = *upd.Status.CreatedAt
		}
		if upd.Status.UpdatedAt != nil && !upd.Status.UpdatedAt.IsZero() {
			sbStatus.WriteString("updatedAt: $UPDATEDAT,")
			vars["UPDATEDAT"] = *upd.Status.UpdatedAt
		}
	}
	if sbStatus.Len() > 0 {
		q.WriteString("status: {")
		q.WriteString(sbStatus.String())
		q.WriteString("},")
	}

	var qSb strings.Builder
	if q.Len() > 0 {
		qSb.WriteString("UPDATE dashboards MERGE {")
		qSb.WriteString(q.String())
		qSb.WriteString("} WHERE ")
		qSb.WriteString(createTargetStatementForObjects(t))
		qSb.WriteString(";")
	}
	sql = qSb.String()
	return
}
