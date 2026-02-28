import { parseUid } from "./uid";

export function getToken(uid) {
  return localStorage.getItem(`token:${uid}`) || "";
}

export function setToken(uid, token) {
  localStorage.setItem(`token:${uid}`, token);
}

export function setUser(uid, user) {
  localStorage.setItem(`user:${uid}`, JSON.stringify(user));
}

export function getUser(uid) {
  const raw = localStorage.getItem(`user:${uid}`);
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

export function setLastUid(uid) {
  localStorage.setItem("last_uid", String(uid));
}

export function getLastUid() {
  const value = localStorage.getItem("last_uid");
  if (!value) {
    return null;
  }
  return parseUid(value);
}
