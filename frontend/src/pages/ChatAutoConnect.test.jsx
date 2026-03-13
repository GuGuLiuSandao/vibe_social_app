import { render, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import Chat from "./Chat";

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
  };
});

describe("Chat websocket auto connect", () => {
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
    globalThis.localStorage.setItem("token:1", "x.eyJleHAiOjQxMDI0NDQ4MDB9.y");
    globalThis.WebSocket = vi.fn(() => ({
      onopen: null,
      onerror: null,
      onclose: null,
    }));
  });

  it("connects automatically when user is ready", async () => {
    render(
      <MemoryRouter initialEntries={["/chat?uid=1"]}>
        <Routes>
          <Route path="/chat" element={<Chat />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(globalThis.WebSocket).toHaveBeenCalled();
    });
  });
});
