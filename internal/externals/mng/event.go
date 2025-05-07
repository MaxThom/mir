package mng

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/pkg/errors"
	"github.com/surrealdb/surrealdb.go"
)

var (
	ErrorListingEvents = errors.New("error listing events from database")
	ErrorNoEventFound  = errors.New("no events found with current targets criteria")
)

const (
	surrealEventTable string = "events"
)

type eventWithId struct {
	Id string `json:"id"`
	mir_models.Event
}

func (s *surrealMirStore) ListEvent(t mir_models.EventTarget) ([]mir_models.Event, error) {
	q, v := createListQueryForEvents(t)
	devs, err := executeQueryForType[[]mir_models.Event](s.db, q, v)
	if err != nil {
		return nil, errors.Wrap(err, ErrorListingDevices.Error())
	}
	return devs, nil
}

func (s *surrealMirStore) CreateEvent(e mir_models.Event) (mir_models.Event, error) {
	// Validate
	if err := e.Validate(); err != nil {
		return mir_models.Event{}, err
	}
	q, v := createIsObjectUniqueQuery(surrealEventTable, e.Meta.Name, e.Meta.Namespace)
	respCheck, err := executeQueryForType[[]mir_models.Event](s.db, q, v)
	if err != nil {
		return mir_models.Event{}, fmt.Errorf("%w for event %s/%s: %w", mir_models.ErrorDbExecutingQuery, e.Meta.Name, e.Meta.Namespace, err)
	}
	if len(respCheck) > 0 {
		return mir_models.Event{}, fmt.Errorf("event %s/%s already exist", e.Meta.Name, e.Meta.Namespace)
	}

	// Create
	respDb, err := s.db.Create(surrealEventTable, e)
	if err != nil {
		return mir_models.Event{}, fmt.Errorf("%w: %w", mir_models.ErrorDbExecutingQuery, err)
	}
	new := []mir_models.Event{}
	err = surrealdb.Unmarshal(respDb, &new)
	if err != nil {
		return mir_models.Event{}, fmt.Errorf("%w for event %s/%s: %w", mir_models.ErrorDbDeserializingResponse, e.Meta.Name, e.Meta.Namespace, err)
	}
	return new[0], nil
}

// This method is too OP
// Maybe it need to be divided into Upsert and Patch
// Upsert is for apply and edit
// Patch is for patch
func (s *surrealMirStore) UpdateEvent(t mir_models.ObjectTarget, upd mir_models.EventUpdate) ([]mir_models.Event, error) {
	if t.HasNoTarget() {
		return nil, mir_models.ErrorNoDeviceTargetProvided
	}

	obj := mir_models.Object{
		Meta: mir_models.Meta{
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

	// Update is full document
	// Change is a merge
	// Modify is a patch
	q := ""
	v := map[string]any{}
	q, v = createUpdateQueryForEvents(t, upd)
	if q == "" {
		return s.ListEvent(mir_models.EventTarget{ObjectTarget: t})
	}
	respDb, err := executeQueryForType[[]mir_models.Event](s.db, q, v)
	if err != nil {
		return nil, errors.Wrap(err, mir_models.ErrorDbExecutingQuery.Error())
	}

	return respDb, nil
}

func (s *surrealMirStore) MergeEvent(t mir_models.ObjectTarget, patch json.RawMessage, op UpdateType) ([]mir_models.Event, error) {
	if t.HasNoTarget() {
		return nil, mir_models.ErrorNoDeviceTargetProvided
	}
	if op == MergePatch {
		// Validate json
		event := mir_models.Event{}
		d := json.NewDecoder(bytes.NewReader(patch))
		d.DisallowUnknownFields()
		if err := d.Decode(&event); err != nil {
			return nil, fmt.Errorf("unknown fields in json patch: %w", err)
		}

		if err := validateObjectMetaForUpdate(s.db, surrealEventTable, t, event.Object); err != nil {
			return nil, err
		}

		var qSb strings.Builder
		if len(patch) > 0 {
			qSb.WriteString("UPDATE " + surrealEventTable + " MERGE ")
			// NONE is a special value for null for SurrealDB
			patch = nullRegEx.ReplaceAll(patch, []byte("${1}NONE"))
			qSb.Write(patch)
			qSb.WriteString(" WHERE ")
			qSb.WriteString(createTargetStatementForObjects(t))
			qSb.WriteString(";")
		}
		sql := qSb.String()

		respDb, err := executeQueryForType[[]mir_models.Event](s.db, sql, map[string]any{})
		if err != nil {
			return nil, errors.Wrap(err, mir_models.ErrorDbExecutingQuery.Error())
		}
		return respDb, nil
	}
	return nil, errors.New("only MergePatch operation is implemented")
}

func (s *surrealMirStore) DeleteEvent(t mir_models.EventTarget) ([]mir_models.Event, error) {
	if t.HasNoTarget() {
		return nil, mir_models.ErrorNoDeviceTargetProvided
	}

	qList, vList := createListQueryForEvents(t)
	respDbList, err := executeQueryForType[[]mir_models.Event](s.db, qList, vList)
	if err != nil {
		return nil, mir_models.ErrorDbExecutingQuery
	}

	q, v := createDeleteQueryForEvents(t)
	_, err = executeQueryForType[[]mir_models.Event](s.db, q, v)
	if err != nil {
		return nil, mir_models.ErrorDbExecutingQuery
	}

	return respDbList, nil
}

func createUpdateQueryForEvents(t mir_models.ObjectTarget, upd mir_models.EventUpdate) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	if upd.Meta != nil {
		var sb strings.Builder
		if upd.Meta.Name != nil && *upd.Meta.Name != "" {
			sb.WriteString("name: $NAME,")
			vars["NAME"] = *upd.Meta.Name
		}
		if upd.Meta.Namespace != nil && *upd.Meta.Namespace != "" {
			sb.WriteString("namespace: $NS,")
			vars["NS"] = *upd.Meta.Namespace
		}
		if upd.Meta.Labels != nil && len(upd.Meta.Labels) > 0 {
			sb.WriteString("labels: {")
			for key, val := range upd.Meta.Labels {
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
		if upd.Meta.Annotations != nil && len(upd.Meta.Annotations) > 0 {
			sb.WriteString("annotations: {")
			for key, val := range upd.Meta.Annotations {
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
	if upd.Spec != nil {
		var sb strings.Builder
		if upd.Spec.Type != nil && *upd.Spec.Type != "" {
			// TODO validate if type is normal|warning
			sb.WriteString("type: $TYPE,")
			vars["TYPE"] = *upd.Spec.Type
		}
		if upd.Spec.Reason != nil {
			sb.WriteString("reason: $RES,")
			vars["RES"] = *upd.Spec.Reason
		}
		if upd.Spec.Message != nil {
			sb.WriteString("message: $MSG,")
			vars["MSG"] = *upd.Spec.Message
		}
		if upd.Spec.Payload != nil {
			x, _ := json.Marshal(upd.Spec.Payload)
			if len(x) > 0 {
				// Curlies are in the desired json already
				sb.WriteString("payload: ")
				sb.Write(nullRegEx.ReplaceAll(x, []byte("${1}NONE")))
			}
		}
		if upd.Spec.RelatedObject != nil {
			var sbObj strings.Builder
			if upd.Spec.RelatedObject.ApiName != nil {
				sb.WriteString("apiName: $API,")
				vars["API"] = *upd.Spec.RelatedObject.ApiName
			}
			if upd.Spec.RelatedObject.ApiVersion != nil {
				sb.WriteString("apiVersion: $VER,")
				vars["API"] = *upd.Spec.RelatedObject.ApiVersion
			}
			if upd.Spec.RelatedObject.Meta != nil {
				var sbMeta strings.Builder
				if upd.Spec.RelatedObject.Meta.Name != nil {
					sb.WriteString("name: $NM,")
					vars["NM"] = *upd.Spec.RelatedObject.Meta.Name
				}
				if upd.Spec.RelatedObject.Meta.Namespace != nil {
					sb.WriteString("namespace: $NSO,")
					vars["NSO"] = *upd.Spec.RelatedObject.Meta.Namespace
				}
				if upd.Spec.RelatedObject.Meta.Labels != nil && len(upd.Spec.RelatedObject.Meta.Labels) > 0 {
					sb.WriteString("labels: {")
					for key, val := range upd.Spec.RelatedObject.Meta.Labels {
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
				if upd.Spec.RelatedObject.Meta.Annotations != nil && len(upd.Spec.RelatedObject.Meta.Annotations) > 0 {
					sb.WriteString("annotations: {")
					for key, val := range upd.Spec.RelatedObject.Meta.Annotations {
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
				if sbMeta.Len() > 0 {
					sbObj.WriteString("meta: {")
					sbObj.WriteString(sbMeta.String())
					sbObj.WriteString("},")
				}
			}
			if sbObj.Len() > 0 {
				sb.WriteString("relatedObject: {")
				sb.WriteString(sbObj.String())
				sb.WriteString("},")
			}

		}
		if sb.Len() > 0 {
			q.WriteString("spec: {")
			q.WriteString(sb.String())
			q.WriteString("},")
		}
	}
	if upd.Status != nil {
		var sb strings.Builder
		if upd.Status.Count != nil {
			sb.WriteString("count: $CO,")
			vars["CO"] = *upd.Status.Count
		}
		if upd.Status.FirstAt != nil {
			sb.WriteString("firstAt: $FA,")
			vars["FA"] = *upd.Status.FirstAt
		}
		if upd.Status.LastAt != nil {
			sb.WriteString("lastAt: $LA,")
			vars["LA"] = *upd.Status.LastAt
		}
		if sb.Len() > 0 {
			q.WriteString("status: {")
			q.WriteString(sb.String())
			q.WriteString("},")
		}
	}

	var qSb strings.Builder
	if q.Len() > 0 {
		qSb.WriteString("UPDATE events MERGE {")
		qSb.WriteString(q.String())
		qSb.WriteString("} WHERE ")
		qSb.WriteString(createTargetStatementForObjects(t))
		qSb.WriteString(";")
	}
	sql = qSb.String()

	return
}

func createListQueryForEvents(t mir_models.EventTarget) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("SELECT * FROM events")
	whereObj := createTargetStatementForEvents(t)

	if len(whereObj) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(whereObj)
	}
	q.WriteString(" ORDER BY status.firstAt DESC")
	if t.Limit > 0 {
		q.WriteString(fmt.Sprintf(" LIMIT %d", t.Limit))
	}
	q.WriteString(";")
	sql = q.String()
	return
}

func createTargetStatementForEvents(t mir_models.EventTarget) string {
	whereSt := []string{}
	whereObj := createTargetStatementForObjects(t.ObjectTarget)
	if len(whereObj) > 0 {
		whereSt = append(whereSt, whereObj)
	}
	if !t.DateFilter.From.IsZero() {
		var dt strings.Builder
		dt.WriteString("status.firstAt >= ")
		dt.WriteString("d\"")
		dt.WriteString(t.DateFilter.From.Format(time.RFC3339Nano))
		dt.WriteString("\"")
		whereSt = append(whereSt, dt.String())
	}
	if !t.DateFilter.To.IsZero() {
		var dt strings.Builder
		dt.WriteString(" status.firstAt <= ")
		dt.WriteString("d\"")
		dt.WriteString(t.DateFilter.To.Format(time.RFC3339Nano))
		dt.WriteString("\"")
		whereSt = append(whereSt, dt.String())
	}
	return strings.Join(whereSt, " AND ")
}

func createDeleteQueryForEvents(t mir_models.EventTarget) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM events WHERE ")
	q.WriteString(createTargetStatementForEvents(t))
	q.WriteString(";")
	sql = q.String()
	return
}
