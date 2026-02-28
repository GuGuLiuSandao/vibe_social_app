import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import Login from "./Login";

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    registerWithProtobuf: vi.fn(async () => ({
      token: "reg-token",
      user: { id: 10000001n, username: "neo", email: "neo@example.com" },
      message: "ok",
    })),
  };
});

describe("Login page", () => {
  it("renders whitelist entry", () => {
    render(
      <MemoryRouter>
        <Login />
      </MemoryRouter>
    );
    expect(screen.getByText("白名单直达")).toBeInTheDocument();
  });

  it("submits registration from modal", async () => {
    render(
      <MemoryRouter>
        <Login />
      </MemoryRouter>
    );

    fireEvent.click(screen.getAllByRole("button", { name: "注册账号" })[0]);
    fireEvent.change(screen.getByLabelText("用户名"), {
      target: { value: "neo" },
    });
    fireEvent.change(screen.getByLabelText("邮箱"), {
      target: { value: "neo@example.com" },
    });
    fireEvent.change(screen.getByLabelText("密码"), {
      target: { value: "secret12" },
    });

    fireEvent.click(screen.getByRole("button", { name: "创建账号" }));

    const { registerWithProtobuf } = await import("../lib/api");
    await waitFor(() =>
      expect(registerWithProtobuf).toHaveBeenCalledWith(
        "neo",
        "neo@example.com",
        "secret12"
      )
    );
  });
});
