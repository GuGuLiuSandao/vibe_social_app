import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import {
  ChatPayloadSchema,
  GetMyGroupInvitationsResponseSchema,
  GroupInvitationSchema,
} from "../proto/chat/chat_pb";
import { WsMessageSchema, WsMessageType } from "../proto/ws_pb";
import { decodeWsMessage, encodeWsMessage } from "../lib/ws";
import Chat from "./Chat";

const routerMocks = vi.hoisted(() => ({ navigate: vi.fn() }));

vi.mock("react-router-dom", async (importOriginal) => ({
  ...(await importOriginal()),
  useNavigate: () => routerMocks.navigate,
}));

class FakeWebSocket {
  static OPEN = 1;
  static instances = [];

  constructor(url) {
    this.url = url;
    this.readyState = FakeWebSocket.OPEN;
    this.send = vi.fn();
    this.close = vi.fn();
    FakeWebSocket.instances.push(this);
  }
}

function renderChat(path = "/chat?uid=1") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Chat />
    </MemoryRouter>,
  );
}

async function renderAuthenticatedChat() {
  localStorage.setItem("token:1", "x.eyJleHAiOjQxMDI0NDQ4MDB9.y");
  localStorage.setItem("user:1", JSON.stringify({ id: "1", username: "tester", nickname: "测试用户" }));
  renderChat();
  await waitFor(() => expect(screen.getByText("Conversations")).toBeInTheDocument());
  await waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
  return FakeWebSocket.instances[0];
}

describe("DLQ_TC_038 Chat control wiring", () => {
  beforeEach(() => {
    const store = new Map();
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      value: {
        getItem: (key) => store.get(key) ?? null,
        setItem: (key, value) => store.set(key, String(value)),
        removeItem: (key) => store.delete(key),
        clear: () => store.clear(),
      },
    });
    routerMocks.navigate.mockReset();
    FakeWebSocket.instances = [];
    globalThis.WebSocket = FakeWebSocket;
  });

  it("returns an authentication error to login with replacement navigation", async () => {
    renderChat("/chat");
    const button = await screen.findByRole("button", { name: "返回登录页" });
    fireEvent.click(button);
    expect(routerMocks.navigate).toHaveBeenCalledWith("/login", { replace: true });
  });

  it("opens, wires, and closes the follow and profile dialogs", async () => {
    const socket = await renderAuthenticatedChat();
    fireEvent.click(screen.getByRole("button", { name: "关系" }));
    fireEvent.click(screen.getByRole("button", { name: "+ 关注" }));

    const cancelFollow = screen.getByRole("button", { name: "取消" });
    const confirmFollow = screen.getByRole("button", { name: "确认关注" });
    expect(cancelFollow.className.endsWith("h-10 rounded-lg border border-border bg-secondary px-4 text-sm text-secondary-foreground shadow-none hover:bg-secondary")).toBe(true);
    expect(confirmFollow.className.endsWith("h-10 rounded-lg px-4 text-sm font-semibold shadow-none")).toBe(true);
    fireEvent.change(screen.getByPlaceholderText("例如 20000002"), { target: { value: "2" } });
    fireEvent.click(confirmFollow);
    const followMessage = decodeWsMessage(socket.send.mock.calls.at(-1)[0]);
    expect(followMessage.payload.value.payload.case).toBe("follow");
    expect(followMessage.payload.value.payload.value.targetUid).toBe(2n);
    fireEvent.click(cancelFollow);
    expect(screen.queryByText("关注用户")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTitle("User: tester"));
    expect(screen.getByText("编辑资料")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "取消" }));
    expect(screen.queryByText("编辑资料")).not.toBeInTheDocument();
  });

  it("sends the selected accept and reject decisions for a group invitation", async () => {
    const socket = await renderAuthenticatedChat();
    const invitation = create(GroupInvitationSchema, {
      id: 7n,
      conversationId: 9n,
      inviterId: 2n,
      inviterNickname: "邀请人",
      groupName: "测试群",
    });
    const response = create(GetMyGroupInvitationsResponseSchema, { invitations: [invitation] });
    const payload = create(ChatPayloadSchema, {
      payload: { case: "getMyGroupInvitationsResponse", value: response },
    });
    const message = create(WsMessageSchema, {
      requestId: 99n,
      type: WsMessageType.WS_TYPE_CHAT_GET_MY_GROUP_INVITATIONS_RESPONSE,
      payload: { case: "chat", value: payload },
    });

    socket.onmessage({ data: encodeWsMessage(message) });
    await screen.findByText("测试群");
    fireEvent.click(screen.getByRole("button", { name: "接受" }));
    fireEvent.click(screen.getByRole("button", { name: "拒绝" }));

    const decisions = socket.send.mock.calls.slice(-2).map(([data]) => {
      const sent = decodeWsMessage(data);
      return sent.payload.value.payload.value;
    });
    expect(decisions.map((decision) => decision.invitationId)).toEqual([7n, 7n]);
    expect(decisions.map((decision) => decision.accept)).toEqual([true, false]);
  });
});
