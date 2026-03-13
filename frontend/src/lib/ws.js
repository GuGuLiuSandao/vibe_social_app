import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import {
  AccountPayloadSchema,
  AuthRequestSchema,
  PingSchema,
  SearchUserRequestSchema,
  UpdateProfileRequestSchema,
  UploadAvatarRequestSchema,
} from "../proto/account/account_pb.ts";
import {
  RelationPayloadSchema,
  FollowRequestSchema,
  UnfollowRequestSchema,
  GetFollowingRequestSchema,
  GetFollowersRequestSchema,
  GetFriendsRequestSchema,
  BlockRequestSchema,
  UnblockRequestSchema,
  GetBlockedRequestSchema,
} from "../proto/relation/relation_pb.ts";
import { WsMessageSchema, WsMessageType } from "../proto/ws_pb.ts";

const WS_BASE = "ws://localhost:8080/ws";

export function buildWsUrl(uid, token) {
  // Always attach token if available, regardless of whitelist status
  // Backend requires token for all connections now
  let url = `${WS_BASE}?uid=${uid}`;
  if (token) {
    url += `&token=${encodeURIComponent(token)}`;
  }
  return url;
}

export function buildAccountPing(requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_PING,
    timestamp,
    payload: {
      case: "account",
      value: create(AccountPayloadSchema, {
        payload: {
          case: "ping",
          value: create(PingSchema, {}),
        },
      }),
    },
  });
}

export function buildSearchUserRequest(query, requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_ACCOUNT_SEARCH_USER,
    timestamp,
    payload: {
      case: "account",
      value: create(AccountPayloadSchema, {
        payload: {
          case: "searchUser",
          value: create(SearchUserRequestSchema, {
            query,
          }),
        },
      }),
    },
  });
}

export function buildAuthRequest(uid = 0, requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_AUTH,
    timestamp,
    payload: {
      case: "account",
      value: create(AccountPayloadSchema, {
        payload: {
          case: "auth",
          value: create(AuthRequestSchema, {
            uid: uid ? BigInt(uid) : 0n,
          }),
        },
      }),
    },
  });
}

export function buildUpdateProfileRequest(updates, requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_ACCOUNT_UPDATE_PROFILE,
    timestamp,
    payload: {
      case: "account",
      value: create(AccountPayloadSchema, {
        payload: {
          case: "updateProfile",
          value: create(UpdateProfileRequestSchema, {
            nickname: updates?.nickname || "",
            avatar: updates?.avatar || "",
            bio: updates?.bio || "",
          }),
        },
      }),
    },
  });
}

export function buildUploadAvatarRequest(filename, data, requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_ACCOUNT_UPLOAD_AVATAR,
    timestamp,
    payload: {
      case: "account",
      value: create(AccountPayloadSchema, {
        payload: {
          case: "uploadAvatar",
          value: create(UploadAvatarRequestSchema, {
            filename: filename || "",
            data: data || new Uint8Array(),
          }),
        },
      }),
    },
  });
}

export function buildFollowRequest(targetUid, requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_RELATION_FOLLOW,
    timestamp,
    payload: {
      case: "relation",
      value: create(RelationPayloadSchema, {
        payload: {
          case: "follow",
          value: create(FollowRequestSchema, {
            targetUid: BigInt(targetUid),
          }),
        },
      }),
    },
  });
}

export function buildUnfollowRequest(targetUid, requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_RELATION_UNFOLLOW,
    timestamp,
    payload: {
      case: "relation",
      value: create(RelationPayloadSchema, {
        payload: {
          case: "unfollow",
          value: create(UnfollowRequestSchema, {
            targetUid: BigInt(targetUid),
          }),
        },
      }),
    },
  });
}

export function buildGetFollowingRequest(requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_RELATION_GET_FOLLOWING,
    timestamp,
    payload: {
      case: "relation",
      value: create(RelationPayloadSchema, {
        payload: {
          case: "getFollowing",
          value: create(GetFollowingRequestSchema, {}),
        },
      }),
    },
  });
}

export function buildGetFollowersRequest(requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_RELATION_GET_FOLLOWERS,
    timestamp,
    payload: {
      case: "relation",
      value: create(RelationPayloadSchema, {
        payload: {
          case: "getFollowers",
          value: create(GetFollowersRequestSchema, {}),
        },
      }),
    },
  });
}

export function buildGetFriendsRequest(pageSize = 20, cursor = "", requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_RELATION_GET_FRIENDS,
    timestamp,
    payload: {
      case: "relation",
      value: create(RelationPayloadSchema, {
        payload: {
          case: "getFriends",
          value: create(GetFriendsRequestSchema, {
            pageSize,
            cursor
          }),
        },
      }),
    },
  });
}

export function buildBlockRequest(targetUid, requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_RELATION_BLOCK,
    timestamp,
    payload: {
      case: "relation",
      value: create(RelationPayloadSchema, {
        payload: {
          case: "block",
          value: create(BlockRequestSchema, {
            targetUid: BigInt(targetUid),
          }),
        },
      }),
    },
  });
}

export function buildUnblockRequest(targetUid, requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_RELATION_UNBLOCK,
    timestamp,
    payload: {
      case: "relation",
      value: create(RelationPayloadSchema, {
        payload: {
          case: "unblock",
          value: create(UnblockRequestSchema, {
            targetUid: BigInt(targetUid),
          }),
        },
      }),
    },
  });
}

export function buildGetBlockedRequest(requestId = BigInt(Date.now())) {
  const timestamp = BigInt(Date.now());
  return create(WsMessageSchema, {
    requestId,
    type: WsMessageType.WS_TYPE_RELATION_GET_BLOCKED,
    timestamp,
    payload: {
      case: "relation",
      value: create(RelationPayloadSchema, {
        payload: {
          case: "getBlocked",
          value: create(GetBlockedRequestSchema, {}),
        },
      }),
    },
  });
}

export function encodeWsMessage(message) {
  return toBinary(WsMessageSchema, message);
}

export function decodeWsMessage(buffer) {
  const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
  return fromBinary(WsMessageSchema, bytes);
}
