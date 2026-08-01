import { describe, expect, it } from "vitest";

import { isWhitelistUid, parseUid } from "./uid.js";

describe("DLQ_TC_007_CLIENT_001 parseUid", () => {
  it.each([
    [10000000, "10000000"],
    [10000000n, "10000000"],
    [" 0010000000 ", "0010000000"],
  ])("parses %s without changing digit text", (value, expected) => {
    expect(parseUid(value)).toBe(expected);
  });

  it.each(["", " ", "abc", "-1", "1.2", 0, 0n])("rejects %s", (value) => {
    expect(parseUid(value)).toBeNull();
  });
});

describe("DLQ_TC_008_CLIENT_001 isWhitelistUid", () => {
  it.each([
    [9999999, false],
    [10000000, true],
    [20000000, true],
    [20000001, false],
    ["10000000", true],
    ["not-a-number", false],
  ])("classifies %s", (value, expected) => {
    expect(isWhitelistUid(value)).toBe(expected);
  });

  it("rejects values that cannot be converted to Number", () => {
    expect(isWhitelistUid(Symbol("uid"))).toBe(false);
  });
});
