import type { Mir } from "./mir";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  ListTelemetryRequestSchema,
  ListTelemetryResponseSchema,
  QueryTelemetryRequestSchema,
  QueryTelemetryResponseSchema,
} from "./gen/proto/mir_api/v1/tlm_pb";
import type { QueryTelemetry_Row_DataPoint } from "./gen/proto/mir_api/v1/tlm_pb";
import { ClientSubject } from "./types";
import type { DeviceTarget } from "./models";
import {
  deviceTargetToProto,
  dateToTimestamp,
  timestampToDate,
} from "./transform";

// ─── SDK types ────────────────────────────────────────────────────────────────

type DeviceId = {
  id: string;
  name: string;
  namespace: string;
};

export type TelemetryDescriptor = {
  name: string;
  labels: Record<string, string>;
  fields: string[];
  exploreQuery: string;
  error: string;
};

export type TelemetryGroup = {
  ids: DeviceId[];
  descriptors: TelemetryDescriptor[];
  error: string;
};

export type QueryRow = {
  values: Record<string, number | boolean | string | Date | null>;
};

export type QueryData = {
  headers: string[];
  fieldUnits: Record<string, string>;
  rows: QueryRow[];
};

// ─── DataPoint extractor ──────────────────────────────────────────────────────

function extractDataPointValue(
  dp: QueryTelemetry_Row_DataPoint,
): number | boolean | string | Date | null {
  if (dp.valueInt32 !== undefined) return dp.valueInt32;
  if (dp.valueInt64 !== undefined) return Number(dp.valueInt64);
  if (dp.valueSint32 !== undefined) return dp.valueSint32;
  if (dp.valueSint64 !== undefined) return Number(dp.valueSint64);
  if (dp.valueUint32 !== undefined) return dp.valueUint32;
  if (dp.valueUint64 !== undefined) return Number(dp.valueUint64);
  if (dp.valueFixed32 !== undefined) return dp.valueFixed32;
  if (dp.valueFixed64 !== undefined) return Number(dp.valueFixed64);
  if (dp.valueSfixed32 !== undefined) return dp.valueSfixed32;
  if (dp.valueSfixed64 !== undefined) return Number(dp.valueSfixed64);
  if (dp.valueFloat !== undefined) return dp.valueFloat;
  if (dp.valueDouble !== undefined) return dp.valueDouble;
  if (dp.valueBool !== undefined) return dp.valueBool;
  if (dp.valueString !== undefined) return dp.valueString;
  if (dp.valueTimestamp)
    return timestampToDate(dp.valueTimestamp) ?? null;
  return null;
}

// ─── Routes ───────────────────────────────────────────────────────────────────

const listTelemetryRoute = new ClientSubject("tlm", "v1alpha", "list", []);
const queryTelemetryRoute = new ClientSubject("tlm", "v1alpha", "query", []);

export class ListTelemetry {
  constructor(private readonly mir: Mir) {}

  async request(t: DeviceTarget): Promise<TelemetryGroup[]> {
    const sbj = listTelemetryRoute.WithId(this.mir.getInstanceName());

    const req = create(ListTelemetryRequestSchema, {
      targets: deviceTargetToProto(t),
      refreshSchema: false,
    });
    const payload = toBinary(ListTelemetryRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(ListTelemetryResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return response.response.value.devicesTelemetry.map((g) => ({
        ids: g.ids.map((p) => ({
          id: p.deviceId,
          name: p.name,
          namespace: p.namespace,
        })),
        descriptors: g.tlmDescriptors.map((d) => ({
          name: d.name,
          labels: d.labels as Record<string, string>,
          fields: d.fields,
          exploreQuery: d.exploreQuery,
          error: d.error,
        })),
        error: g.error,
      }));
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}

export class QueryTelemetry {
  constructor(private readonly mir: Mir) {}

  async request(
    t: DeviceTarget,
    measurement: string,
    fields: string[],
    start: Date,
    end: Date,
    aggregationWindow?: string,
  ): Promise<QueryData> {
    const sbj = queryTelemetryRoute.WithId(this.mir.getInstanceName());

    const req = create(QueryTelemetryRequestSchema, {
      targets: deviceTargetToProto(t),
      measurement,
      fields,
      startTime: dateToTimestamp(start),
      endTime: dateToTimestamp(end),
      aggregationWindow: aggregationWindow ?? '',
    });
    const payload = toBinary(QueryTelemetryRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(QueryTelemetryResponseSchema, msg.data);
    if (response.response.case === "ok") {
      const qt = response.response.value;
      return {
        headers: qt.headers,
        fieldUnits: Object.fromEntries(qt.headers.map((h, i) => [h, qt.units[i] ?? ''])),
        rows: qt.rows.map((row) => {
          const values: Record<string, number | boolean | string | Date | null> = {};
          qt.headers.forEach((h, i) => {
            values[h] = extractDataPointValue(row.datapoints[i]);
          });
          return { values };
        }),
      };
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return { headers: [], fieldUnits: {}, rows: [] };
  }
}
