export default function GroupAnnouncementCard({
  roleLabel,
  memberCount,
  groupKindLabel,
  announcementDraft,
  canManageGroup,
  onChangeAnnouncement,
  onSaveAnnouncement,
  ChatButton,
}) {
  return (
    <div className="space-y-4 rounded-lg border border-border bg-card p-4">
      <div className="rounded-lg border border-border bg-muted p-3 text-xs text-muted-foreground">
        <p>身份：{roleLabel}</p>
        <p className="mt-1">成员数：{memberCount}</p>
        <p className="mt-1">群类型：{groupKindLabel}</p>
      </div>
      <div>
        <label className="mb-1.5 block text-xs font-semibold text-foreground">群公告</label>
        <textarea
          value={announcementDraft}
          onChange={(event) => onChangeAnnouncement(event.target.value)}
          className="min-h-[150px] w-full resize-none rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:ring-2 focus:ring-ring"
        />
      </div>
      {canManageGroup ? <ChatButton className="w-full" onClick={onSaveAnnouncement}>保存群公告</ChatButton> : null}
    </div>
  );
}
