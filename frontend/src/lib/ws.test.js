import { describe, expect, it } from "vitest";
import { WsMessageType } from "../proto/ws_pb.ts";
import {
  buildAccountPing,
  buildAuthRequest,
  buildUpdateProfileRequest,
  buildUploadAvatarRequest,
  buildWsUrl,
  decodeWsMessage,
  encodeWsMessage,
} from "./ws";

describe("ws url builder", () => {
  it("builds whitelist url without token", () => {
    expect(buildWsUrl(20000001, "")).toBe("ws://localhost:8080/ws?uid=20000001");
  });

  it("builds normal url with token", () => {
    expect(buildWsUrl(10000001, "abc")).toBe(
      "ws://localhost:8080/ws?uid=10000001&token=abc"
    );
  });
});

describe("ws protobuf helpers", () => {
  it("builds and encodes account ping", () => {
    const message = buildAccountPing(123n);
    const binary = encodeWsMessage(message);
    const decoded = decodeWsMessage(binary);

    expect(decoded.type).toBe(WsMessageType.WS_TYPE_PING);
    expect(decoded.payload.case).toBe("account");
    expect(decoded.payload.value.payload.case).toBe("ping");
    expect(decoded.requestId).toBe(123n);
  });

  it("builds auth and update profile account payloads", () => {
    const auth = buildAuthRequest("10000001", 200n);
    const update = buildUpdateProfileRequest({ nickname: "neo", bio: "hi" }, 201n);

    expect(auth.type).toBe(WsMessageType.WS_TYPE_AUTH);
    expect(auth.payload.value.payload.case).toBe("auth");

    expect(update.type).toBe(WsMessageType.WS_TYPE_ACCOUNT_UPDATE_PROFILE);
    expect(update.payload.value.payload.case).toBe("updateProfile");
  });

  it("builds upload avatar payload", () => {
    const data = new Uint8Array([1, 2, 3]);
    const upload = buildUploadAvatarRequest("avatar.png", data, 202n);

    expect(upload.type).toBe(WsMessageType.WS_TYPE_ACCOUNT_UPLOAD_AVATAR);
    expect(upload.payload.value.payload.case).toBe("uploadAvatar");
    expect(upload.payload.value.payload.value.filename).toBe("avatar.png");
    expect(upload.payload.value.payload.value.data).toEqual(data);
  });
});
