export default function GroupProfileCard({
  detail,
  canManageGroup,
  inviteTargetId,
  onChangeDetail,
  onChangeInviteTarget,
  onSaveProfile,
  onInvite,
  ChatButton,
  ChatInput,
  joinModePrivate,
  joinModeApproval,
  joinModePublic,
}) {
  if (!detail) return null;

  return (
    <div className="space-y-4 rounded-lg border border-border bg-card p-4">
      <div>
        <label className="mb-1.5 block text-xs font-semibold text-foreground">群名称</label>
        <ChatInput value={detail.name || ""} onChange={(event) => onChangeDetail({ name: event.target.value })} />
      </div>
      <div>
        <label className="mb-1.5 block text-xs font-semibold text-foreground">群头像 URL</label>
        <ChatInput value={detail.avatar || ""} onChange={(event) => onChangeDetail({ avatar: event.target.value })} />
      </div>
      <div>
        <label className="mb-1.5 block text-xs font-semibold text-foreground">群简介</label>
        <textarea
          value={detail.description || ""}
          onChange={(event) => onChangeDetail({ description: event.target.value })}
          className="min-h-[110px] w-full resize-none rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:ring-2 focus:ring-ring"
        />
      </div>
      <div>
        <label className="mb-1.5 block text-xs font-semibold text-foreground">加入方式</label>
        <select
          value={Number(detail.joinMode || joinModePrivate)}
          onChange={(event) => onChangeDetail({ joinMode: Number(event.target.value) })}
          className="h-10 w-full rounded-lg border border-input bg-background px-3 text-sm text-foreground outline-none focus:ring-2 focus:ring-ring"
        >
          <option value={joinModePrivate}>私密群</option>
          <option value={joinModeApproval}>申请加入群</option>
          <option value={joinModePublic}>公开群</option>
        </select>
      </div>
      {canManageGroup ? <ChatButton className="w-full" onClick={onSaveProfile}>保存群资料</ChatButton> : null}
      {canManageGroup ? (
        <div>
          <label className="mb-1.5 block text-xs font-semibold text-foreground">邀请成员（输入 UID）</label>
          <div className="flex gap-2">
            <ChatInput value={inviteTargetId} onChange={(event) => onChangeInviteTarget(event.target.value)} placeholder="输入 UID" />
            <ChatButton className="px-3" onClick={onInvite}>邀请</ChatButton>
          </div>
        </div>
      ) : null}
    </div>
  );
}
