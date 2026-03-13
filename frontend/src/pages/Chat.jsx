import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { create } from "@bufbuild/protobuf";
import ThemeSwitcher from "../components/ThemeSwitcher";
import { useThemeMode } from "../lib/useThemeMode";
import { Button, Input } from "../lib/vercel-ui";
import { loginWithUid } from "../lib/api";
import {
  getLastUid,
  getToken,
  getUser,
  setLastUid,
  setToken,
  setUser,
} from "../lib/storage";
import { isWhitelistUid, parseUid } from "../lib/uid";
import {
  buildAccountPing,
  buildAuthRequest,
  buildBlockRequest,
  buildGetBlockedRequest,
  buildFollowRequest,
  buildGetFollowersRequest,
  buildGetFollowingRequest,
  buildUnblockRequest,
  buildUpdateProfileRequest,
  buildUnfollowRequest,
  buildUploadAvatarRequest,
  buildWsUrl,
  decodeWsMessage,
  encodeWsMessage,
} from "../lib/ws";
import { WsMessageSchema, WsMessageType } from "../proto/ws_pb";
import { ErrorCode } from "../proto/common/error_code_pb";
import {
  ChatPayloadSchema,
  ConversationType,
  CreateConversationRequestSchema,
  GetConversationListRequestSchema,
  GetMessageListRequestSchema,
  GetTopicRoomListRequestSchema,
  GetTopicRoomMembersRequestSchema,
  JoinTopicRoomRequestSchema,
  LeaveTopicRoomRequestSchema,
  MarkAsReadRequestSchema,
  MessageType,
  SendTopicRoomMessageRequestSchema,
  SendMessageRequestSchema,
} from "../proto/chat/chat_pb";

const CONVERSATION_TYPE_PRIVATE =
  ConversationType.PRIVATE ?? ConversationType.CONVERSATION_TYPE_PRIVATE ?? 1;
const CONVERSATION_TYPE_GROUP =
  ConversationType.GROUP ?? ConversationType.CONVERSATION_TYPE_GROUP ?? 2;
const MESSAGE_TYPE_TEXT = MessageType.TEXT ?? MessageType.MESSAGE_TYPE_TEXT ?? 1;

function toIdString(value) {
  if (value === null || value === undefined) return "";
  return String(value);
}

function toIdBigInt(value) {
  return BigInt(String(value));
}

function formatTime(timestamp) {
  if (!timestamp) return "";
  if (timestamp.seconds !== undefined) {
    const date = new Date(Number(timestamp.seconds) * 1000 + Number(timestamp.nanos || 0) / 1e6);
    if (Number.isNaN(date.getTime())) return "";
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }
  const date = new Date(Number(timestamp));
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function getAvatarColor(id) {
  const colors = ["#5865f2", "#57f287", "#eb459e", "#fee75c", "#3ba55d", "#ed4245"];
  try {
    const idx = Number(BigInt(toIdString(id)) % BigInt(colors.length));
    return colors[idx];
  } catch {
    return colors[0];
  }
}

function getInitials(name) {
  if (!name) return "?";
  return name.slice(0, 2).toUpperCase();
}

function isTokenValid(token) {
  if (!token) return false;
  const segments = token.split(".");
  if (segments.length < 2) return false;

  try {
    const payloadPart = segments[1].replace(/-/g, "+").replace(/_/g, "/");
    const padded = payloadPart.padEnd(Math.ceil(payloadPart.length / 4) * 4, "=");
    const decoded = atob(padded);
    const payload = JSON.parse(decoded);
    if (typeof payload?.exp !== "number") return false;
    return payload.exp * 1000 > Date.now() + 5000;
  } catch {
    return false;
  }
}

function DiscordButton({ className = "", ...props }) {
  return (
    <Button
      variant="black"
      className={`!h-10 !rounded-md !border-[#5865f2] !bg-[#5865f2] !px-4 !text-white hover:!bg-[#4752c4] ${className}`}
      {...props}
    />
  );
}

function DiscordSecondaryButton({ className = "", ...props }) {
  return (
    <Button
      variant="secondary"
      className={`!h-10 !rounded-md !border-[#4f545c] !bg-[#2b2d31] !px-4 !text-slate-100 hover:!bg-[#32353b] ${className}`}
      {...props}
    />
  );
}

function DiscordInput(props) {
  return (
    <Input
      {...props}
      className={`!h-10 !rounded-md !border-[#3f4248] !bg-[#1e1f22] !text-slate-100 !placeholder:text-slate-500 focus:!border-[#5865f2] ${props.className || ""}`}
    />
  );
}

export default function Chat() {
  const requestCounterRef = useRef(1n);
  const pendingRequestRef = useRef(new Map());
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [theme, setTheme] = useThemeMode();
  const rawUid = searchParams.get("uid");
  const uid = useMemo(() => parseUid(rawUid), [rawUid]);

  const [user, setUserState] = useState(null);
  const [authStatus, setAuthStatus] = useState("loading");
  const [authError, setAuthError] = useState("");

  const wsRef = useRef(null);
  const [wsState, setWsState] = useState("disconnected");
  const [requestError, setRequestError] = useState("");

  const [conversations, setConversations] = useState([]);
  const [messages, setMessages] = useState({});
  const [activeConvId, setActiveConvId] = useState("");
  const activeConvIdRef = useRef(activeConvId);
  const [topicRooms, setTopicRooms] = useState([]);
  const [topicMessages, setTopicMessages] = useState({});
  const [activeTopicRoomId, setActiveTopicRoomId] = useState("");
  const activeTopicRoomIdRef = useRef(activeTopicRoomId);
  const [topicMembers, setTopicMembers] = useState([]);
  const messagesEndRef = useRef(null);

  const [activeTab, setActiveTab] = useState("messages");
  const [relationTab, setRelationTab] = useState("following");
  const [followingList, setFollowingList] = useState([]);
  const [followersList, setFollowersList] = useState([]);
  const [blockedList, setBlockedList] = useState([]);

  const [inputText, setInputText] = useState("");

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createGroupName, setCreateGroupName] = useState("");
  const [createGroupAvatar, setCreateGroupAvatar] = useState("");
  const [createGroupMemberIds, setCreateGroupMemberIds] = useState([]);

  const [followModalOpen, setFollowModalOpen] = useState(false);
  const [followTargetId, setFollowTargetId] = useState("");

  const [profileModalOpen, setProfileModalOpen] = useState(false);
  const [profileForm, setProfileForm] = useState({ nickname: "", avatar: "", bio: "" });
  const fileInputRef = useRef(null);

  const friendCandidates = useMemo(() => {
    const followingMap = new Map();
    const blockedSet = new Set(blockedList.map((item) => toIdString(item.user?.id)).filter(Boolean));
    followingList.forEach((item) => {
      const memberId = toIdString(item.user?.id);
      if (!memberId) return;
      if (blockedSet.has(memberId)) return;
      followingMap.set(memberId, item.user || {});
    });

    const friends = [];
    followersList.forEach((item) => {
      const memberId = toIdString(item.user?.id);
      if (!memberId || blockedSet.has(memberId) || !followingMap.has(memberId)) return;
      const followingUser = followingMap.get(memberId) || {};
      const followerUser = item.user || {};
      friends.push({
        id: memberId,
        nickname: followerUser.nickname || followingUser.nickname || "",
        username: followerUser.username || followingUser.username || "",
        avatar: followerUser.avatar || followingUser.avatar || "",
      });
    });

    friends.sort((a, b) => {
      const left = a.nickname || a.username || a.id;
      const right = b.nickname || b.username || b.id;
      return left.localeCompare(right);
    });
    return friends;
  }, [blockedList, followersList, followingList]);

  const blockedIdSet = useMemo(
    () => new Set(blockedList.map((item) => toIdString(item.user?.id)).filter(Boolean)),
    [blockedList],
  );

  const friendIdSet = useMemo(
    () => new Set(friendCandidates.map((item) => toIdString(item.id))),
    [friendCandidates],
  );

  const nextRequestId = () => {
    requestCounterRef.current += 1n;
    return BigInt(Date.now()) * 1000n + requestCounterRef.current;
  };

  const settlePendingRequest = (msg) => {
    const requestId = toIdString(msg?.requestId);
    if (!requestId) return false;

    const pending = pendingRequestRef.current.get(requestId);
    if (!pending) return false;

    if (msg.type !== pending.expectedType && msg.type !== WsMessageType.WS_TYPE_ERROR) {
      return false;
    }

    pendingRequestRef.current.delete(requestId);
    clearTimeout(pending.timeout);
    if (msg.payload?.case === "error") {
      pending.reject(new Error(msg.payload.value?.message || "请求失败"));
      return true;
    }
    pending.resolve(msg);
    return true;
  };

  const rejectAllPendingRequests = (reason = "连接已关闭") => {
    for (const pending of pendingRequestRef.current.values()) {
      clearTimeout(pending.timeout);
      pending.reject(new Error(reason));
    }
    pendingRequestRef.current.clear();
  };

  const sendWsRequest = (message, expectedType, socket = wsRef.current) => {
    const target = socket || wsRef.current;
    if (!target || target.readyState !== WebSocket.OPEN) {
      return Promise.reject(new Error("WebSocket 未连接"));
    }

    const requestId = toIdString(message.requestId);
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        pendingRequestRef.current.delete(requestId);
        reject(new Error("请求超时"));
      }, 10000);

      pendingRequestRef.current.set(requestId, {
        expectedType,
        resolve,
        reject,
        timeout,
      });

      try {
        target.send(encodeWsMessage(message));
      } catch (err) {
        clearTimeout(timeout);
        pendingRequestRef.current.delete(requestId);
        reject(err);
      }
    });
  };

  useEffect(() => {
    activeConvIdRef.current = activeConvId;
  }, [activeConvId]);

  useEffect(() => {
    activeTopicRoomIdRef.current = activeTopicRoomId;
  }, [activeTopicRoomId]);

  useEffect(() => {
    let cancelled = false;

    const loadUser = async () => {
      if (!uid) {
        const lastUid = getLastUid();
        if (lastUid) {
          navigate(`/chat?uid=${lastUid}`, { replace: true });
          return;
        }
        setAuthError("缺少 uid");
        setAuthStatus("error");
        return;
      }

      const whitelist = isWhitelistUid(uid);
      let token = getToken(uid);

      setLastUid(uid);
      const cachedUser = getUser(uid);
      if (cachedUser) {
        setUserState(cachedUser);
      }

      if (whitelist) {
        try {
          const data = await loginWithUid(uid);
          if (cancelled) return;
          setToken(uid, data.token);
          token = data.token;
          setUser(uid, data.user);
          setUserState(data.user);
        } catch (err) {
          if (!cancelled) {
            setAuthError(err.message || "登录失败");
            setAuthStatus("error");
          }
          return;
        }
      } else if (!isTokenValid(token)) {
        setToken(uid, "");
        setAuthError(token ? "登录已过期，请重新登录" : "需要登录");
        setAuthStatus("error");
        navigate("/login", { replace: true });
        return;
      }

      setAuthError("");
      setAuthStatus("authenticated");
    };

    loadUser();
    return () => {
      cancelled = true;
    };
  }, [uid, navigate]);

  const sendGetConversationList = (socket) => {
    const target = socket || wsRef.current;
    if (!target || target.readyState !== WebSocket.OPEN) return;

    const req = create(GetConversationListRequestSchema, { pageSize: 50 });
    const payload = create(ChatPayloadSchema, {
      payload: {
        case: "getConversationList",
        value: req,
      },
    });
    const wsMsg = create(WsMessageSchema, {
      requestId: BigInt(Date.now()),
      type: WsMessageType.WS_TYPE_CHAT_GET_CONVERSATION_LIST,
      timestamp: BigInt(Date.now()),
      payload: {
        case: "chat",
        value: payload,
      },
    });
    target.send(encodeWsMessage(wsMsg));
  };

  const sendGetTopicRoomList = (socket) => {
    const target = socket || wsRef.current;
    if (!target || target.readyState !== WebSocket.OPEN) return;

    const req = create(GetTopicRoomListRequestSchema, {});
    const payload = create(ChatPayloadSchema, {
      payload: {
        case: "getTopicRoomList",
        value: req,
      },
    });
    const wsMsg = create(WsMessageSchema, {
      requestId: BigInt(Date.now()),
      type: WsMessageType.WS_TYPE_CHAT_GET_TOPIC_ROOM_LIST,
      timestamp: BigInt(Date.now()),
      payload: {
        case: "chat",
        value: payload,
      },
    });
    target.send(encodeWsMessage(wsMsg));
  };

  const sendChatWsRequest = (type, payloadCase, payloadValue) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    const payload = create(ChatPayloadSchema, {
      payload: {
        case: payloadCase,
        value: payloadValue,
      },
    });
    const wsMsg = create(WsMessageSchema, {
      requestId: BigInt(Date.now()),
      type,
      timestamp: BigInt(Date.now()),
      payload: {
        case: "chat",
        value: payload,
      },
    });
    wsRef.current.send(encodeWsMessage(wsMsg));
  };

  const addTopicMessage = (message) => {
    const roomId = toIdString(message?.roomId);
    if (!roomId) return;
    setTopicMessages((prev) => {
      const current = prev[roomId] || [];
      if (current.some((item) => toIdString(item.id) === toIdString(message.id))) {
        return prev;
      }
      return {
        ...prev,
        [roomId]: [...current, message],
      };
    });
  };

  const leaveTopicRoom = (roomId) => {
    const targetRoomId = roomId || activeTopicRoomIdRef.current;
    if (!targetRoomId) return;
    const req = create(LeaveTopicRoomRequestSchema, {
      roomId: targetRoomId,
    });
    sendChatWsRequest(
      WsMessageType.WS_TYPE_CHAT_LEAVE_TOPIC_ROOM,
      "leaveTopicRoom",
      req,
    );
    setActiveTopicRoomId("");
    setTopicMembers([]);
  };

  const resetCreateConversationForm = () => {
    setCreateGroupName("");
    setCreateGroupAvatar("");
    setCreateGroupMemberIds([]);
  };

  const openCreateConversationModal = () => {
    setRequestError("");
    setCreateModalOpen(true);
    resetCreateConversationForm();
  };

  const closeCreateConversationModal = () => {
    setCreateModalOpen(false);
    resetCreateConversationForm();
  };

  const addMessage = (msg) => {
    const convId = toIdString(msg.conversationId);
    setMessages((prev) => {
      const current = prev[convId] || [];
      if (current.some((item) => toIdString(item.id) === toIdString(msg.id))) {
        return prev;
      }
      return { ...prev, [convId]: [...current, msg] };
    });
  };

  const updateConversationPreview = (msg) => {
    const convId = toIdString(msg.conversationId);
    setConversations((prev) => {
      const index = prev.findIndex((conv) => toIdString(conv.id) === convId);
      if (index === -1) return prev;
      const old = prev[index];
      const unread = toIdString(old.id) === activeConvIdRef.current ? 0 : Number(old.unreadCount || 0) + 1;
      const updated = {
        ...old,
        lastMessage: msg,
        updatedAt: msg.createdAt,
        unreadCount: unread,
      };
      const next = [...prev];
      next.splice(index, 1);
      return [updated, ...next];
    });
  };

  const handleWsMessage = (msg) => {
    if (!msg?.payload) return;

    if (msg.payload.case === "error") {
      const message = msg.payload.value?.message || "请求失败";
      setRequestError(message);
      return;
    }

    if (msg.payload.case === "chat") {
      const payload = msg.payload.value?.payload;
      if (!payload) return;

      switch (payload.case) {
        case "getConversationListResponse": {
          const list = payload.value?.conversations || [];
          setConversations(list);
          if (!activeConvIdRef.current && !activeTopicRoomIdRef.current && list.length > 0) {
            setActiveConvId(toIdString(list[0].id));
          }
          break;
        }
        case "getMessageListResponse": {
          const list = payload.value?.messages || [];
          if (list.length === 0) return;
          const convId = toIdString(list[0].conversationId);
          setMessages((prev) => ({ ...prev, [convId]: [...list].reverse() }));
          break;
        }
        case "sendMessageResponse": {
          const sent = payload.value?.message;
          if (sent) {
            addMessage(sent);
            updateConversationPreview(sent);
          }
          break;
        }
        case "messagePush": {
          const pushed = payload.value?.message;
          if (pushed) {
            addMessage(pushed);
            updateConversationPreview(pushed);
            if (activeConvIdRef.current === toIdString(pushed.conversationId)) {
              const readReq = create(MarkAsReadRequestSchema, {
                conversationId: pushed.conversationId,
                messageIds: [pushed.id],
                lastReadMessageId: pushed.id,
              });
              sendChatWsRequest(
                WsMessageType.WS_TYPE_CHAT_MARK_AS_READ,
                "markAsRead",
                readReq,
              );
            }
          }
          break;
        }
        case "conversationPush": {
          const conv = payload.value?.conversation;
          if (!conv) return;
          setConversations((prev) => [
            conv,
            ...prev.filter((item) => toIdString(item.id) !== toIdString(conv.id)),
          ]);
          break;
        }
        case "createConversationResponse": {
          const conv = payload.value?.conversation;
          if (!conv) return;
          setConversations((prev) => [
            conv,
            ...prev.filter((item) => toIdString(item.id) !== toIdString(conv.id)),
          ]);
          setActiveConvId(toIdString(conv.id));
          closeCreateConversationModal();
          setActiveTab("messages");
          break;
        }
        case "markAsReadResponse": {
          const responseConvId = toIdString(payload.value?.conversationId);
          const unreadCount = Number(payload.value?.unreadCount || 0);
          setConversations((prev) =>
            prev.map((conv) =>
              toIdString(conv.id) === responseConvId ? { ...conv, unreadCount } : conv,
            ),
          );
          break;
        }
        case "getTopicRoomListResponse": {
          const list = payload.value?.rooms || [];
          setTopicRooms(list);
          const joinedRoomId = payload.value?.joinedRoomId;
          if (joinedRoomId) {
            setActiveTopicRoomId(joinedRoomId);
            setActiveConvId("");
            const memberReq = create(GetTopicRoomMembersRequestSchema, {
              roomId: joinedRoomId,
            });
            sendChatWsRequest(
              WsMessageType.WS_TYPE_CHAT_GET_TOPIC_ROOM_MEMBERS,
              "getTopicRoomMembers",
              memberReq,
            );
          }
          break;
        }
        case "joinTopicRoomResponse": {
          const room = payload.value?.room;
          const roomId = toIdString(room?.id);
          if (!roomId) return;

          const recentMessages = payload.value?.recentMessages || [];
          const members = payload.value?.members || [];
          setActiveTopicRoomId(roomId);
          setActiveConvId("");
          setTopicMembers(members);
          setTopicMessages((prev) => ({ ...prev, [roomId]: recentMessages }));
          setTopicRooms((prev) => {
            const exists = prev.some((item) => toIdString(item.id) === roomId);
            if (!exists) {
              return [...prev, room];
            }
            return prev.map((item) =>
              toIdString(item.id) === roomId
                ? { ...item, onlineCount: room.onlineCount }
                : item,
            );
          });
          break;
        }
        case "leaveTopicRoomResponse": {
          const roomId = toIdString(payload.value?.roomId);
          if (roomId && roomId === activeTopicRoomIdRef.current) {
            setActiveTopicRoomId("");
            setTopicMembers([]);
          }
          break;
        }
        case "sendTopicRoomMessageResponse": {
          const topicMessage = payload.value?.message;
          if (topicMessage) {
            addTopicMessage(topicMessage);
          }
          break;
        }
        case "getTopicRoomMembersResponse": {
          const roomId = toIdString(payload.value?.roomId);
          if (roomId && roomId === activeTopicRoomIdRef.current) {
            setTopicMembers(payload.value?.members || []);
          }
          break;
        }
        case "topicRoomMessagePush": {
          const topicMessage = payload.value?.message;
          if (!topicMessage) return;
          addTopicMessage(topicMessage);
          break;
        }
        case "topicRoomMembersPush": {
          const roomId = toIdString(payload.value?.roomId);
          const onlineCount = Number(payload.value?.onlineCount || 0);
          setTopicRooms((prev) =>
            prev.map((room) =>
              toIdString(room.id) === roomId ? { ...room, onlineCount } : room,
            ),
          );
          if (roomId === activeTopicRoomIdRef.current) {
            setTopicMembers(payload.value?.members || []);
          }
          break;
        }
        default:
          break;
      }
      return;
    }

    if (msg.payload.case === "relation") {
      const payload = msg.payload.value?.payload;
      if (!payload) return;

      switch (payload.case) {
        case "getFollowingResponse":
          setFollowingList(payload.value?.followingList || []);
          break;
        case "getFollowersResponse":
          setFollowersList(payload.value?.followerList || []);
          break;
        case "getBlockedResponse":
          setBlockedList(payload.value?.blockedList || []);
          break;
        case "followResponse":
          if (payload.value?.errorCode === 1 || payload.value?.errorCode === 0) {
            wsRef.current?.send(encodeWsMessage(buildGetFollowingRequest()));
            wsRef.current?.send(encodeWsMessage(buildGetFollowersRequest()));
            wsRef.current?.send(encodeWsMessage(buildGetBlockedRequest()));
            setFollowModalOpen(false);
            setFollowTargetId("");
          } else {
            setRequestError(payload.value?.message || "关注失败");
          }
          break;
        case "unfollowResponse":
          if (payload.value?.errorCode === 1 || payload.value?.errorCode === 0) {
            wsRef.current?.send(encodeWsMessage(buildGetFollowingRequest()));
            wsRef.current?.send(encodeWsMessage(buildGetFollowersRequest()));
            wsRef.current?.send(encodeWsMessage(buildGetBlockedRequest()));
          } else {
            setRequestError(payload.value?.message || "取消关注失败");
          }
          break;
        case "blockResponse":
          if (payload.value?.errorCode === 1 || payload.value?.errorCode === 0) {
            wsRef.current?.send(encodeWsMessage(buildGetFollowingRequest()));
            wsRef.current?.send(encodeWsMessage(buildGetFollowersRequest()));
            wsRef.current?.send(encodeWsMessage(buildGetBlockedRequest()));
          } else {
            setRequestError(payload.value?.message || "拉黑失败");
          }
          break;
        case "unblockResponse":
          if (payload.value?.errorCode === 1 || payload.value?.errorCode === 0) {
            wsRef.current?.send(encodeWsMessage(buildGetBlockedRequest()));
          } else {
            setRequestError(payload.value?.message || "取消拉黑失败");
          }
          break;
        case "relationPush":
          wsRef.current?.send(encodeWsMessage(buildGetFollowingRequest()));
          wsRef.current?.send(encodeWsMessage(buildGetFollowersRequest()));
          wsRef.current?.send(encodeWsMessage(buildGetBlockedRequest()));
          break;
        default:
          break;
      }
    }
  };

  useEffect(() => {
    if (authStatus !== "authenticated" || !uid) return;
    const token = getToken(uid);
    if (!token) return;

    const url = buildWsUrl(uid, token);
    const socket = new WebSocket(url);
    socket.binaryType = "arraybuffer";
    wsRef.current = socket;
    setWsState("connecting");

    socket.onopen = async () => {
      setWsState("connected");
      setRequestError("");
      socket.send(encodeWsMessage(buildAccountPing()));
      sendGetConversationList(socket);
      sendGetTopicRoomList(socket);
      socket.send(encodeWsMessage(buildGetFollowingRequest()));
      socket.send(encodeWsMessage(buildGetFollowersRequest()));
      socket.send(encodeWsMessage(buildGetBlockedRequest()));

      try {
        const authMessage = buildAuthRequest(uid, nextRequestId());
        const response = await sendWsRequest(
          authMessage,
          WsMessageType.WS_TYPE_AUTH_RESPONSE,
          socket,
        );
        const payload =
          response.payload?.case === "account" ? response.payload.value?.payload : undefined;
        if (!payload || payload.case !== "authResponse") {
          throw new Error("鉴权响应异常");
        }
        if (
          payload.value?.errorCode !== ErrorCode.OK &&
          payload.value?.errorCode !== ErrorCode.UNSPECIFIED
        ) {
          throw new Error(payload.value?.message || "鉴权失败");
        }
        if (payload.value?.user) {
          const currentUser = {
            ...payload.value.user,
            id: toIdString(payload.value.user.id),
          };
          setUser(uid, currentUser);
          setUserState(currentUser);
        }
      } catch (err) {
        setAuthError(err.message || "鉴权失败");
        setAuthStatus("error");
        if (!isWhitelistUid(uid)) {
          navigate("/login", { replace: true });
        }
      }
    };

    socket.onmessage = (event) => {
      try {
        const wsMessage = decodeWsMessage(event.data);
        if (settlePendingRequest(wsMessage)) return;
        handleWsMessage(wsMessage);
      } catch {
        setRequestError("收到无法解析的消息");
      }
    };

    socket.onclose = () => {
      setWsState("disconnected");
      setActiveTopicRoomId("");
      setTopicMembers([]);
      rejectAllPendingRequests("连接已关闭");
    };

    socket.onerror = () => {
      setWsState("error");
    };

    return () => {
      rejectAllPendingRequests("连接已关闭");
      socket.close();
      wsRef.current = null;
    };
  }, [authStatus, uid, navigate]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, activeConvId]);

  const handleSelectConversation = (conv) => {
    if (activeTopicRoomIdRef.current) {
      leaveTopicRoom(activeTopicRoomIdRef.current);
    }

    const convId = toIdString(conv.id);
    setActiveTab("messages");
    setActiveConvId(convId);

    if (!messages[convId]) {
      const req = create(GetMessageListRequestSchema, {
        conversationId: toIdBigInt(convId),
        pageSize: 50,
      });
      sendChatWsRequest(WsMessageType.WS_TYPE_CHAT_GET_MESSAGE_LIST, "getMessageList", req);
    }

    if (Number(conv.unreadCount) > 0 && conv.lastMessage?.id) {
      const req = create(MarkAsReadRequestSchema, {
        conversationId: conv.id,
        messageIds: [conv.lastMessage.id],
        lastReadMessageId: conv.lastMessage.id,
      });
      sendChatWsRequest(WsMessageType.WS_TYPE_CHAT_MARK_AS_READ, "markAsRead", req);
      setConversations((prev) =>
        prev.map((item) =>
          toIdString(item.id) === convId ? { ...item, unreadCount: 0 } : item,
        ),
      );
    }
  };

  const handleSelectTopicRoom = (room) => {
    const roomId = toIdString(room?.id);
    if (!roomId) return;

    setActiveTab("messages");
    setActiveConvId("");

    if (activeTopicRoomIdRef.current === roomId) {
      const memberReq = create(GetTopicRoomMembersRequestSchema, {
        roomId,
      });
      sendChatWsRequest(
        WsMessageType.WS_TYPE_CHAT_GET_TOPIC_ROOM_MEMBERS,
        "getTopicRoomMembers",
        memberReq,
      );
      return;
    }

    const req = create(JoinTopicRoomRequestSchema, {
      roomId,
    });
    sendChatWsRequest(
      WsMessageType.WS_TYPE_CHAT_JOIN_TOPIC_ROOM,
      "joinTopicRoom",
      req,
    );
  };

  const handleSendMessage = (event) => {
    event.preventDefault();
    const trimmed = inputText.trim();
    if (!trimmed) return;

    if (activeTopicRoomId) {
      const req = create(SendTopicRoomMessageRequestSchema, {
        roomId: activeTopicRoomId,
        content: trimmed,
      });
      sendChatWsRequest(
        WsMessageType.WS_TYPE_CHAT_SEND_TOPIC_ROOM_MESSAGE,
        "sendTopicRoomMessage",
        req,
      );
      setInputText("");
      return;
    }

    if (!activeConvId) return;
    const req = create(SendMessageRequestSchema, {
      conversationId: toIdBigInt(activeConvId),
      content: trimmed,
      type: MESSAGE_TYPE_TEXT,
    });
    sendChatWsRequest(WsMessageType.WS_TYPE_CHAT_SEND_MESSAGE, "sendMessage", req);
    setInputText("");
  };

  const startPrivateConversation = (targetId) => {
    const normalized = parseUid(targetId);
    if (!normalized) {
      setRequestError("请输入有效的目标 UID");
      return;
    }
    setRequestError("");
    setActiveTab("messages");
    const req = create(CreateConversationRequestSchema, {
      type: CONVERSATION_TYPE_PRIVATE,
      participantIds: [toIdBigInt(normalized)],
    });
    sendChatWsRequest(
      WsMessageType.WS_TYPE_CHAT_CREATE_CONVERSATION,
      "createConversation",
      req,
    );
  };

  const handleCreateConversation = () => {
    const groupName = createGroupName.trim();
    if (!groupName) {
      setRequestError("请输入群聊名称");
      return;
    }

    const selfUid = toIdString(uid);
    const participantIDs = Array.from(new Set(createGroupMemberIds)).filter(
      (item) => item && item !== selfUid && friendIdSet.has(item),
    );

    if (participantIDs.length < 2) {
      setRequestError("请至少选择 2 位好友（不包含自己）");
      return;
    }

    const req = create(CreateConversationRequestSchema, {
      type: CONVERSATION_TYPE_GROUP,
      participantIds: participantIDs.map((id) => toIdBigInt(id)),
      name: groupName,
      avatar: createGroupAvatar.trim(),
    });
    sendChatWsRequest(
      WsMessageType.WS_TYPE_CHAT_CREATE_CONVERSATION,
      "createConversation",
      req,
    );
  };

  const handleFollowUser = (targetId = followTargetId) => {
    const normalized = parseUid(targetId);
    if (!normalized) {
      setRequestError("请输入有效的用户 UID");
      return;
    }
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(encodeWsMessage(buildFollowRequest(toIdBigInt(normalized))));
  };

  const handleUnfollowUser = (targetId) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(encodeWsMessage(buildUnfollowRequest(toIdBigInt(targetId))));
  };

  const handleBlockUser = (targetId) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(encodeWsMessage(buildBlockRequest(toIdBigInt(targetId))));
  };

  const handleUnblockUser = (targetId) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(encodeWsMessage(buildUnblockRequest(toIdBigInt(targetId))));
  };

  const refreshRelationList = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    if (relationTab === "following") {
      wsRef.current.send(encodeWsMessage(buildGetFollowingRequest()));
    } else if (relationTab === "blocked") {
      wsRef.current.send(encodeWsMessage(buildGetBlockedRequest()));
    } else {
      wsRef.current.send(encodeWsMessage(buildGetFollowersRequest()));
    }
  };

  const openProfileModal = () => {
    if (user) {
      setProfileForm({
        nickname: user.nickname || "",
        avatar: user.avatar || "",
        bio: user.bio || "",
      });
    }
    setProfileModalOpen(true);
  };

  const toggleCreateGroupMember = (memberId) => {
    setCreateGroupMemberIds((prev) =>
      prev.includes(memberId)
        ? prev.filter((item) => item !== memberId)
        : [...prev, memberId],
    );
  };

  const handleFileUpload = async (event) => {
    const file = event.target.files?.[0];
    if (!file) return;
    try {
      const data = new Uint8Array(await file.arrayBuffer());
      const response = await sendWsRequest(
        buildUploadAvatarRequest(file.name, data, nextRequestId()),
        WsMessageType.WS_TYPE_ACCOUNT_UPLOAD_AVATAR_RESPONSE,
      );
      const payload =
        response.payload?.case === "account" ? response.payload.value?.payload : undefined;
      if (!payload || payload.case !== "uploadAvatarResponse") {
        throw new Error("上传响应异常");
      }
      if (
        payload.value?.errorCode !== ErrorCode.OK &&
        payload.value?.errorCode !== ErrorCode.UNSPECIFIED
      ) {
        throw new Error(payload.value?.message || "上传失败");
      }
      setProfileForm((prev) => ({ ...prev, avatar: payload.value?.url || "" }));
    } catch (err) {
      setRequestError(err.message || "上传失败");
    }
  };

  const handleUpdateProfile = async () => {
    if (!uid) return;
    try {
      const response = await sendWsRequest(
        buildUpdateProfileRequest(profileForm, nextRequestId()),
        WsMessageType.WS_TYPE_ACCOUNT_UPDATE_PROFILE_RESPONSE,
      );
      const payload =
        response.payload?.case === "account" ? response.payload.value?.payload : undefined;
      if (!payload || payload.case !== "updateProfileResponse") {
        throw new Error("更新响应异常");
      }
      if (
        payload.value?.errorCode !== ErrorCode.OK &&
        payload.value?.errorCode !== ErrorCode.UNSPECIFIED
      ) {
        throw new Error(payload.value?.message || "更新失败");
      }

      const updated = payload.value?.user
        ? { ...payload.value.user, id: toIdString(payload.value.user.id) }
        : null;
      if (!updated) {
        throw new Error("用户信息为空");
      }

      setUser(uid, updated);
      setUserState(updated);
      setProfileModalOpen(false);
    } catch (err) {
      setRequestError(err.message || "更新失败");
    }
  };

  const handleLogout = () => {
    if (uid) {
      setToken(uid, "");
    }
    navigate("/login", { replace: true });
  };

  if (authStatus === "loading") {
    return (
      <div className="flex h-screen items-center justify-center bg-[#1e1f22] text-slate-300">
        正在连接聊天服务...
      </div>
    );
  }

  if (authStatus === "error") {
    return (
      <div className="flex h-screen items-center justify-center bg-[#1e1f22] p-4">
        <div className="discord-surface w-full max-w-lg rounded-xl p-6 text-center">
          <p className="text-sm text-[#ffb4bf]">Error: {authError}</p>
          <DiscordButton className="mt-4 w-full" onClick={() => navigate("/login", { replace: true })}>
            返回登录页
          </DiscordButton>
        </div>
      </div>
    );
  }

  const relationList =
    relationTab === "following"
      ? followingList
      : relationTab === "blocked"
        ? blockedList
        : followersList;
  const activeMessages = activeConvId ? messages[activeConvId] || [] : [];
  const activeConv = conversations.find((conv) => toIdString(conv.id) === activeConvId) || null;
  const activeTopicRoom = topicRooms.find((room) => toIdString(room.id) === activeTopicRoomId) || null;
  const activeTopicMessages = activeTopicRoomId ? topicMessages[activeTopicRoomId] || [] : [];
  const wsLabel = {
    connecting: "连接中",
    connected: "在线",
    disconnected: "离线",
    error: "异常",
  }[wsState] || "未知";
  const privateConversations = conversations.filter(
    (conv) => Number(conv.type) === Number(CONVERSATION_TYPE_PRIVATE),
  );
  const groupConversations = conversations.filter(
    (conv) => Number(conv.type) === Number(CONVERSATION_TYPE_GROUP),
  );
  const conversationGroups = [
    { id: "group", label: "群聊", items: groupConversations },
    { id: "private", label: "私聊", items: privateConversations },
  ].filter((group) => group.items.length > 0);
  const unreadTotal = conversations.reduce((count, conv) => count + Number(conv.unreadCount || 0), 0);

  const memberMap = new Map();
  const addMember = (id, name, avatar, badge) => {
    const memberId = toIdString(id);
    if (!memberId) return;
    if (memberMap.has(memberId)) return;
    memberMap.set(memberId, {
      id: memberId,
      name: name || `UID ${memberId}`,
      avatar: avatar || "",
      badge,
    });
  };
  addMember(user?.id, user?.nickname || user?.username, user?.avatar, "你");
  activeMessages.forEach((msg) =>
    addMember(msg.senderId, msg.sender?.nickname || msg.sender?.username, msg.sender?.avatar, "频道中"),
  );
  followingList.forEach((item) =>
    addMember(item.user?.id, item.user?.nickname || item.user?.username, item.user?.avatar, "关注"),
  );
  followersList.forEach((item) =>
    addMember(item.user?.id, item.user?.nickname || item.user?.username, item.user?.avatar, "粉丝"),
  );
  const defaultMemberPanelList = Array.from(memberMap.values()).slice(0, 16);
  const topicMemberPanelList = topicMembers.map((member) => ({
    id: toIdString(member.id),
    name: member.nickname || member.username || `UID ${toIdString(member.id)}`,
    avatar: member.avatar || "",
    badge: "在线",
  }));
  const memberPanelList = activeTopicRoomId ? topicMemberPanelList : defaultMemberPanelList;

  return (
    <div className="flex h-screen flex-col bg-[#1e1f22] text-slate-100 md:flex-row">
      <aside className="flex h-[72px] w-full items-center border-b discord-divider bg-[#191a1d] px-3 md:h-auto md:w-[72px] md:flex-col md:items-center md:border-b-0 md:border-r md:px-0 md:py-3">
        <button
          type="button"
          title={`User: ${user?.username || ""}`}
          onClick={openProfileModal}
          className="mr-2 flex h-12 w-12 items-center justify-center rounded-2xl border border-[#3d4047] bg-[#2b2d31] text-sm font-bold transition hover:rounded-xl hover:bg-[#5865f2] md:mb-3 md:mr-0"
          style={user?.avatar ? { backgroundColor: "transparent", overflow: "hidden" } : undefined}
        >
          {user?.avatar ? (
            <img src={user.avatar} alt="avatar" className="h-full w-full object-cover" />
          ) : (
            getInitials(user?.nickname || user?.username || "ME")
          )}
        </button>

        <button
          type="button"
          onClick={() => setActiveTab("messages")}
          className={`mr-2 h-11 w-11 rounded-2xl border border-[#3d4047] text-[11px] font-semibold transition md:mb-2 md:mr-0 md:h-12 md:w-12 ${
            activeTab === "messages"
              ? "bg-[#5865f2] text-white"
              : "bg-[#2b2d31] text-slate-300 hover:bg-[#3a3d43]"
          }`}
        >
          聊天
        </button>
        <button
          type="button"
          onClick={() => {
            if (activeTopicRoomIdRef.current) {
              leaveTopicRoom(activeTopicRoomIdRef.current);
            }
            setActiveTab("contacts");
          }}
          className={`h-11 w-11 rounded-2xl border border-[#3d4047] text-[11px] font-semibold transition md:mb-2 md:h-12 md:w-12 ${
            activeTab === "contacts"
              ? "bg-[#5865f2] text-white"
              : "bg-[#2b2d31] text-slate-300 hover:bg-[#3a3d43]"
          }`}
        >
          关系
        </button>

        <div className="ml-auto rounded-full border border-[#3d4047] bg-[#232428] px-2 py-0.5 text-[10px] text-slate-400 md:ml-0 md:mt-auto">
          {wsLabel}
        </div>
      </aside>

      <aside className="flex max-h-[42vh] w-full flex-col border-b discord-divider bg-[#2b2d31] md:max-h-none md:w-[320px] md:border-b-0 md:border-r">
        <div className="border-b discord-divider px-4 py-3">
          <div className="flex items-center justify-between gap-2">
            <h2 className="font-display text-sm font-bold uppercase tracking-wider text-slate-200">
              {activeTab === "messages" ? "Conversations" : "Relations"}
            </h2>
            {activeTab === "messages" ? (
              <button
                type="button"
                onClick={openCreateConversationModal}
                className="rounded-md border border-[#42454d] bg-[#35373c] px-2 py-1 text-xs text-slate-200 transition hover:bg-[#464952]"
              >
                + 群聊
              </button>
            ) : (
              <button
                type="button"
                onClick={() => setFollowModalOpen(true)}
                className="rounded-md border border-[#42454d] bg-[#35373c] px-2 py-1 text-xs text-slate-200 transition hover:bg-[#464952]"
              >
                + 关注
              </button>
            )}
          </div>
          <DiscordInput className="mt-3 !h-9" placeholder="搜索会话 / UID..." />
        </div>

        {activeTab === "messages" ? (
          <div className="flex-1 overflow-y-auto px-2 pb-2 pt-1">
            <div className="mb-2">
              <div className="flex items-center justify-between px-2 py-1">
                <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-500">
                  官方社区
                </p>
                <span className="text-[11px] text-slate-500">{topicRooms.length}</span>
              </div>
              {topicRooms.map((room) => {
                const roomId = toIdString(room.id);
                const selected = roomId === activeTopicRoomId;
                return (
                  <button
                    key={roomId}
                    type="button"
                    onClick={() => handleSelectTopicRoom(room)}
                    className={`mb-1 flex w-full items-center gap-3 rounded-md px-2 py-2 text-left transition ${
                      selected
                        ? "bg-[#404249] shadow-[inset_3px_0_0_0_#5865f2]"
                        : "hover:bg-[#35373c]"
                    }`}
                  >
                    <div
                      className="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-full bg-[#232428] text-xs font-bold text-white"
                    >
                      {room.icon ? (
                        <img src={room.icon} alt="icon" className="h-full w-full object-cover" />
                      ) : (
                        getInitials(room.name || roomId)
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-semibold text-slate-100">
                        {room.name || roomId}
                      </p>
                      <p className="truncate text-xs text-slate-400">
                        {room.description || "官方话题聊天室"}
                      </p>
                    </div>
                    <span className="rounded-full border border-[#41444d] bg-[#232428] px-2 py-0.5 text-[10px] text-slate-300">
                      {room.onlineCount || 0}
                    </span>
                  </button>
                );
              })}
            </div>

            {conversations.length === 0 ? (
              <div className="rounded-lg border border-[#3a3d43] bg-[#232428] p-3 text-sm text-slate-400">
                还没有会话，点击右上角创建私聊或群聊。
              </div>
            ) : null}
            {conversationGroups.map((group) => (
              <div key={group.id} className="mb-2">
                <div className="flex items-center justify-between px-2 py-1">
                  <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-500">
                    {group.label}
                  </p>
                  <span className="text-[11px] text-slate-500">{group.items.length}</span>
                </div>
                {group.items.map((conv) => {
                  const convId = toIdString(conv.id);
                  const selected = convId === activeConvId;
                  const isGroup = Number(conv.type) === Number(CONVERSATION_TYPE_GROUP);
                  return (
                    <button
                      key={convId}
                      type="button"
                      onClick={() => handleSelectConversation(conv)}
                      className={`mb-1 flex w-full items-center gap-3 rounded-md px-2 py-2 text-left transition ${
                        selected
                          ? "bg-[#404249] shadow-[inset_3px_0_0_0_#5865f2]"
                          : "hover:bg-[#35373c]"
                      }`}
                    >
                      <div
                        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                        style={{
                          backgroundColor: conv.avatar ? "transparent" : getAvatarColor(conv.id),
                          overflow: "hidden",
                        }}
                      >
                        {conv.avatar ? (
                          <img src={conv.avatar} alt="avatar" className="h-full w-full object-cover" />
                        ) : (
                          getInitials(conv.name || conv.id)
                        )}
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex min-w-0 items-center gap-1.5">
                            <p className="truncate text-sm font-semibold text-slate-100">
                              {conv.name || `会话 ${convId}`}
                            </p>
                            {isGroup ? (
                              <span className="shrink-0 rounded border border-[#4f545c] bg-[#232428] px-1.5 py-0.5 text-[10px] text-slate-400">
                                群
                              </span>
                            ) : null}
                          </div>
                          <span className="text-[11px] text-slate-400">{formatTime(conv.updatedAt)}</span>
                        </div>
                        <p className="truncate text-xs text-slate-400">
                          {conv.lastMessage?.content || "还没有消息"}
                        </p>
                      </div>
                      {Number(conv.unreadCount || 0) > 0 ? (
                        <span className="ml-auto rounded-full bg-[#ed4245] px-2 py-0.5 text-[10px] font-bold text-white">
                          {conv.unreadCount}
                        </span>
                      ) : null}
                    </button>
                  );
                })}
              </div>
            ))}
          </div>
        ) : (
          <div className="flex flex-1 flex-col">
                <div className="grid grid-cols-3 gap-2 p-2">
                  <button
                type="button"
                onClick={() => setRelationTab("following")}
                className={`rounded-md px-2 py-2 text-xs font-semibold transition ${
                  relationTab === "following"
                    ? "bg-[#5865f2] text-white"
                    : "bg-[#35373c] text-slate-300 hover:bg-[#404249]"
                }`}
              >
                Following
              </button>
              <button
                type="button"
                onClick={() => setRelationTab("followers")}
                className={`rounded-md px-2 py-2 text-xs font-semibold transition ${
                  relationTab === "followers"
                    ? "bg-[#5865f2] text-white"
                    : "bg-[#35373c] text-slate-300 hover:bg-[#404249]"
                }`}
                  >
                    Followers
                  </button>
                  <button
                    type="button"
                    onClick={() => setRelationTab("blocked")}
                    className={`rounded-md px-2 py-2 text-xs font-semibold transition ${
                      relationTab === "blocked"
                        ? "bg-[#5865f2] text-white"
                        : "bg-[#35373c] text-slate-300 hover:bg-[#404249]"
                    }`}
                  >
                    Blocked
                  </button>
                </div>
            <div className="flex-1 overflow-y-auto p-2">
              {relationList.length === 0 ? (
                <div className="rounded-lg border border-[#3a3d43] bg-[#232428] p-3 text-sm text-slate-400">
                  暂无数据
                </div>
              ) : null}
              {relationList.map((relation) => {
                const rid = toIdString(relation.user?.id);
                const isFollowing = followingList.some(
                  (item) => toIdString(item.user?.id) === rid,
                );
                const isFollower = followersList.some(
                  (item) => toIdString(item.user?.id) === rid,
                );
                const isFriend = isFollowing && isFollower;
                const isBlocked = blockedIdSet.has(rid);
                return (
                  <div
                    key={rid}
                    className="mb-2 rounded-lg border border-[#3a3d43] bg-[#232428] p-3"
                  >
                    <div className="flex items-center gap-3">
                      <div
                        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                        style={{
                          backgroundColor: relation.user?.avatar ? "transparent" : getAvatarColor(rid),
                          overflow: "hidden",
                        }}
                      >
                        {relation.user?.avatar ? (
                          <img
                            src={relation.user.avatar}
                            alt="avatar"
                            className="h-full w-full object-cover"
                          />
                        ) : (
                          getInitials(relation.user?.nickname || relation.user?.username)
                        )}
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-semibold text-slate-100">
                          {relation.user?.nickname || relation.user?.username}
                        </p>
                        <p className="truncate text-xs text-slate-400">UID: {rid}</p>
                      </div>
                    </div>
                    <div className="mt-2 flex gap-2">
                      <DiscordSecondaryButton
                        className={`!h-8 !text-xs ${
                          isBlocked
                            ? "!cursor-not-allowed !border-[#3d4047] !bg-[#232428] !text-slate-500 hover:!bg-[#232428]"
                            : ""
                        }`}
                        disabled={isBlocked}
                        title={isBlocked ? "已拉黑，无法发起私聊" : "发起私聊"}
                        onClick={() => startPrivateConversation(rid)}
                      >
                        发消息
                      </DiscordSecondaryButton>
                      <DiscordSecondaryButton
                        className={`!h-8 !text-xs ${
                          isFriend
                            && !isBlocked
                            ? ""
                            : "!cursor-not-allowed !border-[#3d4047] !bg-[#232428] !text-slate-500 hover:!bg-[#232428]"
                        }`}
                        disabled={!isFriend || isBlocked}
                        title={
                          isBlocked ? "已拉黑，无法邀请入群" : isFriend ? "创建好友群聊" : "仅可邀请互关好友"
                        }
                        onClick={openCreateConversationModal}
                      >
                        拉群
                      </DiscordSecondaryButton>
                      {relationTab === "blocked" ? (
                        <DiscordSecondaryButton
                          className="!h-8 !text-xs !border-[#305f4a] !bg-[#1f3b30] !text-[#b7f7cc] hover:!bg-[#28503f]"
                          onClick={() => handleUnblockUser(rid)}
                        >
                          取消拉黑
                        </DiscordSecondaryButton>
                      ) : (
                        <DiscordSecondaryButton
                          className="!h-8 !text-xs !border-[#5f2a33] !bg-[#3b1f24] !text-[#ffb4bf] hover:!bg-[#4a252d]"
                          onClick={() => handleBlockUser(rid)}
                        >
                          拉黑
                        </DiscordSecondaryButton>
                      )}
                      {relationTab === "following" ? (
                        <DiscordSecondaryButton
                          className="!h-8 !text-xs !border-[#5f2a33] !bg-[#3b1f24] !text-[#ffb4bf] hover:!bg-[#4a252d]"
                          onClick={() => handleUnfollowUser(rid)}
                        >
                          取消关注
                        </DiscordSecondaryButton>
                      ) : !isFollowing && relationTab !== "blocked" ? (
                        <DiscordButton
                          className="!h-8 !text-xs"
                          onClick={() => handleFollowUser(rid)}
                        >
                          回关
                        </DiscordButton>
                      ) : null}
                    </div>
                  </div>
                );
              })}
            </div>
            <div className="border-t discord-divider p-2">
              <DiscordSecondaryButton className="w-full !h-9 !text-xs" onClick={refreshRelationList}>
                刷新列表
              </DiscordSecondaryButton>
            </div>
          </div>
        )}
      </aside>

      <main className="flex min-h-0 min-w-0 flex-1 bg-[#313338]">
        <div className="flex min-w-0 flex-1 flex-col">
          <header className="flex h-14 items-center justify-between border-b discord-divider px-4">
            <div className="flex min-w-0 items-center gap-2">
              <div className="flex h-7 w-7 items-center justify-center rounded-md bg-[#232428] text-sm text-slate-300">
                #
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-bold text-slate-100">
                  {activeTab === "messages"
                    ? activeTopicRoom
                      ? activeTopicRoom.name || "官方聊天室"
                      : activeConv?.name || "选择一个会话开始聊天"
                    : relationTab === "following"
                      ? "My Following"
                      : "My Followers"}
                </p>
                <p className="truncate text-xs text-slate-400">
                  {activeTab === "messages"
                      ? activeTopicRoomId
                      ? `官方话题 · 在线 ${activeTopicRoom?.onlineCount || topicMembers.length}`
                      : activeConvId
                      ? `${Number(activeConv?.type) === Number(CONVERSATION_TYPE_GROUP) ? "群聊" : "私聊"} · 频道 ID: ${activeConvId}`
                      : "尚未选择会话"
                    : `${relationList.length} users`}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <ThemeSwitcher theme={theme} onChange={setTheme} compact />
              <span className="hidden rounded-full border border-[#41444d] bg-[#232428] px-2 py-1 text-[10px] font-semibold text-slate-300 md:inline-flex">
                未读 {unreadTotal}
              </span>
              <span
                className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ${
                  wsState === "connected"
                    ? "bg-[#1f6f43] text-[#b7f7cc]"
                    : "bg-[#3a3d43] text-slate-300"
                }`}
              >
                WS: {wsLabel}
              </span>
            </div>
          </header>

          {requestError ? (
            <div className="flex items-center justify-between gap-3 border-b border-[#5f2a33] bg-[#3b1f24] px-4 py-2">
              <p className="min-w-0 text-xs text-[#ffb4bf]">{requestError}</p>
              <button
                type="button"
                onClick={() => setRequestError("")}
                aria-label="关闭提示"
                className="shrink-0 rounded border border-[#7a3b47] px-1.5 py-0.5 text-[11px] font-semibold text-[#ffb4bf] transition hover:bg-[#4a252d]"
              >
                ×
              </button>
            </div>
          ) : null}

          {activeTab === "contacts" ? (
            <div className="flex flex-1 items-center justify-center px-4 text-sm text-slate-400">
              左侧可查看并管理关系链路。
            </div>
          ) : activeTopicRoomId ? (
            <>
              <div className="flex-1 overflow-y-auto px-4 py-4">
                {activeTopicMessages.length === 0 ? (
                  <div className="rounded-xl border border-[#3a3d43] bg-[#2b2d31] p-4 text-sm text-slate-400">
                    还没有消息，来发第一条话题消息吧。
                  </div>
                ) : null}
                {activeTopicMessages.map((msg) => {
                  const self = toIdString(msg.senderId) === toIdString(user?.id);
                  const senderName =
                    msg.sender?.nickname ||
                    msg.sender?.username ||
                    (self ? user?.nickname || user?.username : "Unknown");
                  const senderAvatar = msg.sender?.avatar || (self ? user?.avatar : "");
                  return (
                    <div
                      key={toIdString(msg.id)}
                      className={`chat-message-row group relative mb-2 flex gap-3 px-2 py-1 ${
                        self ? "flex-row-reverse" : ""
                      }`}
                    >
                      <div
                        className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                        style={{
                          backgroundColor: senderAvatar ? "transparent" : getAvatarColor(msg.senderId),
                          overflow: "hidden",
                        }}
                      >
                        {senderAvatar ? (
                          <img src={senderAvatar} alt="avatar" className="h-full w-full object-cover" />
                        ) : (
                          getInitials(senderName)
                        )}
                      </div>
                      <div className={`max-w-[80%] ${self ? "items-end" : "items-start"} flex flex-col`}>
                        <div className="chat-message-meta mb-1 text-xs">
                          {senderName} · {formatTime(msg.createdAt)}
                        </div>
                        <div
                          className={`chat-message-bubble ${
                            self ? "chat-message-bubble-self" : "chat-message-bubble-other"
                          }`}
                        >
                          {msg.content}
                        </div>
                      </div>
                    </div>
                  );
                })}
                <div ref={messagesEndRef} />
              </div>
              <div className="border-t discord-divider px-4 py-3">
                <form onSubmit={handleSendMessage} className="flex gap-2">
                  <textarea
                    value={inputText}
                    onChange={(event) => setInputText(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" && !event.shiftKey) {
                        event.preventDefault();
                        handleSendMessage(event);
                      }
                    }}
                    placeholder={`聊聊 ${activeTopicRoom?.name || "这个话题"}`}
                    className="h-11 min-h-[44px] flex-1 resize-none rounded-md border border-[#3f4248] bg-[#383a40] px-3 py-2 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-[#5865f2]"
                  />
                  <DiscordButton type="submit" className="!h-11 !min-w-[88px]">
                    发送
                  </DiscordButton>
                </form>
              </div>
            </>
          ) : activeConvId ? (
            <>
              <div className="flex-1 overflow-y-auto px-4 py-4">
                {activeMessages.length === 0 ? (
                  <div className="rounded-xl border border-[#3a3d43] bg-[#2b2d31] p-4 text-sm text-slate-400">
                    开始发送第一条消息吧。
                  </div>
                ) : null}
                {activeMessages.map((msg) => {
                  const self = toIdString(msg.senderId) === toIdString(user?.id);
                  const senderName =
                    msg.sender?.nickname ||
                    msg.sender?.username ||
                    (self ? user?.nickname || user?.username : "Unknown");
                  const senderAvatar = msg.sender?.avatar || (self ? user?.avatar : "");
                  return (
                    <div
                      key={toIdString(msg.id)}
                      className={`chat-message-row group relative mb-2 flex gap-3 px-2 py-1 ${
                        self ? "flex-row-reverse" : ""
                      }`}
                    >
                      <div
                        className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                        style={{
                          backgroundColor: senderAvatar ? "transparent" : getAvatarColor(msg.senderId),
                          overflow: "hidden",
                        }}
                      >
                        {senderAvatar ? (
                          <img src={senderAvatar} alt="avatar" className="h-full w-full object-cover" />
                        ) : (
                          getInitials(senderName)
                        )}
                      </div>
                      <div className={`max-w-[80%] ${self ? "items-end" : "items-start"} flex flex-col`}>
                        <div className="chat-message-meta mb-1 text-xs">
                          {senderName} · {formatTime(msg.createdAt)}
                        </div>
                        <div
                          className={`chat-message-bubble ${
                            self ? "chat-message-bubble-self" : "chat-message-bubble-other"
                          }`}
                        >
                          {msg.content}
                        </div>
                      </div>
                      <div
                        className={`absolute top-0 hidden -translate-y-1/2 items-center gap-1 rounded-md border border-[#42454d] bg-[#222326] px-1 py-1 text-[11px] text-slate-300 shadow-lg group-hover:flex ${
                          self ? "left-14" : "right-3"
                        }`}
                      >
                        <button type="button" className="rounded px-1 hover:bg-[#3b3e45]">
                          回复
                        </button>
                        <button type="button" className="rounded px-1 hover:bg-[#3b3e45]">
                          更多
                        </button>
                      </div>
                    </div>
                  );
                })}
                <div ref={messagesEndRef} />
              </div>
              <div className="border-t discord-divider px-4 py-3">
                <form onSubmit={handleSendMessage} className="flex gap-2">
                  <textarea
                    value={inputText}
                    onChange={(event) => setInputText(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" && !event.shiftKey) {
                        event.preventDefault();
                        handleSendMessage(event);
                      }
                    }}
                    placeholder={`Message ${activeConv?.name || ""}`}
                    className="h-11 min-h-[44px] flex-1 resize-none rounded-md border border-[#3f4248] bg-[#383a40] px-3 py-2 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-[#5865f2]"
                  />
                  <DiscordButton type="submit" className="!h-11 !min-w-[88px]">
                    Send
                  </DiscordButton>
                </form>
              </div>
            </>
          ) : (
            <div className="flex flex-1 items-center justify-center px-4 text-center">
              <div>
                <div className="mx-auto mb-4 h-14 w-14 rounded-2xl bg-[#3a3d43]" />
                <p className="font-display text-xl font-bold text-slate-100">Welcome to Social Chat</p>
                <p className="mt-1 text-sm text-slate-400">从左侧进入官方话题聊天室，或创建一个新的私聊/群聊。</p>
              </div>
            </div>
          )}
        </div>

        <aside className="hidden w-[260px] shrink-0 border-l discord-divider bg-[#2b2d31] xl:flex xl:flex-col">
          <div className="border-b discord-divider px-4 py-3">
            <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">
              {activeTopicRoomId ? "聊天室在线成员" : "成员列表"}
            </p>
            <p className="mt-1 text-xs text-slate-500">当前可见 {memberPanelList.length} 人</p>
          </div>
          <div className="flex-1 overflow-y-auto px-2 py-2">
            {memberPanelList.map((member) => (
              <div
                key={member.id}
                className="mb-1 flex items-center gap-2 rounded-md px-2 py-2 transition hover:bg-[#35373c]"
              >
                <div
                  className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-[10px] font-bold text-white"
                  style={{
                    backgroundColor: member.avatar ? "transparent" : getAvatarColor(member.id),
                    overflow: "hidden",
                  }}
                >
                  {member.avatar ? (
                    <img src={member.avatar} alt="avatar" className="h-full w-full object-cover" />
                  ) : (
                    getInitials(member.name)
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-semibold text-slate-100">{member.name}</p>
                  <p className="truncate text-[11px] text-slate-400">UID: {member.id}</p>
                </div>
                <span className="rounded-full border border-[#41444d] bg-[#232428] px-2 py-0.5 text-[10px] text-slate-400">
                  {member.badge}
                </span>
              </div>
            ))}
          </div>
          <div className="border-t discord-divider px-4 py-3">
            {activeTopicRoomId ? (
              <>
                <p className="text-[11px] uppercase tracking-wider text-slate-500">聊天室状态</p>
                <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
                  <div className="rounded-md border border-[#3f4248] bg-[#232428] px-2 py-2 text-center">
                    <p className="text-[10px] text-slate-500">在线人数</p>
                    <p className="mt-1 font-semibold text-slate-100">{activeTopicRoom?.onlineCount || 0}</p>
                  </div>
                  <div className="rounded-md border border-[#3f4248] bg-[#232428] px-2 py-2 text-center">
                    <p className="text-[10px] text-slate-500">聊天室总数</p>
                    <p className="mt-1 font-semibold text-slate-100">{topicRooms.length}</p>
                  </div>
                </div>
              </>
            ) : (
              <>
                <p className="text-[11px] uppercase tracking-wider text-slate-500">关系快照</p>
                <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
                  <div className="rounded-md border border-[#3f4248] bg-[#232428] px-2 py-2 text-center">
                    <p className="text-[10px] text-slate-500">Following</p>
                    <p className="mt-1 font-semibold text-slate-100">{followingList.length}</p>
                  </div>
                  <div className="rounded-md border border-[#3f4248] bg-[#232428] px-2 py-2 text-center">
                    <p className="text-[10px] text-slate-500">Followers</p>
                    <p className="mt-1 font-semibold text-slate-100">{followersList.length}</p>
                  </div>
                </div>
              </>
            )}
          </div>
        </aside>
      </main>

      {createModalOpen ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/65 p-4">
          <div className="discord-surface w-full max-w-4xl rounded-xl p-5">
            <h3 className="font-display text-xl font-bold text-white">创建群聊</h3>
            <p className="mt-1 text-sm text-slate-400">
              填写群信息，并从右侧好友列表中勾选成员。
            </p>

            <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(320px,360px)]">
              <div className="space-y-3">
                <div>
                  <label className="mb-1 block text-xs font-semibold text-slate-300">群聊名称</label>
                  <DiscordInput
                    value={createGroupName}
                    onChange={(event) => setCreateGroupName(event.target.value)}
                    placeholder="例如 产品讨论组"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-semibold text-slate-300">群头像 URL（可选）</label>
                  <DiscordInput
                    value={createGroupAvatar}
                    onChange={(event) => setCreateGroupAvatar(event.target.value)}
                    placeholder="https://..."
                  />
                </div>
                <div className="rounded-md border border-[#3f4248] bg-[#232428] px-3 py-2 text-xs text-slate-400">
                  需选择至少 2 位好友（不包含自己）才能创建群聊。
                </div>
              </div>

              <div className="min-h-0 rounded-md border border-[#3f4248] bg-[#1e1f22]">
                <div className="flex items-center justify-between border-b border-[#2a2c30] px-3 py-2">
                  <label className="block text-xs font-semibold text-slate-300">群成员（仅好友）</label>
                  <span className="text-[11px] text-slate-400">已选 {createGroupMemberIds.length} 人</span>
                </div>
                <div className="max-h-72 overflow-y-auto">
                  {friendCandidates.length === 0 ? (
                    <p className="px-3 py-4 text-xs text-slate-400">
                      暂无可邀请好友，请先与对方互相关注。
                    </p>
                  ) : (
                    friendCandidates.map((member) => {
                      const memberId = toIdString(member.id);
                      const selected = createGroupMemberIds.includes(memberId);
                      const memberName = member.nickname || member.username || `UID ${memberId}`;
                      return (
                        <label
                          key={memberId}
                          className={`flex cursor-pointer items-center gap-2 border-b border-[#2a2c30] px-3 py-2 transition last:border-b-0 ${
                            selected ? "bg-[#2a315a]" : "hover:bg-[#2a2c30]"
                          }`}
                        >
                          <input
                            type="checkbox"
                            checked={selected}
                            onChange={() => toggleCreateGroupMember(memberId)}
                            className="h-4 w-4 accent-[#5865f2]"
                          />
                          <div
                            className="flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full text-[10px] font-bold text-white"
                            style={{
                              backgroundColor: member.avatar ? "transparent" : getAvatarColor(memberId),
                            }}
                          >
                            {member.avatar ? (
                              <img src={member.avatar} alt="avatar" className="h-full w-full object-cover" />
                            ) : (
                              getInitials(memberName)
                            )}
                          </div>
                          <div className="min-w-0">
                            <p className="truncate text-xs font-semibold text-slate-100">
                              {memberName}
                            </p>
                            <p className="truncate text-[11px] text-slate-400">UID: {memberId}</p>
                          </div>
                        </label>
                      );
                    })
                  )}
                </div>
              </div>
            </div>
            <div className="mt-4 grid grid-cols-2 gap-2">
              <DiscordSecondaryButton onClick={closeCreateConversationModal}>
                取消
              </DiscordSecondaryButton>
              <DiscordButton onClick={handleCreateConversation}>创建群聊</DiscordButton>
            </div>
          </div>
        </div>
      ) : null}

      {followModalOpen ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/65 p-4">
          <div className="discord-surface w-full max-w-md rounded-xl p-5">
            <h3 className="font-display text-xl font-bold text-white">关注用户</h3>
            <p className="mt-1 text-sm text-slate-400">输入 UID 建立关注关系。</p>
            <DiscordInput
              className="mt-4"
              value={followTargetId}
              onChange={(event) => setFollowTargetId(event.target.value)}
              placeholder="例如 20000002"
            />
            <div className="mt-4 grid grid-cols-2 gap-2">
              <DiscordSecondaryButton onClick={() => setFollowModalOpen(false)}>
                取消
              </DiscordSecondaryButton>
              <DiscordButton onClick={() => handleFollowUser()}>确认关注</DiscordButton>
            </div>
          </div>
        </div>
      ) : null}

      {profileModalOpen ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/65 p-4">
          <div className="discord-surface w-full max-w-lg rounded-xl p-5">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-xl font-bold text-white">编辑资料</h3>
              <DiscordSecondaryButton
                className="!h-8 !border-[#5f2a33] !bg-[#3b1f24] !px-3 !text-xs !text-[#ffb4bf] hover:!bg-[#4a252d]"
                onClick={handleLogout}
              >
                退出登录
              </DiscordSecondaryButton>
            </div>

            <div className="mt-4 space-y-3">
              <div>
                <label className="mb-1 block text-xs font-semibold text-slate-300">头像 URL</label>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    className="h-10 shrink-0 rounded-md border border-[#4f545c] bg-[#2b2d31] px-3 text-xs text-slate-200 hover:bg-[#32353b]"
                  >
                    上传
                  </button>
                  <input
                    ref={fileInputRef}
                    type="file"
                    className="hidden"
                    accept="image/*"
                    onChange={handleFileUpload}
                  />
                  <DiscordInput
                    value={profileForm.avatar}
                    onChange={(event) =>
                      setProfileForm((prev) => ({ ...prev, avatar: event.target.value }))
                    }
                  />
                </div>
              </div>

              <div>
                <label className="mb-1 block text-xs font-semibold text-slate-300">昵称</label>
                <DiscordInput
                  value={profileForm.nickname}
                  onChange={(event) =>
                    setProfileForm((prev) => ({ ...prev, nickname: event.target.value }))
                  }
                />
              </div>

              <div>
                <label className="mb-1 block text-xs font-semibold text-slate-300">简介</label>
                <textarea
                  rows={3}
                  value={profileForm.bio}
                  onChange={(event) =>
                    setProfileForm((prev) => ({ ...prev, bio: event.target.value }))
                  }
                  className="w-full rounded-md border border-[#3f4248] bg-[#1e1f22] px-3 py-2 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-[#5865f2]"
                />
              </div>
            </div>

            <div className="mt-4 grid grid-cols-2 gap-2">
              <DiscordSecondaryButton onClick={() => setProfileModalOpen(false)}>
                取消
              </DiscordSecondaryButton>
              <DiscordButton onClick={handleUpdateProfile}>保存修改</DiscordButton>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
