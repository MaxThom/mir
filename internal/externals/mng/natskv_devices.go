package mng

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/pkg/errors"
)

var (
	scope             string = "default"
	apiVersionV1Alpha string = "mir/v1alpha"
	kindDevice        string = "device"
	bucketStrDevice   string = scope + "." + apiVersionV1Alpha + "." + kindDevice
)

func (s *natskvMirStore) ListDevice(t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error) {
	// keys, err := s.bucketDevice.Keys(context.Background(), nil)
	// if err != nil {
	// 	return []mir_v1.Device{}, fmt.Errorf("error listing devices in storage: %w", err)
	// }

	// if len(t.Names) == 0 || len(t.Names) == 1 &&
	// 	len(t.Namespaces) == 0 || len(t.Namespaces) == 1 {
	// }

	// pattern := "*"
	// matches := []string{}
	// for _, key := range keys {
	// 	if matched, _ := filepath.Match(pattern, key); matched {
	// 		val, _ := bucket.Get(ctx, key)
	// 		matches = append(matches, fmt.Sprintf("%s-%s", key, val.Value()))
	// 	}
	// }
	// fmt.Println(matches)
	return []mir_v1.Device{}, nil
}

func (s *natskvMirStore) CreateDevice(d mir_v1.Device) (mir_v1.Device, error) {
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
	// q, v := createSurrealIsDeviceUniqueQuery(d.Meta.Name, d.Meta.Namespace, d.Spec.DeviceId)
	// respCheck, err := executeSurrealQueryForType[[]mir_v1.Device](s.db, q, v)
	// if err != nil {
	// 	return mir_v1.Device{}, fmt.Errorf("%w for device %s/%s: %w", mir_v1.ErrorDbExecutingQuery, d.Meta.Name, d.Meta.Namespace, err)
	// }
	// if len(respCheck) > 0 {
	// 	return mir_v1.Device{}, fmt.Errorf("device %s/%s with deviceId %s already exist", d.Meta.Name, d.Meta.Namespace, d.Spec.DeviceId)
	// }

	// Create
	key := fmt.Sprintf("%s.%s.%s", d.Meta.Namespace, d.Meta.Name, generateLabelsHash(d.Meta.Labels))
	data, err := json.Marshal(d)
	if err != nil {
		return mir_v1.Device{}, fmt.Errorf("error marshaling device: %w", err)
	}
	if _, err := s.bucketDevice.Create(context.Background(), key, data); err != nil {
		return mir_v1.Device{}, fmt.Errorf("error creating device in storage: %w", err)
	}

	return d, nil
}

// UpdateDevice This method is too OP
// Maybe it need to be divided into Upsert and Patch
// Upsert is for apply and edit
// Patch is for patch
func (s *natskvMirStore) UpdateDevice(t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}

	return []mir_v1.Device{}, nil
}

func (s *natskvMirStore) MergeDevice(t mir_v1.DeviceTarget, patch json.RawMessage, op UpdateType) ([]mir_v1.Device, error) {
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

	}
	return nil, errors.New("only MergePatch operation is implemented")
}

func (s *natskvMirStore) DeleteDevice(t mir_v1.DeviceTarget) ([]mir_v1.Device, error) {
	if t.HasNoTarget() {
		return nil, mir_v1.ErrorNoDeviceTargetProvided
	}

	return []mir_v1.Device{}, nil
}

// If unique fields are provided, check if they are still unique
// - name/namespace composable unique key
// - deviceid unique key
// If deviceid or name/ns, only one device can be updated
// If name/ns, and no change, mean new device (upsert)
// If name only, multiple devices can be updated if no collision
// If namespace only, multiple devices can be updated if no collision
// DILEMMA: maybe we should limite name/namespace/deviceId changes to one device only
func (s *natskvMirStore) validateDeviceUniqueness(targets mir_v1.DeviceTarget, name, ns, deviceId string) error {
	return nil
}

func natskvCreateIsDeviceUniqueQuery(name, ns, id string) (sql string, vars map[string]any) {
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

func natskvCreateListQueryForDevice(t mir_v1.DeviceTarget, includeEvents bool) (sql string, vars map[string]any) {
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

func natskvCreateUpdateQueryForDevice(t mir_v1.DeviceTarget, d mir_v1.Device) (sql string, vars map[string]any) {
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

func natskvCreateDeleteQueryForDevice(t mir_v1.DeviceTarget) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createSurrealDeviceWhereStatementWithTargets(t))
	q.WriteString(";")
	sql = q.String()
	return
}

func natskvCreateDeviceWhereStatementWithTargets(t mir_v1.DeviceTarget) string {
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
