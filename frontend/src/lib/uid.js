export const WHITELIST_MIN = 10000000;
export const WHITELIST_MAX = 20000000;

export function parseUid(value) {
  if (typeof value === 'bigint') {
    return value.toString();
  }
  const str = String(value).trim();
  if (!/^\d+$/.test(str)) {
    return null;
  }
  const num = BigInt(str);
  if (num <= 0n) {
    return null;
  }
  return str;
}

export function isWhitelistUid(uid) {
  try {
    const num = Number(uid);
    return Number.isInteger(num) && num >= WHITELIST_MIN && num <= WHITELIST_MAX;
  } catch {
    return false;
  }
}
