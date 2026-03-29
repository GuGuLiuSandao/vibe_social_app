export default function GroupProfileCard({
  detail,
  canManageGroup,
  inviteTargetId,
  onChangeDetail,
  onChangeInviteTarget,
  onSaveProfile,
  onInvite,
  DiscordButton,
  DiscordInput,
  joinModePrivate,
  joinModeApproval,
  joinModePublic,
}) {
  if (!detail) return null;

  return (
    <div className="space-y-3">
      <div>
        <label className="mb-1 block text-xs font-semibold text-slate-300">群名称</label>
        <DiscordInput value={detail.name || ""} onChange={(event) => onChangeDetail({ name: event.target.value })} />
      </div>
      <div>
        <label className="mb-1 block text-xs font-semibold text-slate-300">群头像 URL</label>
        <DiscordInput value={detail.avatar || ""} onChange={(event) => onChangeDetail({ avatar: event.target.value })} />
      </div>
      <div>
        <label className="mb-1 block text-xs font-semibold text-slate-300">群简介</label>
        <textarea
          value={detail.description || ""}
          onChange={(event) => onChangeDetail({ description: event.target.value })}
          className="min-h-[84px] w-full resize-none rounded-md border border-[#3f4248] bg-[#1e1f22] px-3 py-2 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-[#5865f2]"
        />
      </div>
      <div>
        <label className="mb-1 block text-xs font-semibold text-slate-300">加入方式</label>
        <select
          value={Number(detail.joinMode || joinModePrivate)}
          onChange={(event) => onChangeDetail({ joinMode: Number(event.target.value) })}
          className="h-10 w-full rounded-md border border-[#3f4248] bg-[#1e1f22] px-3 text-sm text-slate-100 outline-none focus:border-[#5865f2]"
        >
          <option value={joinModePrivate}>私密群</option>
          <option value={joinModeApproval}>申请加入群</option>
          <option value={joinModePublic}>公开群</option>
        </select>
      </div>
      {canManageGroup ? <DiscordButton className="w-full" onClick={onSaveProfile}>保存群资料</DiscordButton> : null}
      {canManageGroup ? (
        <div>
          <label className="mb-1 block text-xs font-semibold text-slate-300">邀请成员（输入 UID）</label>
          <div className="flex gap-2">
            <DiscordInput value={inviteTargetId} onChange={(event) => onChangeInviteTarget(event.target.value)} placeholder="输入 UID" />
            <DiscordButton className="!px-3" onClick={onInvite}>邀请</DiscordButton>
          </div>
        </div>
      ) : null}
    </div>
  );
}
