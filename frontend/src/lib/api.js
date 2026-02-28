import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import {
  LoginRequestSchema,
  LoginResponseSchema,
  RegisterRequestSchema,
  RegisterResponseSchema,
} from "../proto/account/account_pb.ts";
import { ErrorCode } from "../proto/common/error_code_pb.ts";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080";

async function loginWithProtobuf({ email = "", password = "", uid = 0 }) {
  const message = create(LoginRequestSchema, {
    email,
    password,
    uid: uid ? BigInt(uid) : 0n,
  });
  const body = toBinary(LoginRequestSchema, message);
  const response = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-protobuf",
      Accept: "application/x-protobuf",
    },
    body,
  });
  const buffer = new Uint8Array(await response.arrayBuffer());
  if (buffer.length === 0) {
    if (!response.ok) {
      throw new Error("登录失败");
    }
    throw new Error("登录响应为空");
  }
  const data = fromBinary(LoginResponseSchema, buffer);
  if (!response.ok || data.errorCode !== ErrorCode.OK) {
    throw new Error(data.message || "登录失败");
  }
  if (data.user && typeof data.user.id === "bigint") {
    data.user = { ...data.user, id: String(data.user.id) };
  }
  return { user: data.user, token: data.token };
}

export function loginWithPassword(email, password) {
  return loginWithProtobuf({ email, password });
}

export function loginWithUid(uid) {
  return loginWithProtobuf({ uid });
}

export async function registerWithProtobuf(username, email, password) {
  const message = create(RegisterRequestSchema, { username, email, password });
  const body = toBinary(RegisterRequestSchema, message);
  const response = await fetch(`${API_BASE}/api/v1/auth/register`, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-protobuf",
      Accept: "application/x-protobuf",
    },
    body,
  });
  const buffer = new Uint8Array(await response.arrayBuffer());
  if (buffer.length === 0) {
    if (!response.ok) {
      throw new Error("注册失败");
    }
    throw new Error("注册响应为空");
  }
  const data = fromBinary(RegisterResponseSchema, buffer);
  if (!response.ok || data.errorCode !== ErrorCode.OK) {
    throw new Error(data.message || "注册失败");
  }
  if (data.user && typeof data.user.id === "bigint") {
    data.user = { ...data.user, id: String(data.user.id) };
  }
  return { user: data.user, token: data.token };
}
