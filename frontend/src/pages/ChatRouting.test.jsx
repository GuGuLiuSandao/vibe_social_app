import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import Chat from "./Chat";
import Login from "./Login";
import { setLastUid } from "../lib/storage";

const LocationDisplay = () => {
  const location = useLocation();
  return <div>{location.pathname}{location.search}</div>;
};

describe("Chat routing", () => {
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
    globalThis.localStorage.clear();
    globalThis.WebSocket = vi.fn(() => ({
      onopen: null,
      onerror: null,
      onclose: null,
      onmessage: null,
      close: vi.fn(),
      send: vi.fn(),
      readyState: 1,
    }));
  });

  it("redirects to last uid when missing in url", async () => {
    setLastUid(10000000);
    globalThis.localStorage.setItem("token:10000000", "test-token");
    render(
      <MemoryRouter initialEntries={["/chat"]}>
        <LocationDisplay />
        <Routes>
          <Route path="/chat" element={<Chat />} />
          <Route path="/login" element={<Login />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() =>
      expect(screen.getByText("/chat?uid=10000000")).toBeInTheDocument()
    );
  });

  it("redirects to login when uid has no token", async () => {
    render(
      <MemoryRouter initialEntries={["/chat?uid=1"]}>
        <LocationDisplay />
        <Routes>
          <Route path="/chat" element={<Chat />} />
          <Route path="/login" element={<Login />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => expect(screen.getAllByText("/login").length).toBeGreaterThan(0));
  });
});
