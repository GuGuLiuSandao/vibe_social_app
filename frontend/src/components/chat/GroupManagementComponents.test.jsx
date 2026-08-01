import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import GroupCreateModal from "./GroupCreateModal";
import GroupJoinRequestsCard from "./GroupJoinRequestsCard";
import GroupMembersCard from "./GroupMembersCard";
import GroupProfileCard from "./GroupProfileCard";

function ChatButton(props) {
  return <button {...props} />;
}

function ChatSecondaryButton(props) {
  return <button {...props} />;
}

function ChatInput(props) {
  return <input {...props} />;
}

const roles = {
  groupRoleOwner: 1,
  groupRoleAdmin: 2,
  groupRoleMember: 3,
};

function renderMembers(activeGroupRole, memberRole, overrides = {}) {
  const callbacks = {
    onSetAdmin: vi.fn(),
    onUnsetAdmin: vi.fn(),
    onTransferOwnership: vi.fn(),
    onRemoveMember: vi.fn(),
  };
  const rendered = render(
    <GroupMembersCard
      members={[{ userId: 2n, nickname: "成员", role: memberRole }]}
      currentUserId={1}
      activeGroupRole={activeGroupRole}
      {...roles}
      canManageGroup
      canTransferOwnership
      {...callbacks}
      {...overrides}
      ChatSecondaryButton={ChatSecondaryButton}
    />,
  );
  return Object.assign(callbacks, rendered);
}

describe("DLQ_TC_037 group management component behavior", () => {
  it("propagates create form edits and actions through the neutral chat controls", () => {
    const onChangeForm = vi.fn();
    const onClose = vi.fn();
    const onSubmit = vi.fn();
    render(
      <GroupCreateModal
        open
        form={{ name: "旧群名", avatar: "old.png", description: "", joinMode: 1 }}
        selectedMemberIds={[]}
        friendCandidates={[]}
        onChangeForm={onChangeForm}
        onToggleMember={() => {}}
        onClose={onClose}
        onSubmit={onSubmit}
        ChatButton={ChatButton}
        ChatSecondaryButton={ChatSecondaryButton}
        ChatInput={ChatInput}
        getAvatarColor={() => "red"}
        getInitials={() => "A"}
        toIdString={String}
        joinModePrivate={1}
        joinModeApproval={2}
        joinModePublic={3}
      />,
    );

    fireEvent.change(screen.getByDisplayValue("旧群名"), { target: { value: "新群名" } });
    fireEvent.change(screen.getByDisplayValue("old.png"), { target: { value: "new.png" } });
    fireEvent.click(screen.getByText("取消"));
    fireEvent.click(screen.getByRole("button", { name: "创建群聊" }));
    expect(onChangeForm).toHaveBeenCalledWith({ name: "新群名" });
    expect(onChangeForm).toHaveBeenCalledWith({ avatar: "new.png" });
    expect(onClose).toHaveBeenCalledOnce();
    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it("preserves profile values, fallbacks, edits, and invite actions", () => {
    const onChangeDetail = vi.fn();
    const onChangeInviteTarget = vi.fn();
    const onSaveProfile = vi.fn();
    const onInvite = vi.fn();
    const { rerender } = render(
      <GroupProfileCard
        detail={{ name: "测试群", avatar: "avatar.png", description: "简介", joinMode: 1 }}
        canManageGroup
        inviteTargetId="7"
        onChangeDetail={onChangeDetail}
        onChangeInviteTarget={onChangeInviteTarget}
        onSaveProfile={onSaveProfile}
        onInvite={onInvite}
        ChatButton={ChatButton}
        ChatInput={ChatInput}
        joinModePrivate={1}
        joinModeApproval={2}
        joinModePublic={3}
      />,
    );

    fireEvent.change(screen.getByDisplayValue("测试群"), { target: { value: "新群名" } });
    fireEvent.change(screen.getByDisplayValue("avatar.png"), { target: { value: "new.png" } });
    fireEvent.change(screen.getByDisplayValue("7"), { target: { value: "8" } });
    fireEvent.click(screen.getByText("保存群资料"));
    fireEvent.click(screen.getByText("邀请"));
    expect(onChangeDetail).toHaveBeenCalledWith({ name: "新群名" });
    expect(onChangeDetail).toHaveBeenCalledWith({ avatar: "new.png" });
    expect(onChangeInviteTarget).toHaveBeenCalledWith("8");
    expect(onSaveProfile).toHaveBeenCalledOnce();
    expect(onInvite).toHaveBeenCalledOnce();

    rerender(
      <GroupProfileCard
        detail={{ name: "", avatar: "", description: "", joinMode: 1 }}
        canManageGroup={false}
        inviteTargetId=""
        onChangeDetail={onChangeDetail}
        onChangeInviteTarget={onChangeInviteTarget}
        onSaveProfile={onSaveProfile}
        onInvite={onInvite}
        ChatButton={ChatButton}
        ChatInput={ChatInput}
        joinModePrivate={1}
        joinModeApproval={2}
        joinModePublic={3}
      />,
    );
    expect(screen.getAllByDisplayValue("")).toHaveLength(3);
    expect(screen.queryByText("保存群资料")).not.toBeInTheDocument();
  });

  it("dispatches pending join-request decisions with the request identity", () => {
    const onApprove = vi.fn();
    const onReject = vi.fn();
    render(
      <GroupJoinRequestsCard
        requests={[{ id: 7n, applicantNickname: "申请人", status: 1, message: "想加入" }]}
        pendingStatus={1}
        onApprove={onApprove}
        onReject={onReject}
        ChatButton={ChatButton}
        ChatSecondaryButton={ChatSecondaryButton}
      />,
    );
    fireEvent.click(screen.getByText("通过"));
    fireEvent.click(screen.getByText("拒绝"));
    expect(onApprove).toHaveBeenCalledWith(7n);
    expect(onReject).toHaveBeenCalledWith(7n);
  });

  it("shows exactly the owner actions allowed for an ordinary member", () => {
    const callbacks = renderMembers(roles.groupRoleOwner, roles.groupRoleMember);
    for (const label of ["设为管理员", "转让群主", "移出群聊"]) {
      fireEvent.click(screen.getByText(label));
    }
    expect(screen.queryByText("取消管理员")).not.toBeInTheDocument();
    expect(callbacks.onSetAdmin).toHaveBeenCalledWith("2");
    expect(callbacks.onTransferOwnership).toHaveBeenCalledWith("2");
    expect(callbacks.onRemoveMember).toHaveBeenCalledWith("2");
  });

  it("shows exactly the owner actions allowed for an administrator", () => {
    const callbacks = renderMembers(roles.groupRoleOwner, roles.groupRoleAdmin);
    expect(screen.queryByText("设为管理员")).not.toBeInTheDocument();
    fireEvent.click(screen.getByText("取消管理员"));
    expect(screen.getByText("转让群主")).toBeInTheDocument();
    expect(screen.getByText("移出群聊")).toBeInTheDocument();
    expect(callbacks.onUnsetAdmin).toHaveBeenCalledWith("2");
  });

  it("does not offer owner-only actions to an administrator", () => {
    const { rerender } = render(
      <GroupMembersCard
        members={[{ userId: 2n, nickname: "普通成员", role: roles.groupRoleMember }]}
        currentUserId={1}
        activeGroupRole={roles.groupRoleAdmin}
        {...roles}
        canManageGroup
        canTransferOwnership={false}
        onSetAdmin={() => {}}
        onUnsetAdmin={() => {}}
        onTransferOwnership={() => {}}
        onRemoveMember={() => {}}
        ChatSecondaryButton={ChatSecondaryButton}
      />,
    );
    expect(screen.getByText("移出群聊")).toBeInTheDocument();
    expect(screen.queryByText("设为管理员")).not.toBeInTheDocument();
    expect(screen.queryByText("转让群主")).not.toBeInTheDocument();

    rerender(
      <GroupMembersCard
        members={[{ userId: 2n, nickname: "管理员", role: roles.groupRoleAdmin }]}
        currentUserId={1}
        activeGroupRole={roles.groupRoleAdmin}
        {...roles}
        canManageGroup
        canTransferOwnership={false}
        onSetAdmin={() => {}}
        onUnsetAdmin={() => {}}
        onTransferOwnership={() => {}}
        onRemoveMember={() => {}}
        ChatSecondaryButton={ChatSecondaryButton}
      />,
    );
    expect(screen.queryByText("移出群聊")).not.toBeInTheDocument();
    expect(screen.queryByText("取消管理员")).not.toBeInTheDocument();

    rerender(
      <GroupMembersCard
        members={[{ userId: 2n, nickname: "普通成员", role: roles.groupRoleMember }]}
        currentUserId={1}
        activeGroupRole={99}
        {...roles}
        canManageGroup
        canTransferOwnership={false}
        onSetAdmin={() => {}}
        onUnsetAdmin={() => {}}
        onTransferOwnership={() => {}}
        onRemoveMember={() => {}}
        ChatSecondaryButton={ChatSecondaryButton}
      />,
    );
    expect(screen.queryByText("移出群聊")).not.toBeInTheDocument();
  });

  it("offers no management action for the owner row or without management permission", () => {
    const { unmount } = renderMembers(roles.groupRoleOwner, roles.groupRoleOwner);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    unmount();
    renderMembers(roles.groupRoleOwner, roles.groupRoleMember, { canManageGroup: false });
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
