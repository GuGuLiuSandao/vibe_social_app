import { describe, expect, it } from "vitest";

import { WsMessageType } from "../proto/ws_pb.ts";
import { buildAccountPing, buildAuthRequest, buildWsUrl } from "./ws.js";

describe("DLQ_TC_009_CLIENT_002 buildWsUrl", () => {
  it("builds the configured bare endpoint and encodes the token", () => {
    expect(buildWsUrl("10000001", "a+b/c=")).toBe(
      "ws://localhost:8080/ws?uid=10000001&token=a%2Bb%2Fc%3D",
    );
  });

  it("omits an absent token", () => {
    expect(buildWsUrl("10000001", "")).toBe("ws://localhost:8080/ws?uid=10000001");
  });
});

describe("DLQ_TC_010_CLIENT_002 protobuf request builders", () => {
  it("preserves ping request identity and type", () => {
    const message = buildAccountPing(123n);
    expect(message.requestId).toBe(123n);
    expect(message.type).toBe(WsMessageType.WS_TYPE_PING);
    expect(message.payload.case).toBe("account");
    expect(message.payload.value.payload.case).toBe("ping");
  });

  it("preserves auth UID and request identity", () => {
    const message = buildAuthRequest("10000001", 456n);
    expect(message.requestId).toBe(456n);
    expect(message.type).toBe(WsMessageType.WS_TYPE_AUTH);
    expect(message.payload.value.payload.case).toBe("auth");
    expect(message.payload.value.payload.value.uid).toBe(10000001n);
  });
});
