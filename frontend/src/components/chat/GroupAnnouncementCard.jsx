export default function GroupAnnouncementCard({
  roleLabel,
  memberCount,
  groupKindLabel,
  announcementDraft,
  canManageGroup,
  onChangeAnnouncement,
  onSaveAnnouncement,
  DiscordButton,
}) {
  return (
    <div className="space-y-3">
      <div className="rounded-md border border-[#3f4248] bg-[#232428] p-3 text-xs text-slate-400">
        <p>身份：{roleLabel}</p>
        <p className="mt-1">成员数：{memberCount}</p>
        <p className="mt-1">群类型：{groupKindLabel}</p>
      </div>
      <div>
        <label className="mb-1 block text-xs font-semibold text-slate-300">群公告</label>
        <textarea
          value={announcementDraft}
          onChange={(event) => onChangeAnnouncement(event.target.value)}
          className="min-h-[140px] w-full resize-none rounded-md border border-[#3f4248] bg-[#1e1f22] px-3 py-2 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-[#5865f2]"
        />
      </div>
      {canManageGroup ? <DiscordButton className="w-full" onClick={onSaveAnnouncement}>保存群公告</DiscordButton> : null}
    </div>
  );
}
