import { describe, expect, it } from "vitest";
import { isWhitelistUid, parseUid } from "./uid";

describe("uid helpers", () => {
  it("marks whitelist uid as true", () => {
    expect(isWhitelistUid(10000001)).toBe(true);
    expect(isWhitelistUid(20000000)).toBe(true);
  });

  it("marks non-whitelist uid as false", () => {
    expect(isWhitelistUid(20000001)).toBe(false);
    expect(isWhitelistUid(9999999)).toBe(false);
  });

  it("parses uid from input", () => {
    expect(parseUid("20000001")).toBe("20000001");
    expect(parseUid("0")).toBe(null);
    expect(parseUid("abc")).toBe(null);
  });
});
