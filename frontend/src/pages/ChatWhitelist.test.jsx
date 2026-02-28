import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import Chat from "./Chat";

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    loginWithUid: vi.fn(async () => ({
      token: "wl-token",
      user: { id: 20000000, username: "whitelist_20000000", email: "whitelist@local" },
    })),
  };
});

describe("Chat whitelist behavior", () => {
  beforeEach(() => {
    const store = new Map();
    globalThis.localStorage = {
      getItem: (key) => (store.has(key) ? store.get(key) : null),
      setItem: (key, value) => {
        store.set(key, String(value));
      },
      removeItem: (key) => {
        store.delete(key);
      },
      clear: () => {
        store.clear();
      },
    };
    globalThis.WebSocket = vi.fn(() => ({
      onopen: null,
      onerror: null,
      onclose: null,
      close: vi.fn(),
    }));
  });

  it("auto logins whitelist uid and renders user", async () => {
    render(
      <MemoryRouter initialEntries={["/chat?uid=20000000"]}>
        <Routes>
          <Route path="/chat" element={<Chat />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      const avatar = screen.getByTitle(/User: whitelist_20000000/);
      expect(avatar).toBeInTheDocument();
    });
  });
});
