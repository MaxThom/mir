package mng

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

var (
	ErrorListingDevices        = errors.New("error listing devices from database")
	ErrorNoDeviceFound         = errors.New("no device found with current targets criteria")
	ErrorDeviceShouldBeCreated = errors.New("device should be created")
)

const (
	surrealDeviceTable string = "devices"
)

type deviceWithId struct {
	Id string `json:"id"`
	mir_v1.Device
}

func (s *surrealMirStore) ListDevice(t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error) {
	q, v := createListQueryForDevice(t, includeEvents)
	devs, err := surreal.Query[[]mir_v1.Device](s.db, q, v)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrorListingDevices, err)
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
	q, v := createIsDeviceUniqueQuery(d.Meta.Name, d.Meta.Namespace, d.Spec.DeviceId)
	respCheck, err := surreal.Query[[]mir_v1.Device](s.db, q, v)
	if err != nil {
		return mir_v1.Device{}, fmt.Errorf("%w for device %s/%s: %w", mir_v1.ErrorDbExecutingQuery, d.Meta.Name, d.Meta.Namespace, err)
	}
	if len(respCheck) > 0 {
		return mir_v1.Device{}, fmt.Errorf("device %s/%s with deviceId %s already exist", d.Meta.Name, d.Meta.Namespace, d.Spec.DeviceId)
	}

	// Create
	respDb, err := surreal.Create[mir_v1.Device](s.db, surrealDeviceTable, d)
	if err != nil {
		return mir_v1.Device{}, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
	}
	return *respDb, nil
}

func (s *surrealMirStore) CreateDevices(devs []mir_v1.Device) ([]mir_v1.Device, error) {
	// Validate
	var errs error
	for i := 0; i < len(devs); i++ {
		if devs[i].Spec.DeviceId == "" {
			errs = errors.Join(errs, fmt.Errorf("device at index %d is missing its id", i))
			devs = append(devs[:i], devs[i+1:]...)
			i -= 1
			continue
		}
		if devs[i].Meta.Name == "" {
			devs[i].Meta.Name = devs[i].Spec.DeviceId
		}
		if devs[i].Meta.Namespace == "" {
			devs[i].Meta.Namespace = "default"
		}
		devs[i].Status = mir_v1.DeviceStatus{}
	}

	// Remove duplicates based on deviceId or name/namespace combination
	seen := make(map[string]bool)
	uniqueDevs := make([]mir_v1.Device, 0, len(devs))

	for _, d := range devs {
		deviceIdKey := d.Spec.DeviceId
		nameNsKey := d.Meta.Name + "/" + d.Meta.Namespace
		if !seen[deviceIdKey] && !seen[nameNsKey] {
			seen[deviceIdKey] = true
			seen[nameNsKey] = true
			uniqueDevs = append(uniqueDevs, d)
		} else if seen[deviceIdKey] {
			errs = errors.Join(errs, fmt.Errorf("device with id %s is duplicated", deviceIdKey))
		} else if seen[nameNsKey] {
			errs = errors.Join(errs, fmt.Errorf("device with name/namespace %s is duplicated", nameNsKey))
		}
	}
	devs = uniqueDevs
	if errs != nil {
		return []mir_v1.Device{}, errs
	}

	// Check duplicates already in database
	var q strings.Builder
	q.WriteString("BEGIN TRANSACTION;\n")
	for _, d := range devs {
		devQ, _ := createIsDeviceUniqueQuery(d.Meta.Name, d.Meta.Namespace, d.Spec.DeviceId)
		q.WriteString(devQ + "\n")
	}
	q.WriteString("COMMIT TRANSACTION;\n")

	resp, err := surreal.QueryMultiple[[]mir_v1.Device](s.db, q.String(), map[string]any{})
	if err != nil {
		return []mir_v1.Device{}, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
	}

	// Flatten the nested response arrays into a single array
	var flatResp []mir_v1.Device
	for _, devices := range resp {
		flatResp = append(flatResp, devices...)
	}

	if len(flatResp) > 0 {
		// Mean some devices already exist
		// Which one so we remove from the created list
		for _, existingD := range flatResp {
			for i := 0; i < len(devs); i++ {
				if existingD.Meta.Name == devs[i].Meta.Name &&
					existingD.Meta.Namespace == devs[i].Meta.Namespace ||
					existingD.Spec.DeviceId == devs[i].Spec.DeviceId {
					errs = errors.Join(errs, fmt.Errorf("device with id %s and name/namespace %s already exist", devs[i].Spec.DeviceId, devs[i].Meta.Name+"/"+devs[i].Meta.Namespace))
					devs = append(devs[:i], devs[i+1:]...)
					break
				}
			}
		}
	}

	// Create
	respDb, err := surreal.Insert[mir_v1.Device](s.db, surrealDeviceTable, devs)
	if err != nil {
		return []mir_v1.Device{}, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
	}
	return *respDb, nil
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

	// divided the query into multiple as we reach surreal limits
	if len(t.Ids) > 100 {
		var allResults []mir_v1.Device
		for i := 0; i < len(t.Ids); i += 100 {
			end := i + 100
			if end > len(t.Ids) {
				end = len(t.Ids)
			}
			batchTarget := t
			batchTarget.Ids = t.Ids[i:end]

			q := ""
			v := map[string]any{}
			q, v = createUpdateQueryForDevice(batchTarget, d)
			if q == "" {
				results, err := s.ListDevice(batchTarget, false)
				if err != nil {
					return nil, err
				}
				allResults = append(allResults, results...)
				continue
			}
			respDb, err := surreal.Query[[]mir_v1.Device](s.db, q, v)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
			}
			allResults = append(allResults, respDb...)
		}
		return allResults, nil
	} else {
		// Update is full document
		// Change is a merge
		// Modify is a patch
		q := ""
		v := map[string]any{}
		q, v = createUpdateQueryForDevice(t, d)
		if q == "" {
			return s.ListDevice(t, false)
		}
		respDb, err := surreal.Query[[]mir_v1.Device](s.db, q, v)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
		}

		return respDb, nil
	}
}

// UpdateDeviceHeartbeats performs bulk heartbeat updates for multiple devices in a single query
func (s *surrealMirStore) UpdateDeviceHello(updates map[mir_v1.DeviceId]mir_v1.DeviceHello) ([]mir_v1.Device, error) {
	if len(updates) == 0 {
		return []mir_v1.Device{}, nil
	}
	vars := map[string]any{}

	var q strings.Builder
	q.WriteString("BEGIN TRANSACTION;\n")

	for deviceId, hb := range updates {
		var sbStatus strings.Builder
		if hb.Schema != nil {
			sch, err := mir_v1.NewSchemaFromProtoSchema(hb.Schema)
			if err == nil {
				sch.LastSchemaFetch = &models.CustomDateTime{Time: hb.Hearthbeat}
				var sbStatusSchema strings.Builder
				if sch.CompressedSchema != nil {
					sbStatusSchema.WriteString("compressedSchema: $" + string(deviceId) + "_SCH,")
					vars[string(deviceId)+"_SCH"] = sch.CompressedSchema
				}
				if sch.PackageNames != nil {
					sbStatusSchema.WriteString("packageNames: $" + string(deviceId) + "_PKGS,")
					vars[string(deviceId)+"_PKGS"] = sch.PackageNames
				}
				if sch.LastSchemaFetch != nil && !sch.LastSchemaFetch.IsZero() {
					sbStatusSchema.WriteString("lastSchemaFetch: $" + string(deviceId) + "_FETCH,")
					vars[string(deviceId)+"_FETCH"] = sch.LastSchemaFetch
				}
				if sbStatusSchema.Len() > 0 {
					sbStatus.WriteString(", schema: { ")
					sbStatus.WriteString(sbStatusSchema.String())
					sbStatus.WriteString(" }")
				}
			}
		}

		q.WriteString(fmt.Sprintf(
			"UPDATE devices MERGE { status: { online: true, lastHearthbeat: d\"%s\"%s } } WHERE spec.deviceId = \"%s\";\n",
			hb.Hearthbeat.Format(time.RFC3339Nano),
			sbStatus.String(),
			deviceId,
		))
	}

	q.WriteString("COMMIT TRANSACTION;")

	resp, err := surreal.QueryMultiple[[]mir_v1.Device](s.db, q.String(), vars)
	if err != nil {
		return []mir_v1.Device{}, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
	}

	// Flatten the nested response arrays into a single array
	var flatResp []mir_v1.Device
	for _, devices := range resp {
		flatResp = append(flatResp, devices...)
	}

	return flatResp, nil
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
			qSb.WriteString(createDeviceWhereStatementWithTargets(t))
			qSb.WriteString(";")
		}
		sql := qSb.String()

		respDb, err := surreal.Query[[]mir_v1.Device](s.db, sql, map[string]any{})
		if err != nil {
			return nil, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
		}
		return respDb, nil
	}
	return nil, errors.New("only MergePatch operation is implemented")
}

func (s *surrealMirStore) DeleteDevice(t mir_v1.DeviceTarget) ([]mir_v1.Device, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}

	qList, vList := createListQueryForDevice(t, false)
	respDbList, err := surreal.Query[[]mir_v1.Device](s.db, qList, vList)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
	}

	q, v := createDeleteQueryForDevice(t)
	_, err = surreal.Query[[]mir_v1.Device](s.db, q, v)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", mir_v1.ErrorDbExecutingQuery, err)
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
				q, v := createIsDeviceUniqueQuery("", "", deviceId)
				respCheck, err := surreal.Query[[]mir_v1.Device](s.db, q, v)
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
				q, v := createIsDeviceUniqueQuery(name, ns, "")
				respCheck, err := surreal.Query[[]mir_v1.Device](s.db, q, v)
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

func createListQueryForDevice(t mir_v1.DeviceTarget, includeEvents bool) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	if includeEvents {
		q.WriteString("SELECT *, ")
		q.WriteString("(")
		q.WriteString("SELECT spec.type as type, spec.message ?? '' as message, spec.reason as reason, status.firstAt ?? NULL as firstAt FROM events")
		q.WriteString(" WHERE $parent.meta.name = spec.relatedObject.meta.name AND $parent.meta.namespace = spec.relatedObject.meta.namespace ORDER firstAt DESC LIMIT 5")
		q.WriteString(") as status.events")
		q.WriteString(" FROM devices")
	} else {
		q.WriteString("SELECT * FROM devices")
	}
	where := createDeviceWhereStatementWithTargets(t)
	if len(where) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(where)
	}

	q.WriteString(";")
	sql = q.String()
	return
}

func createUpdateQueryForDevice(t mir_v1.DeviceTarget, d mir_v1.Device) (sql string, vars map[string]any) {
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
	if len(d.Meta.Labels) > 0 {
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
	if len(d.Meta.Annotations) > 0 {
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
	if len(d.Properties.Desired) > 0 {
		x, _ := json.Marshal(d.Properties.Desired)
		if len(x) > 0 {
			// Curlies are in the desired json already
			sbProps.WriteString("desired: ")
			sbProps.Write(nullRegEx.ReplaceAll(x, []byte("${1}NONE")))
			sbProps.WriteString(",")
		}
	}
	if len(d.Properties.Reported) > 0 {
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
				sbStatusProps.WriteString("d\"")
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
				sbStatusProps.WriteString("d\"")
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
		qSb.WriteString(createDeviceWhereStatementWithTargets(t))
		qSb.WriteString(";")
	}
	sql = qSb.String()

	return
}

func createDeleteQueryForDevice(t mir_v1.DeviceTarget) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createDeviceWhereStatementWithTargets(t))
	q.WriteString(";")
	sql = q.String()
	return
}

func createDeviceWhereStatementWithTargets(t mir_v1.DeviceTarget) string {
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
