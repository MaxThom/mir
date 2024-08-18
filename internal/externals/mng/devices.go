package mng

import (
	"fmt"
	"strings"

	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/pkg/errors"
	"github.com/surrealdb/surrealdb.go"
)

type DeviceStore interface {
	ListDevice(req *core_apiv1.ListDeviceRequest) ([]mir_models.DeviceWithId, error)
	CreateDevice(req *core_apiv1.CreateDeviceRequest) ([]mir_models.DeviceWithId, error)
	UpdateDevice(req *core_apiv1.UpdateDeviceRequest) ([]mir_models.DeviceWithId, error)
	DeleteDevice(req *core_apiv1.DeleteDeviceRequest) ([]mir_models.DeviceWithId, error)
}

type surrealDeviceStore struct {
	db *surrealdb.DB
}

func NewSurrealDeviceStore(db *surrealdb.DB) *surrealDeviceStore {
	return &surrealDeviceStore{
		db: db,
	}
}

func (s *surrealDeviceStore) ListDevice(req *core_apiv1.ListDeviceRequest) ([]mir_models.DeviceWithId, error) {
	q, v := createListQueryForDevice(req)
	return executeQueryForType[[]mir_models.DeviceWithId](s.db, q, v)
}

func (s *surrealDeviceStore) CreateDevice(req *core_apiv1.CreateDeviceRequest) ([]mir_models.DeviceWithId, error) {
	// Validate
	if req.DeviceId == "" {
		return nil, mir_models.ErrorInvalidDeviceID
	}
	q, v := createListQueryForDevice(&core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{req.DeviceId},
		},
	})
	respCheck, err := executeQueryForType[[]mir_models.DeviceWithId](s.db, q, v)
	if err != nil {
		// TODO check on how to use error.wrap
		return nil, mir_models.ErrorDbExecutingQuery
	}
	if len(respCheck) > 0 {
		return nil, mir_models.ErrorDeviceIdAlreadyExist
	}

	// Create
	respDb, err := s.db.Create("devices", mir_models.NewDeviceFromCreateDeviceReq(req))
	if err != nil {
		return nil, mir_models.ErrorDbExecutingQuery
	}
	newDev := []mir_models.DeviceWithId{}
	err = surrealdb.Unmarshal(respDb, &newDev)
	if err != nil {
		return nil, mir_models.ErrorDbDeserializingResponse
	}
	return newDev, nil
}

func (s *surrealDeviceStore) UpdateDevice(req *core_apiv1.UpdateDeviceRequest) ([]mir_models.DeviceWithId, error) {
	if req.Targets == nil ||
		len(req.Targets.Ids) == 0 &&
			len(req.Targets.Names) == 0 &&
			len(req.Targets.Namespaces) == 0 &&
			len(req.Targets.Labels) == 0 &&
			len(req.Targets.Annotations) == 0 {
		return nil, mir_models.ErrorNoDeviceTargetProvided
	}

	// Update is full document
	// Change is a merge
	// Modify is a patch

	q := ""
	v := map[string]any{}
	q, v = createUpdateQueryForDevice(req.Targets, req)
	respDb, err := executeQueryForType[[]mir_models.DeviceWithId](s.db, q, v)
	if err != nil {
		return nil, errors.Wrap(err, mir_models.ErrorDbExecutingQuery.Error())
	}

	return respDb, nil
}

func (s *surrealDeviceStore) DeleteDevice(req *core_apiv1.DeleteDeviceRequest) ([]mir_models.DeviceWithId, error) {
	if req.Targets == nil ||
		len(req.Targets.Ids) == 0 &&
			len(req.Targets.Names) == 0 &&
			len(req.Targets.Namespaces) == 0 &&
			len(req.Targets.Labels) == 0 &&
			len(req.Targets.Annotations) == 0 {
		return nil, mir_models.ErrorNoDeviceTargetProvided
	}

	qList, vList := createListQueryForDevice(&core_apiv1.ListDeviceRequest{
		Targets: req.Targets,
	})
	respDbList, err := executeQueryForType[[]mir_models.DeviceWithId](s.db, qList, vList)
	if err != nil {
		return nil, mir_models.ErrorDbExecutingQuery
	}

	q, v := createDeleteQueryForDevice(req)
	_, err = executeQueryForType[[]mir_models.DeviceWithId](s.db, q, v)
	if err != nil {
		return nil, mir_models.ErrorDbExecutingQuery
	}

	return respDbList, nil
}

func createListQueryForDevice(req *core_apiv1.ListDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("SELECT * FROM devices")
	where := createWhereStatementWithTargets(req.Targets)
	if len(where) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(where)
	}

	q.WriteString(";")
	sql = q.String()
	return
}

func createUpdateQueryForDevice(t *core_apiv1.Targets, upd *core_apiv1.UpdateDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	q.WriteString("UPDATE devices MERGE {")
	if upd.Meta != nil {
		q.WriteString("meta: {")
		if upd.Meta.Name != nil {
			q.WriteString("name: $NAME,")
			vars["NAME"] = *upd.Meta.Name
		}
		if upd.Meta.Namespace != nil {
			q.WriteString("namespace: $NS,")
			vars["NS"] = *upd.Meta.Namespace
		}
		if upd.Meta.Labels != nil && len(upd.Meta.Labels) > 0 {
			q.WriteString("labels: {")
			for key, val := range upd.Meta.Labels {
				q.WriteString("\"")
				q.WriteString(key)
				q.WriteString("\"")
				q.WriteString(": ")
				if val == nil || val.Value == nil {
					q.WriteString("NONE")
				} else {
					q.WriteString(fmt.Sprintf("\"%s\"", val.GetValue()))
				}
				q.WriteString(",")
			}
			q.WriteString("},")
		}
		if upd.Meta.Annotations != nil && len(upd.Meta.Annotations) > 0 {
			q.WriteString("annotations: {")
			for key, val := range upd.Meta.Annotations {
				q.WriteString("\"")
				q.WriteString(key)
				q.WriteString("\"")
				q.WriteString(": ")
				if val == nil || val.Value == nil {
					q.WriteString("NONE")
				} else {
					q.WriteString(fmt.Sprintf("\"%s\"", val.GetValue()))
				}
				q.WriteString(",")
			}
			q.WriteString("},")
		}
		q.WriteString("},")
	}
	if upd.Spec != nil {
		q.WriteString("spec: {")
		if upd.Spec.Disabled != nil {
			q.WriteString("disabled: $DIS,")
			vars["DIS"] = *upd.Spec.Disabled
		}
		q.WriteString("},")
	}
	if upd.Status != nil {
		q.WriteString("status: {")
		if upd.Status.LastHearthbeat != nil && !mir_models.AsGoTime(upd.Status.LastHearthbeat).IsZero() {
			q.WriteString("lastHearthbeat: $BEAT,")
			vars["BEAT"] = mir_models.AsGoTime(upd.Status.LastHearthbeat)
		}
		if upd.Status.Online != nil {
			q.WriteString("online: $ON,")
			vars["ON"] = upd.Status.Online
		}
		if upd.Status.Schema != nil {
			q.WriteString("schema: {")
			if upd.Status.Schema.CompressedSchema != nil {
				q.WriteString("compressedSchema: $COMPSCHEMA,")
				vars["COMPSCHEMA"] = upd.Status.Schema.CompressedSchema
			}
			if upd.Status.Schema.PackageNames != nil {
				q.WriteString("packageNames: $PACKNAMES,")
				vars["PACKNAMES"] = upd.Status.Schema.PackageNames
			}
			if upd.Status.Schema.LastSchemaFetch != nil && !mir_models.AsGoTime(upd.Status.Schema.LastSchemaFetch).IsZero() {
				q.WriteString("lastSchemaFetch: $LASTSCHFETCH,")
				vars["LASTSCHFETCH"] = mir_models.AsGoTime(upd.Status.Schema.LastSchemaFetch)
			}
			q.WriteString("},")
		}
		q.WriteString("},")
	}

	q.WriteString("} WHERE ")
	q.WriteString(createWhereStatementWithTargets(t))
	q.WriteString(";")
	sql = q.String()

	return
}

func createDeleteQueryForDevice(req *core_apiv1.DeleteDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createWhereStatementWithTargets(req.Targets))
	q.WriteString(";")
	sql = q.String()
	return
}

func createWhereStatementWithTargets(t *core_apiv1.Targets) string {
	var q strings.Builder
	if t == nil {
		return ""
	}

	cond := []string{}
	if len(t.Ids) > 0 {
		var i []string
		for _, id := range t.Ids {
			i = append(i, fmt.Sprintf("spec.deviceId = \"%s\"", id))
		}
		cond = append(cond, strings.Join(i, " OR "))
	}
	if len(t.Names) > 0 {
		var i []string
		for _, ns := range t.Names {
			i = append(i, fmt.Sprintf("meta.name = \"%s\"", ns))
		}
		cond = append(cond, strings.Join(i, " OR "))
	}
	if len(t.Namespaces) > 0 {
		var i []string
		for _, ns := range t.Namespaces {
			i = append(i, fmt.Sprintf("meta.namespace = \"%s\"", ns))
		}
		cond = append(cond, strings.Join(i, " OR "))
	}
	if len(t.Labels) > 0 {
		var i []string
		for k, v := range t.Labels {
			i = append(i, fmt.Sprintf("meta.labels.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(i, " AND ")+")")
	}
	if len(t.Annotations) > 0 {
		var i []string
		for k, v := range t.Annotations {
			i = append(i, fmt.Sprintf("meta.annotations.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(i, " AND ")+")")
	}
	// TODO switch this to AND, must add ( ) above
	q.WriteString(strings.Join(cond, " OR "))
	ti := q.String()
	return ti
}

func executeQueryForType[T any](db *surrealdb.DB, query string, vars map[string]any) (T, error) {
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
