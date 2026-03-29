import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import GroupAnnouncementCard from "./GroupAnnouncementCard";
import GroupJoinRequestsCard from "./GroupJoinRequestsCard";
import GroupMembersCard from "./GroupMembersCard";
import GroupProfileCard from "./GroupProfileCard";

function DiscordButton(props) {
  return <button {...props} />;
}

function DiscordSecondaryButton(props) {
  return <button {...props} />;
}

function DiscordInput(props) {
  return <input {...props} />;
}

describe("group management components", () => {
  it("renders and edits group profile card", () => {
    const onChangeDetail = vi.fn();
    const onInvite = vi.fn();
    render(
      <GroupProfileCard
        detail={{ name: "测试群", avatar: "", description: "简介", joinMode: 1 }}
        canManageGroup
        inviteTargetId=""
        onChangeDetail={onChangeDetail}
        onChangeInviteTarget={() => {}}
        onSaveProfile={() => {}}
        onInvite={onInvite}
        DiscordButton={DiscordButton}
        DiscordInput={DiscordInput}
        joinModePrivate={1}
        joinModeApproval={2}
        joinModePublic={3}
      />,
    );

    expect(screen.getByDisplayValue("测试群")).toBeInTheDocument();
    fireEvent.change(screen.getByDisplayValue("测试群"), { target: { value: "新群名" } });
    expect(onChangeDetail).toHaveBeenCalled();
  });

  it("renders announcement, members, and join requests actions", () => {
    const onApprove = vi.fn();
    const onReject = vi.fn();
    const onSetAdmin = vi.fn();

    render(
      <>
        <GroupAnnouncementCard
          roleLabel="群主"
          memberCount={3}
          groupKindLabel="玩家自建群"
          announcementDraft="公告"
          canManageGroup
          onChangeAnnouncement={() => {}}
          onSaveAnnouncement={() => {}}
          DiscordButton={DiscordButton}
        />
        <GroupMembersCard
          members={[
            { userId: 1n, nickname: "我", role: 1 },
            { userId: 2n, nickname: "成员A", role: 3 },
          ]}
          currentUserId={1}
          activeGroupRole={1}
          groupRoleOwner={1}
          groupRoleAdmin={2}
          groupRoleMember={3}
          canManageGroup
          canTransferOwnership
          onSetAdmin={onSetAdmin}
          onUnsetAdmin={() => {}}
          onTransferOwnership={() => {}}
          onRemoveMember={() => {}}
          DiscordSecondaryButton={DiscordSecondaryButton}
        />
        <GroupJoinRequestsCard
          requests={[{ id: 7n, applicantNickname: "申请人", status: 1, message: "想加入" }]}
          pendingStatus={1}
          onApprove={onApprove}
          onReject={onReject}
          DiscordButton={DiscordButton}
          DiscordSecondaryButton={DiscordSecondaryButton}
        />
      </>,
    );

    expect(screen.getByText("成员操作 (2)")).toBeInTheDocument();
    expect(screen.getByText("入群申请 (1)")).toBeInTheDocument();
    fireEvent.click(screen.getByText("设为管理员"));
    fireEvent.click(screen.getByText("通过"));
    fireEvent.click(screen.getByText("拒绝"));
    expect(onSetAdmin).toHaveBeenCalled();
    expect(onApprove).toHaveBeenCalled();
    expect(onReject).toHaveBeenCalled();
  });
});
