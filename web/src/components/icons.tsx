import {
  ArrowDown,
  ArrowUp,
  CalendarCheck,
  CalendarDays,
  CalendarMinus,
  Check,
  ChevronLeft,
  ChevronRight,
  CircleAlert,
  FileText,
  Flag,
  FolderClosed,
  FolderPlus,
  GripVertical,
  Image,
  Inbox,
  Link,
  ListChecks,
  type LucideIcon,
  type LucideProps,
  NotebookPen,
  Paperclip,
  Repeat,
  StickyNote,
  Sun,
  Trash2,
  Undo2,
  X,
} from "lucide-react";

// Lucide, drawn the DESIGN.md way: 1.8 px stroke, round caps, hidden from
// screen readers (the button or text around an icon carries its meaning).
export type IconProps = Omit<LucideProps, "ref">;

function icon(Lucide: LucideIcon) {
  return function DesignIcon({ size = 24, ...props }: IconProps) {
    return <Lucide size={size} strokeWidth={1.8} aria-hidden="true" focusable="false" {...props} />;
  };
}

export const TodayIcon = icon(Sun);
export const InboxIcon = icon(Inbox);
export const PlanIcon = icon(CalendarDays);
export const GoalsIcon = icon(Flag);
export const GripIcon = icon(GripVertical);
export const CheckIcon = icon(Check);
export const CloseIcon = icon(X);
export const UpIcon = icon(ArrowUp);
export const DownIcon = icon(ArrowDown);
export const RepeatIcon = icon(Repeat);
export const NoteIcon = icon(StickyNote);
export const LinkIcon = icon(Link);
export const FileIcon = icon(FileText);
export const ImageIcon = icon(Image);
export const BackIcon = icon(ChevronLeft);
export const NextIcon = icon(ChevronRight);
export const ProjectIcon = icon(FolderClosed);
export const MakeProjectIcon = icon(FolderPlus);
export const AttachmentIcon = icon(Paperclip);
export const TasksIcon = icon(ListChecks);
export const NotesIcon = icon(NotebookPen);
export const PlannedIcon = icon(CalendarCheck);
export const UnplannedIcon = icon(CalendarMinus);
export const DeleteIcon = icon(Trash2);
export const NoticeIcon = icon(CircleAlert);
export const UndoIcon = icon(Undo2);
