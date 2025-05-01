package mng

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/pkg/errors"
	"github.com/surrealdb/surrealdb.go"
)

var (
	ErrorListingDevices        = errors.New("error listing devices from database")
	ErrorNoDeviceFound         = errors.New("no device found with current targets criteria")
	ErrorDeviceShouldBeCreated = errors.New("device should be created")
)

type deviceWithId struct {
	Id string `json:"id"`
	mir_models.Device
}

func (s *surrealMirStore) ListDevice(req *core_apiv1.ListDeviceRequest) ([]mir_models.Device, error) {
	q, v := createListQueryForDevice(req)
	devs, err := executeQueryForType[[]mir_models.Device](s.db, q, v)
	if err != nil {
		return nil, errors.Wrap(err, ErrorListingDevices.Error())
	}
	return devs, nil
}

func (s *surrealMirStore) CreateDevice(cdr *core_apiv1.CreateDeviceRequest) (mir_models.Device, error) {
	// Validate
	if cdr.Meta != nil && cdr.Meta.Name == "" && cdr.Spec.DeviceId == "" {
		return mir_models.Device{}, fmt.Errorf("device name and id are missing")
	}
	if cdr.Meta == nil {
		cdr.Meta = &core_apiv1.Meta{}
	}
	if cdr.Meta.Name == "" {
		cdr.Meta.Name = cdr.Spec.DeviceId
	}
	if cdr.Meta.Namespace == "" {
		cdr.Meta.Namespace = "default"
	}
	q, v := createIsDeviceUniqueQuery(cdr.Meta.Name, cdr.Meta.Namespace, cdr.Spec.DeviceId)
	respCheck, err := executeQueryForType[[]mir_models.Device](s.db, q, v)
	if err != nil {
		return mir_models.Device{}, fmt.Errorf("%w for device %s/%s: %w", mir_models.ErrorDbExecutingQuery, cdr.Meta.Name, cdr.Meta.Namespace, err)
	}
	if len(respCheck) > 0 {
		return mir_models.Device{}, fmt.Errorf("device %s/%s with deviceId %s already exist", cdr.Meta.Name, cdr.Meta.Namespace, cdr.Spec.DeviceId)
	}

	// Create
	respDb, err := s.db.Create("devices", mir_models.NewDeviceFromCreateDeviceReq(cdr))
	if err != nil {
		return mir_models.Device{}, fmt.Errorf("%w: %w", mir_models.ErrorDbExecutingQuery, err)
	}
	newDev := []mir_models.Device{}
	err = surrealdb.Unmarshal(respDb, &newDev)
	if err != nil {
		return mir_models.Device{}, fmt.Errorf("%w for device %s/%s: %w", mir_models.ErrorDbDeserializingResponse, cdr.Meta.Name, cdr.Meta.Namespace, err)
	}
	return newDev[0], nil
}

// This method is too OP
// Maybe it need to be divided into Upsert and Patch
// Upsert is for apply and edit
// Patch is for patch
func (s *surrealMirStore) UpdateDevice(req *core_apiv1.UpdateDeviceRequest) ([]mir_models.Device, error) {
	if req.Targets == nil ||
		len(req.Targets.Ids) == 0 &&
			len(req.Targets.Names) == 0 &&
			len(req.Targets.Namespaces) == 0 &&
			len(req.Targets.Labels) == 0 {
		return nil, mir_models.ErrorNoDeviceTargetProvided
	}

	name := ""
	ns := ""
	id := ""
	if req.Meta != nil && req.Meta.Name != nil {
		name = *req.Meta.Name
	}
	if req.Meta != nil && req.Meta.Namespace != nil {
		ns = *req.Meta.Namespace
	}
	if req.Spec != nil && req.Spec.DeviceId != nil {
		id = *req.Spec.DeviceId
	}
	if err := s.validateDeviceUniqueness(req.Targets, name, ns, id); err != nil {
		return nil, err
	}

	// Update is full document
	// Change is a merge
	// Modify is a patch
	q := ""
	v := map[string]any{}
	q, v = createUpdateQueryForDevice(req.Targets, req)
	if q == "" {
		return s.ListDevice(&core_apiv1.ListDeviceRequest{
			Targets: req.Targets,
		})
	}
	respDb, err := executeQueryForType[[]mir_models.Device](s.db, q, v)
	if err != nil {
		return nil, errors.Wrap(err, mir_models.ErrorDbExecutingQuery.Error())
	}

	return respDb, nil
}

func (s *surrealMirStore) MergeDevice(targets *core_apiv1.Targets, patch json.RawMessage, op UpdateType) ([]mir_models.Device, error) {
	if targets == nil ||
		len(targets.Ids) == 0 &&
			len(targets.Names) == 0 &&
			len(targets.Namespaces) == 0 &&
			len(targets.Labels) == 0 {
		return nil, mir_models.ErrorNoDeviceTargetProvided
	}
	if op == MergePatch {
		// Validate json
		dev := mir_models.Device{}
		d := json.NewDecoder(bytes.NewReader(patch))
		d.DisallowUnknownFields()
		if err := d.Decode(&dev); err != nil {
			return nil, fmt.Errorf("unknown fields in json patch: %w", err)
		}

		if err := s.validateDeviceUniqueness(targets, dev.Meta.Name, dev.Meta.Namespace, dev.Spec.DeviceId); err != nil {
			return nil, err
		}

		var qSb strings.Builder
		if len(patch) > 0 {
			qSb.WriteString("UPDATE devices MERGE ")
			// NONE is a special value for null for SurrealDB
			patch = nullRegEx.ReplaceAll(patch, []byte("${1}NONE"))
			qSb.Write(patch)
			qSb.WriteString(" WHERE ")
			qSb.WriteString(createDeviceWhereStatementWithTargets(targets))
			qSb.WriteString(";")
		}
		sql := qSb.String()

		respDb, err := executeQueryForType[[]mir_models.Device](s.db, sql, map[string]any{})
		if err != nil {
			return nil, errors.Wrap(err, mir_models.ErrorDbExecutingQuery.Error())
		}
		return respDb, nil
	}
	return nil, errors.New("only MergePatch operation is implemented")
}

func (s *surrealMirStore) DeleteDevice(req *core_apiv1.DeleteDeviceRequest) ([]mir_models.Device, error) {
	if req.Targets == nil ||
		len(req.Targets.Ids) == 0 &&
			len(req.Targets.Names) == 0 &&
			len(req.Targets.Namespaces) == 0 &&
			len(req.Targets.Labels) == 0 {
		return nil, mir_models.ErrorNoDeviceTargetProvided
	}

	qList, vList := createListQueryForDevice(&core_apiv1.ListDeviceRequest{
		Targets: req.Targets,
	})
	respDbList, err := executeQueryForType[[]mir_models.Device](s.db, qList, vList)
	if err != nil {
		return nil, mir_models.ErrorDbExecutingQuery
	}

	q, v := createDeleteQueryForDevice(req)
	_, err = executeQueryForType[[]mir_models.Device](s.db, q, v)
	if err != nil {
		return nil, mir_models.ErrorDbExecutingQuery
	}

	return respDbList, nil
}

// If unique fields are provided, check if they are still unique
// - name/namespace composable unique key
// - deviceid unique key
// If deviceid or name/ns, only one device can be updated
// If name/ns, and no change, mean new device (upsert)
// If name only, multiple devices can be updated if no collision
// If namespace only, multiple devices can be updated if no collision
// DILEMMA: maybe we should limite name/namespace/deviceId changes to one device only
func (s *surrealMirStore) validateDeviceUniqueness(targets *core_apiv1.Targets, name, ns, deviceId string) error {
	if name != "" || ns != "" || deviceId != "" {
		changingDevs, err := s.ListDevice(&core_apiv1.ListDeviceRequest{
			Targets: targets,
		})
		if err != nil {
			return fmt.Errorf("%w: %w", mir_models.ErrorDbExecutingQuery, err)
		}

		if deviceId != "" {
			if len(changingDevs) > 1 {
				return fmt.Errorf("cannot update multiple devices as deviceId must be unique")
			} else if len(changingDevs) == 1 {
				// Check if deviceId is unique
				q, v := createIsDeviceUniqueQuery("", "", deviceId)
				respCheck, err := executeQueryForType[[]mir_models.Device](s.db, q, v)
				if err != nil {
					return fmt.Errorf("device unique check: %w: %w", mir_models.ErrorDbExecutingQuery, err)
				}
				if len(respCheck) > 0 && (respCheck[0].Meta.Name != changingDevs[0].Meta.Name || respCheck[0].Meta.Namespace != changingDevs[0].Meta.Namespace) {
					return fmt.Errorf("cannot update device has deviceId '%s' is already in use", deviceId)
				}
			}
		}
		if name != "" && ns != "" {
			if len(changingDevs) > 1 {
				return fmt.Errorf("cannot update multiple devices as name/namespace '%s/%s' must be unique", name, ns)
			} else if len(changingDevs) == 1 {
				// Check if name/ns is unique
				q, v := createIsDeviceUniqueQuery(name, ns, "")
				respCheck, err := executeQueryForType[[]mir_models.Device](s.db, q, v)
				if err != nil {
					return fmt.Errorf("%w: %w", mir_models.ErrorDbExecutingQuery, err)
				}
				if len(respCheck) > 0 && (respCheck[0].Meta.Name != changingDevs[0].Meta.Name || respCheck[0].Meta.Namespace != changingDevs[0].Meta.Namespace) {
					return fmt.Errorf("cannot update device has '%s/%s' is already in use", name, ns)
				}
			} else if len(changingDevs) == 0 {
				// Create device
				// We can't create it here since we wont get the device create event.
				return ErrorDeviceShouldBeCreated
			}
		} else if ns != "" {
			currentDevs, err := s.ListDevice(&core_apiv1.ListDeviceRequest{
				Targets: &core_apiv1.Targets{
					Namespaces: []string{ns},
				},
			})
			if err != nil {
				return fmt.Errorf("%w: %w", mir_models.ErrorDbExecutingQuery, err)
			}

			names := []string{}
			for _, d := range changingDevs {
				if slices.Contains(names, d.Meta.Name) {
					return fmt.Errorf("cannot update device as multiple device will have the same name '%s' in namespace '%s'", d.Meta.Name, ns)
				}
				names = append(names, d.Meta.Name)
			}
			names = []string{}
			for _, d := range currentDevs {
				names = append(names, d.Meta.Name)
			}
			for _, d := range changingDevs {
				if slices.Contains(names, d.Meta.Name) {
					return fmt.Errorf("cannot update device as name '%s' is already in use in namespace '%s'", d.Meta.Name, ns)
				}
			}
		} else if name != "" {
			currentDevs, err := s.ListDevice(&core_apiv1.ListDeviceRequest{
				Targets: &core_apiv1.Targets{
					Names: []string{name},
				},
			})
			if err != nil {
				return fmt.Errorf("%w: %w", mir_models.ErrorDbExecutingQuery, err)
			}

			namespaces := []string{}
			for _, d := range changingDevs {
				if slices.Contains(namespaces, d.Meta.Namespace) {
					return fmt.Errorf("cannot update device as multiple device will have the same name '%s' in namespace '%s'", name, d.Meta.Namespace)
				}
				namespaces = append(namespaces, d.Meta.Namespace)
			}
			for _, newD := range changingDevs {
				for _, oldD := range currentDevs {
					if newD.Meta.Namespace == oldD.Meta.Namespace {
						return fmt.Errorf("cannot update device as name '%s' is already in use in namespace '%s'", oldD.Meta.Name, oldD.Meta.Namespace)
					}
				}
			}
		}
	}
	return nil
}

func createIsDeviceUniqueQuery(name, ns, id string) (sql string, vars map[string]any) {
	var q strings.Builder
	q.WriteString("SELECT * FROM devices WHERE ")
	if id != "" {
		q.WriteString(fmt.Sprintf("spec.deviceId = \"%s\"", id))
	}
	if name != "" && ns != "" {
		if id != "" {
			q.WriteString(" OR ")
		}
		q.WriteString(fmt.Sprintf("(meta.name = \"%s\"", name))
		q.WriteString(" AND ")
		q.WriteString(fmt.Sprintf("meta.namespace = \"%s\")", ns))
	}
	q.WriteString(";")
	sql = q.String()
	return
}

func createListQueryForDevice(req *core_apiv1.ListDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("SELECT * FROM devices")
	where := createDeviceWhereStatementWithTargets(req.Targets)
	if len(where) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(where)
	}

	q.WriteString(";")
	sql = q.String()
	fmt.Println(sql)
	return
}

func createUpdateQueryForDevice(t *core_apiv1.Targets, upd *core_apiv1.UpdateDeviceRequest) (sql string, vars map[string]any) {
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
				if val == nil || val.Value == nil {
					sb.WriteString("NONE")
				} else {
					sb.WriteString(fmt.Sprintf("\"%s\"", val.GetValue()))
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
				if val == nil || val.Value == nil {
					sb.WriteString("NONE")
				} else {
					sb.WriteString(fmt.Sprintf("\"%s\"", val.GetValue()))
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
		if upd.Spec.DeviceId != nil && *upd.Spec.DeviceId != "" {
			sb.WriteString("deviceId: $ID,")
			vars["ID"] = *upd.Spec.DeviceId
		}
		if upd.Spec.Disabled != nil {
			sb.WriteString("disabled: $DIS,")
			vars["DIS"] = *upd.Spec.Disabled
		}
		if sb.Len() > 0 {
			q.WriteString("spec: {")
			q.WriteString(sb.String())
			q.WriteString("},")
		}
	}
	if upd.Props != nil {
		var sb strings.Builder
		if upd.Props.Desired != nil {
			x, _ := upd.Props.Desired.MarshalJSON()
			if len(x) > 0 {
				// Curlies are in the desired json already
				sb.WriteString("desired: ")
				sb.Write(nullRegEx.ReplaceAll(x, []byte("${1}NONE")))
			}
		}
		if sb.Len() > 0 {
			q.WriteString("properties: {")
			q.WriteString(sb.String())
			q.WriteString("},")
		}
	}
	if upd.Status != nil {
		var sb strings.Builder
		if upd.Status.LastHearthbeat != nil && !mir_models.AsGoTime(upd.Status.LastHearthbeat).IsZero() {
			sb.WriteString("lastHearthbeat: $BEAT,")
			vars["BEAT"] = mir_models.AsGoTime(upd.Status.LastHearthbeat)
		}
		if upd.Status.Online != nil {
			sb.WriteString("online: $ON,")
			vars["ON"] = upd.Status.Online
		}
		if upd.Status.Schema != nil {
			sb.WriteString("schema: {")
			if upd.Status.Schema.CompressedSchema != nil {
				sb.WriteString("compressedSchema: $COMPSCHEMA,")
				vars["COMPSCHEMA"] = upd.Status.Schema.CompressedSchema
			}
			if upd.Status.Schema.PackageNames != nil {
				sb.WriteString("packageNames: $PACKNAMES,")
				vars["PACKNAMES"] = upd.Status.Schema.PackageNames
			}
			if upd.Status.Schema.LastSchemaFetch != nil && !mir_models.AsGoTime(upd.Status.Schema.LastSchemaFetch).IsZero() {
				sb.WriteString("lastSchemaFetch: $LASTSCHFETCH,")
				vars["LASTSCHFETCH"] = mir_models.AsGoTime(upd.Status.Schema.LastSchemaFetch)
			}
			sb.WriteString("},")
		}
		if upd.Status.Properties != nil {
			sb.WriteString("properties: {")
			if upd.Status.Properties.Desired != nil {
				sb.WriteString("desired: {")
				for k, v := range upd.Status.Properties.Desired {
					sb.WriteString("\"")
					sb.WriteString(k)
					sb.WriteString("\"")
					sb.WriteString(": ")
					if v == nil {
						sb.WriteString("NONE")
					} else {
						sb.WriteString("\"")
						sb.WriteString(mir_models.AsGoTime(v).Format(time.RFC3339Nano))
						sb.WriteString("\"")
					}
					sb.WriteString(",")
				}
				sb.WriteString("},")
			}
			if upd.Status.Properties.Reported != nil {
				sb.WriteString("reported: {")
				for k, v := range upd.Status.Properties.Reported {
					sb.WriteString("\"")
					sb.WriteString(k)
					sb.WriteString("\"")
					sb.WriteString(": ")
					if v == nil {
						sb.WriteString("NONE")
					} else {
						sb.WriteString("\"")
						sb.WriteString(mir_models.AsGoTime(v).Format(time.RFC3339Nano))
						sb.WriteString("\"")
					}
					sb.WriteString(",")
				}
				sb.WriteString("},")
			}
			sb.WriteString("},")
		}
		if sb.Len() > 0 {
			q.WriteString("status: {")
			q.WriteString(sb.String())
			q.WriteString("},")
		}
	}

	var qSb strings.Builder
	if q.Len() > 0 {
		qSb.WriteString("UPDATE devices MERGE {")
		qSb.WriteString(q.String())
		qSb.WriteString("} WHERE ")
		qSb.WriteString(createDeviceWhereStatementWithTargets(t))
		qSb.WriteString(";")
	}
	sql = qSb.String()

	return
}

func createDeleteQueryForDevice(req *core_apiv1.DeleteDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createDeviceWhereStatementWithTargets(req.Targets))
	q.WriteString(";")
	sql = q.String()
	return
}

func createDeviceWhereStatementWithTargets(t *core_apiv1.Targets) string {
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
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Names) > 0 {
		var i []string
		for _, ns := range t.Names {
			i = append(i, fmt.Sprintf("meta.name CONTAINS \"%s\"", ns))
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
