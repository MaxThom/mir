package mng

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/pkg/errors"
	"github.com/surrealdb/surrealdb.go"
)

var ()

type deviceWithId struct {
	Id string `json:"id"`
	mir_v1.Device
}

func (s *surrealMirStore) ListDevice(t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error) {
	q, v := createSurrealListQueryForDevice(t, includeEvents)
	devs, err := executeSurrealQueryForType[[]mir_v1.Device](s.db, q, v)
	if err != nil {
		return nil, errors.Wrap(err, ErrorListingDevices.Error())
	}
	return devs, nil
}

func (s *surrealMirStore) CreateDevice(d mir_v1.Device) (mir_v1.Device, error) {
	// Validate
	if d.Spec.DeviceId == "" {
		return mir_v1.Device{}, fmt.Errorf("device id is missing")
	}
	if d.Meta.Name == "" {
		d.Meta.Name = d.Spec.DeviceId
	}
	if d.Meta.Namespace == "" {
		d.Meta.Namespace = "default"
	}
	// Device status is readonly and set by the system
	d.Status = mir_v1.DeviceStatus{}
	q, v := createSurrealIsDeviceUniqueQuery(d.Meta.Name, d.Meta.Namespace, d.Spec.DeviceId)
	respCheck, err := executeSurrealQueryForType[[]mir_v1.Device](s.db, q, v)
	if err != nil {
		return mir_v1.Device{}, fmt.Errorf("%w for device %s/%s: %w", mir_v1.ErrorDbExecutingQuery, d.Meta.Name, d.Meta.Namespace, err)
	}
	if len(respCheck) > 0 {
		return mir_v1.Device{}, fmt.Errorf("device %s/%s with deviceId %s already exist", d.Meta.Name, d.Meta.Namespace, d.Spec.DeviceId)
	}

	// Create
	respDb, err := s.db.Create("devices", d)
	if err != nil {
		return mir_v1.Device{}, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
	}
	newDev := []mir_v1.Device{}
	err = surrealdb.Unmarshal(respDb, &newDev)
	if err != nil {
		return mir_v1.Device{}, fmt.Errorf("%w for device %s/%s: %w", mir_v1.ErrorDbDeserializingResponse, d.Meta.Name, d.Meta.Namespace, err)
	}
	return newDev[0], nil
}

// UpdateDevice This method is too OP
// Maybe it need to be divided into Upsert and Patch
// Upsert is for apply and edit
// Patch is for patch
func (s *surrealMirStore) UpdateDevice(t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}

	if err := s.validateDeviceUniqueness(t, d.Meta.Name, d.Meta.Namespace, d.Spec.DeviceId); err != nil {
		return nil, err
	}

	// Update is full document
	// Change is a merge
	// Modify is a patch
	q := ""
	v := map[string]any{}
	q, v = createSurrealUpdateQueryForDevice(t, d)
	if q == "" {
		return s.ListDevice(t, false)
	}
	respDb, err := executeSurrealQueryForType[[]mir_v1.Device](s.db, q, v)
	if err != nil {
		return nil, errors.Wrap(err, mir_v1.ErrorDbExecutingQuery.Error())
	}

	return respDb, nil
}

func (s *surrealMirStore) MergeDevice(t mir_v1.DeviceTarget, patch json.RawMessage, op UpdateType) ([]mir_v1.Device, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}
	if op == MergePatch {
		// Validate json
		dev := mir_v1.Device{}
		d := json.NewDecoder(bytes.NewReader(patch))
		d.DisallowUnknownFields()
		if err := d.Decode(&dev); err != nil {
			return nil, fmt.Errorf("unknown fields in json patch: %w", err)
		}

		if err := s.validateDeviceUniqueness(t, dev.Meta.Name, dev.Meta.Namespace, dev.Spec.DeviceId); err != nil {
			return nil, err
		}

		var qSb strings.Builder
		if len(patch) > 0 {
			qSb.WriteString("UPDATE devices MERGE ")
			// NONE is a special value for null for SurrealDB
			patch = nullRegEx.ReplaceAll(patch, []byte("${1}NONE"))
			qSb.Write(patch)
			qSb.WriteString(" WHERE ")
			qSb.WriteString(createSurrealDeviceWhereStatementWithTargets(t))
			qSb.WriteString(";")
		}
		sql := qSb.String()

		respDb, err := executeSurrealQueryForType[[]mir_v1.Device](s.db, sql, map[string]any{})
		if err != nil {
			return nil, errors.Wrap(err, mir_v1.ErrorDbExecutingQuery.Error())
		}
		return respDb, nil
	}
	return nil, errors.New("only MergePatch operation is implemented")
}

func (s *surrealMirStore) DeleteDevice(t mir_v1.DeviceTarget) ([]mir_v1.Device, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}

	qList, vList := createSurrealListQueryForDevice(t, false)
	respDbList, err := executeSurrealQueryForType[[]mir_v1.Device](s.db, qList, vList)
	if err != nil {
		return nil, mir_v1.ErrorDbExecutingQuery
	}

	q, v := createSurrealDeleteQueryForDevice(t)
	_, err = executeSurrealQueryForType[[]mir_v1.Device](s.db, q, v)
	if err != nil {
		return nil, mir_v1.ErrorDbExecutingQuery
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
func (s *surrealMirStore) validateDeviceUniqueness(targets mir_v1.DeviceTarget, name, ns, deviceId string) error {
	if name != "" || ns != "" || deviceId != "" {
		changingDevs, err := s.ListDevice(targets, false)
		if err != nil {
			return fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
		}

		if deviceId != "" {
			if len(changingDevs) > 1 {
				return fmt.Errorf("cannot update multiple devices as deviceId must be unique")
			} else if len(changingDevs) == 1 {
				// Check if deviceId is unique
				q, v := createSurrealIsDeviceUniqueQuery("", "", deviceId)
				respCheck, err := executeSurrealQueryForType[[]mir_v1.Device](s.db, q, v)
				if err != nil {
					return fmt.Errorf("device unique check: %w: %w", mir_v1.ErrorDbExecutingQuery, err)
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
				q, v := createSurrealIsDeviceUniqueQuery(name, ns, "")
				respCheck, err := executeSurrealQueryForType[[]mir_v1.Device](s.db, q, v)
				if err != nil {
					return fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
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
			currentDevs, err := s.ListDevice(mir_v1.DeviceTarget{
				Namespaces: []string{ns},
			}, false)
			if err != nil {
				return fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
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
			currentDevs, err := s.ListDevice(mir_v1.DeviceTarget{
				Names: []string{name},
			}, false)
			if err != nil {
				return fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
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

func createSurrealIsDeviceUniqueQuery(name, ns, id string) (sql string, vars map[string]any) {
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

func createSurrealListQueryForDevice(t mir_v1.DeviceTarget, includeEvents bool) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	if includeEvents {
		q.WriteString("SELECT *, ")
		q.WriteString("(")
		q.WriteString("SELECT spec.type as type, spec.message as message, spec.reason as reason, status.firstAt as firstAt FROM events")
		q.WriteString(" WHERE $parent.meta.name = spec.relatedObject.meta.name AND $parent.meta.namespace = spec.relatedObject.meta.namespace ORDER firstAt DESC LIMIT 5")
		q.WriteString(") as status.events")
		q.WriteString(" FROM devices")
	} else {
		q.WriteString("SELECT * FROM devices")
	}
	where := createSurrealDeviceWhereStatementWithTargets(t)
	if len(where) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(where)
	}

	q.WriteString(";")
	sql = q.String()
	return
}

func createSurrealUpdateQueryForDevice(t mir_v1.DeviceTarget, d mir_v1.Device) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	// META
	var sbMeta strings.Builder
	if d.Meta.Name != "" {
		sbMeta.WriteString("name: $NAME,")
		vars["NAME"] = d.Meta.Name
	}
	if d.Meta.Namespace != "" {
		sbMeta.WriteString("namespace: $NS,")
		vars["NS"] = d.Meta.Namespace
	}
	if d.Meta.Labels != nil && len(d.Meta.Labels) > 0 {
		sbMeta.WriteString("labels: {")
		for key, val := range d.Meta.Labels {
			sbMeta.WriteString("\"")
			sbMeta.WriteString(key)
			sbMeta.WriteString("\"")
			sbMeta.WriteString(": ")
			if val == "" {
				sbMeta.WriteString("NONE")
			} else {
				sbMeta.WriteString(fmt.Sprintf("\"%s\"", val))
			}
			sbMeta.WriteString(",")
		}
		sbMeta.WriteString("},")
	}
	if d.Meta.Annotations != nil && len(d.Meta.Annotations) > 0 {
		sbMeta.WriteString("annotations: {")
		for key, val := range d.Meta.Annotations {
			sbMeta.WriteString("\"")
			sbMeta.WriteString(key)
			sbMeta.WriteString("\"")
			sbMeta.WriteString(": ")
			if val == "" {
				sbMeta.WriteString("NONE")
			} else {
				sbMeta.WriteString(fmt.Sprintf("\"%s\"", val))
			}
			sbMeta.WriteString(",")
		}
		sbMeta.WriteString("},")
	}
	if sbMeta.Len() > 0 {
		q.WriteString("meta: {")
		q.WriteString(sbMeta.String())
		q.WriteString("},")
	}

	// SPEC
	var sbSpec strings.Builder
	if d.Spec.DeviceId != "" {
		sbSpec.WriteString("deviceId: $ID,")
		vars["ID"] = d.Spec.DeviceId
	}
	if d.Spec.Disabled != nil {
		sbSpec.WriteString("disabled: $DIS,")
		vars["DIS"] = *d.Spec.Disabled
	}
	if sbSpec.Len() > 0 {
		q.WriteString("spec: {")
		q.WriteString(sbSpec.String())
		q.WriteString("},")
	}

	// PROPS
	var sbProps strings.Builder
	if d.Properties.Desired != nil && len(d.Properties.Desired) > 0 {
		x, _ := json.Marshal(d.Properties.Desired)
		if len(x) > 0 {
			// Curlies are in the desired json already
			sbProps.WriteString("desired: ")
			sbProps.Write(nullRegEx.ReplaceAll(x, []byte("${1}NONE")))
			sbProps.WriteString(",")
		}
	}
	if d.Properties.Reported != nil && len(d.Properties.Reported) > 0 {
		x, _ := json.Marshal(d.Properties.Reported)
		if len(x) > 0 {
			// Curlies are in the desired json already
			sbProps.WriteString("reported: ")
			sbProps.Write(nullRegEx.ReplaceAll(x, []byte("${1}NONE")))
			sbProps.WriteString(",")
		}
	}
	if sbProps.Len() > 0 {
		q.WriteString("properties: {")
		q.WriteString(sbProps.String())
		q.WriteString("},")
	}

	// STATUS
	var sbStatus strings.Builder
	if d.Status.LastHearthbeat != nil && !d.Status.LastHearthbeat.IsZero() {
		sbStatus.WriteString("lastHearthbeat: $BEAT,")
		vars["BEAT"] = d.Status.LastHearthbeat
	}
	if d.Status.Online != nil {
		sbStatus.WriteString("online: $ON,")
		vars["ON"] = d.Status.Online
	}

	// STATUS SCHEMA
	var sbStatusSchema strings.Builder
	if d.Status.Schema.CompressedSchema != nil {
		sbStatusSchema.WriteString("compressedSchema: $COMPSCHEMA,")
		vars["COMPSCHEMA"] = d.Status.Schema.CompressedSchema
	}
	if d.Status.Schema.PackageNames != nil {
		sbStatusSchema.WriteString("packageNames: $PACKNAMES,")
		vars["PACKNAMES"] = d.Status.Schema.PackageNames
	}
	if d.Status.Schema.LastSchemaFetch != nil && !d.Status.Schema.LastSchemaFetch.IsZero() {
		sbStatusSchema.WriteString("lastSchemaFetch: $LASTSCHFETCH,")
		vars["LASTSCHFETCH"] = d.Status.Schema.LastSchemaFetch
	}
	if sbStatusSchema.Len() > 0 {
		sbStatus.WriteString("schema: {")
		sbStatus.WriteString(sbStatusSchema.String())
		sbStatus.WriteString("},")
	}

	// STATUS PROPERTIES
	var sbStatusProps strings.Builder
	if d.Status.Properties.Desired != nil {
		sbStatusProps.WriteString("desired: {")
		for k, v := range d.Status.Properties.Desired {
			sbStatusProps.WriteString("\"")
			sbStatusProps.WriteString(k)
			sbStatusProps.WriteString("\"")
			sbStatusProps.WriteString(": ")
			if v.IsZero() {
				sbStatusProps.WriteString("NONE")
			} else {
				sbStatusProps.WriteString("\"")
				sbStatusProps.WriteString(v.Format(time.RFC3339Nano))
				sbStatusProps.WriteString("\"")
			}
			sbStatusProps.WriteString(",")
		}
		sbStatusProps.WriteString("},")
	}
	if d.Status.Properties.Reported != nil {
		sbStatusProps.WriteString("reported: {")
		for k, v := range d.Status.Properties.Reported {
			sbStatusProps.WriteString("\"")
			sbStatusProps.WriteString(k)
			sbStatusProps.WriteString("\"")
			sbStatusProps.WriteString(": ")
			if v.IsZero() {
				sbStatusProps.WriteString("NONE")
			} else {
				sbStatusProps.WriteString("\"")
				sbStatusProps.WriteString(v.Format(time.RFC3339Nano))
				sbStatusProps.WriteString("\"")
			}
			sbStatusProps.WriteString(",")
		}
		sbStatusProps.WriteString("},")
	}

	if sbStatusProps.Len() > 0 {
		sbStatus.WriteString("properties: {")
		sbStatus.WriteString(sbStatusProps.String())
		sbStatus.WriteString("},")
	}

	if sbStatus.Len() > 0 {
		q.WriteString("status: {")
		q.WriteString(sbStatus.String())
		q.WriteString("},")
	}

	var qSb strings.Builder
	if q.Len() > 0 {
		qSb.WriteString("UPDATE devices MERGE {")
		qSb.WriteString(q.String())
		qSb.WriteString("} WHERE ")
		qSb.WriteString(createSurrealDeviceWhereStatementWithTargets(t))
		qSb.WriteString(";")
	}
	sql = qSb.String()

	return
}

func createSurrealDeleteQueryForDevice(t mir_v1.DeviceTarget) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createSurrealDeviceWhereStatementWithTargets(t))
	q.WriteString(";")
	sql = q.String()
	return
}

func createSurrealDeviceWhereStatementWithTargets(t mir_v1.DeviceTarget) string {
	var q strings.Builder

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
			i = append(i, fmt.Sprintf("meta.name = \"%s\"", ns))
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
