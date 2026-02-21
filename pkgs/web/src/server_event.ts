import { Mir } from "./mir";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  ListEventsRequestSchema,
  ListEventsResponseSchema,
} from "./gen/proto/mir_api/v1/event_pb";
import { ClientSubject } from "./types";
import type { MirEvent, EventTarget } from "./models";
import { eventsFromProto, eventTargetToProto } from "./transform";

const listEventRoute = new ClientSubject("evt", "v1alpha", "list", []);

export class ListEvent {
  constructor(private readonly mir: Mir) {}

  async request(t: EventTarget): Promise<MirEvent[]> {
    const sbj = listEventRoute.WithId(this.mir.getInstanceName());

    const req = create(ListEventsRequestSchema, {
      target: eventTargetToProto(t),
    });
    const payload = toBinary(ListEventsRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(ListEventsResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return eventsFromProto(response.response.value.events);
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}
