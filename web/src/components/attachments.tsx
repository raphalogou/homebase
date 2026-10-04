import { type DragEvent, type FormEvent, useId, useRef, useState } from "react";
import { ActionError } from "@/data/hooks";
import { type Owner, useAttachments, useFileMeta, useStructureActions } from "@/data/structure";
import type { Attachment } from "@/data/types";
import { useIsDesktop } from "@/lib/use-media";
import { cn } from "@/lib/utils";
import {
  AttachmentIcon,
  CloseIcon,
  DeleteIcon,
  FileIcon,
  ImageIcon,
  LinkIcon,
  NoteIcon,
  NoticeIcon,
} from "./icons";
import { SectionLabel } from "./section";
import { useNotify } from "./toaster";
import { Button } from "./ui/button";
import { Input, Label, Textarea } from "./ui/input";

type Adding = "link" | "note" | null;

// DESIGN.md "Attachment row": a 44 px icon box, a title, one meta line and a
// remove button. Beneath: Add link, Add note, Upload file; desktop adds a
// dashed drop area.
export function Attachments({
  owner,
  heading = "Attachments",
}: {
  owner: Owner;
  heading?: string;
}) {
  const list = useAttachments(owner);
  const actions = useStructureActions();
  const desktop = useIsDesktop();
  const [adding, setAdding] = useState<Adding>(null);
  const notify = useNotify();
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<{ name: string; message: string } | null>(null);
  const [dragging, setDragging] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const headingId = useId();

  async function run(fn: () => Promise<unknown>) {
    try {
      await fn();
      return true;
    } catch (err) {
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
      return false;
    }
  }

  // A failed upload stays under the buttons, with the file's name, until the
  // next try; a toast would be gone before the person reads which file it was.
  async function uploadAll(files: FileList | File[]) {
    setBusy(true);
    setFailed(null);
    for (const f of Array.from(files)) {
      try {
        await actions.upload(owner, f);
      } catch (err) {
        if (!(err instanceof ActionError)) throw err;
        setFailed({ name: f.name, message: err.message });
        break;
      }
      notify({ title: `Uploaded ${f.name}`, icon: AttachmentIcon });
    }
    setBusy(false);
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    setDragging(false);
    if (e.dataTransfer.files.length) void uploadAll(e.dataTransfer.files);
  }

  return (
    <section aria-labelledby={headingId}>
      <SectionLabel id={headingId}>{heading}</SectionLabel>
      {list.length === 0 && adding === null && (
        <p className="border-t border-line pt-3 text-muted-foreground">
          Links, notes and files for this item.
        </p>
      )}
      {list.length > 0 && (
        <ul>
          {list.map((a) => (
            <AttachmentRow
              key={a.id}
              att={a}
              onRemove={() =>
                void run(async () => {
                  const undo = await actions.removeAttachment(a.id);
                  notify({
                    title: "Attachment removed",
                    icon: DeleteIcon,
                    action: { label: "Undo", run: () => void undo() },
                  });
                })
              }
            />
          ))}
        </ul>
      )}

      {adding === "link" && (
        <LinkForm
          onCancel={() => setAdding(null)}
          onAdd={async (url, name) => {
            if (await run(() => actions.addLink(owner, url, name))) setAdding(null);
          }}
        />
      )}
      {adding === "note" && (
        <NoteForm
          onCancel={() => setAdding(null)}
          onAdd={async (body) => {
            if (await run(() => actions.addNote(owner, body))) setAdding(null);
          }}
        />
      )}

      <div
        className={cn(
          "flex flex-wrap gap-x-5 gap-y-1",
          list.length > 0 && "border-t border-line pt-2",
        )}
      >
        <Button
          variant="text"
          aria-expanded={adding === "link"}
          onClick={() => setAdding(adding === "link" ? null : "link")}
        >
          Add link
        </Button>
        <Button
          variant="text"
          aria-expanded={adding === "note"}
          onClick={() => setAdding(adding === "note" ? null : "note")}
        >
          Add note
        </Button>
        <Button variant="text" disabled={busy} onClick={() => fileInput.current?.click()}>
          {busy ? "Uploading" : "Upload file"}
        </Button>
        <input
          ref={fileInput}
          type="file"
          multiple
          className="sr-only"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => {
            if (e.target.files?.length) void uploadAll(e.target.files);
            e.target.value = "";
          }}
        />
      </div>

      {failed && (
        <div className="mt-3 flex gap-2.5" role="alert">
          <NoticeIcon size={18} className="mt-[3px] shrink-0" />
          <p>
            <strong className="block font-semibold">{failed.name} did not upload</strong>
            <span className="text-muted-foreground">{failed.message}</span>
          </p>
        </div>
      )}

      {desktop && (
        <button
          type="button"
          disabled={busy}
          onClick={() => fileInput.current?.click()}
          onDragOver={(e) => {
            e.preventDefault();
            setDragging(true);
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={onDrop}
          className={cn(
            "mt-3 grid min-h-20 w-full place-items-center rounded-[12px] border border-dashed border-border-strong px-4 text-sm text-muted-foreground",
            dragging && "bg-soft",
          )}
        >
          Drop files here, or choose them. Up to 25 MB each.
        </button>
      )}
    </section>
  );
}

function kindLabel(name: string, type: string): string {
  if (type === "application/pdf") return "PDF";
  if (type.startsWith("image/")) return "Image";
  const ext = name.includes(".") ? name.split(".").pop() : "";
  return ext && ext.length <= 5 ? ext.toUpperCase() : "File";
}

function sizeText(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function hostOf(url: string): string {
  try {
    const u = new URL(url);
    return u.protocol === "mailto:" ? u.pathname : u.hostname.replace(/^www\./, "");
  } catch {
    return url;
  }
}

function AttachmentRow({ att, onRemove }: { att: Attachment; onRemove: () => void }) {
  const meta = useFileMeta(att.kind === "file" ? att.fileSha : null);
  const [editing, setEditing] = useState(false);
  const isImage = att.kind === "file" && meta?.type.startsWith("image/");
  const Icon =
    att.kind === "note"
      ? NoteIcon
      : att.kind === "link"
        ? LinkIcon
        : isImage
          ? ImageIcon
          : FileIcon;

  let title = att.name;
  let sub = "";
  let href: string | null = null;
  if (att.kind === "link" && att.url) {
    title = att.name || hostOf(att.url);
    sub = hostOf(att.url);
    href = att.url;
  } else if (att.kind === "note") {
    const lines = (att.body ?? "").trim().split("\n");
    title = att.name || lines[0] || "Note";
    sub = lines.length > 1 ? `${lines.length} lines` : "Note";
  } else if (att.kind === "file" && att.fileSha) {
    sub = meta
      ? `${kindLabel(att.name, meta.type)}, ${sizeText(meta.size)}`
      : meta === null
        ? "No longer stored"
        : kindLabel(att.name, "");
    href = `/api/files/${att.fileSha}?name=${encodeURIComponent(att.name)}`;
  }

  const label = (
    <>
      <span
        aria-hidden="true"
        className="grid size-11 shrink-0 place-items-center rounded-md bg-soft"
      >
        <Icon size={20} />
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="truncate font-medium">{title}</span>
        <span className="truncate text-[13px] text-muted-foreground">{sub}</span>
      </span>
    </>
  );

  return (
    <li className="border-t border-line py-2.5">
      <div className="flex min-h-11 items-center gap-3">
        {att.kind === "note" ? (
          <button
            type="button"
            aria-expanded={editing}
            onClick={() => setEditing(!editing)}
            className="flex min-w-0 flex-1 items-center gap-3 text-left"
          >
            {label}
          </button>
        ) : href ? (
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="flex min-w-0 flex-1 items-center gap-3"
          >
            {label}
          </a>
        ) : (
          <span className="flex min-w-0 flex-1 items-center gap-3">{label}</span>
        )}
        <Button variant="icon" aria-label={`Remove attachment: ${title}`} onClick={onRemove}>
          <CloseIcon size={20} />
        </Button>
      </div>
      {att.kind === "note" && editing && <NoteEditor att={att} />}
    </li>
  );
}

function NoteEditor({ att }: { att: Attachment }) {
  const actions = useStructureActions();
  const [body, setBody] = useState(att.body ?? "");
  const id = useId();
  return (
    <div className="mt-2">
      <label htmlFor={id} className="sr-only">
        Note
      </label>
      <Textarea
        id={id}
        value={body}
        maxLength={20000}
        onChange={(e) => setBody(e.target.value)}
        onBlur={() => void actions.updateNote(att.id, body)}
      />
    </div>
  );
}

function LinkForm({
  onAdd,
  onCancel,
}: {
  onAdd: (url: string, name: string) => void;
  onCancel: () => void;
}) {
  const [url, setUrl] = useState("");
  const [name, setName] = useState("");
  const id = useId();
  return (
    <form
      className="flex flex-col gap-3 border-t border-line py-3"
      onSubmit={(e: FormEvent) => {
        e.preventDefault();
        onAdd(url, name);
      }}
    >
      <div>
        <Label htmlFor={`${id}-url`}>Web address</Label>
        <Input
          id={`${id}-url`}
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          inputMode="url"
          autoFocus
        />
      </div>
      <div>
        <Label htmlFor={`${id}-name`}>Name, if you want one</Label>
        <Input
          id={`${id}-name`}
          value={name}
          maxLength={300}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div className="flex gap-3">
        <Button type="submit" size="compact">
          Add link
        </Button>
        <Button variant="text" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}

function NoteForm({ onAdd, onCancel }: { onAdd: (body: string) => void; onCancel: () => void }) {
  const [body, setBody] = useState("");
  const id = useId();
  return (
    <form
      className="flex flex-col gap-3 border-t border-line py-3"
      onSubmit={(e: FormEvent) => {
        e.preventDefault();
        onAdd(body);
      }}
    >
      <div>
        <Label htmlFor={id}>Note</Label>
        <Textarea
          id={id}
          value={body}
          maxLength={20000}
          onChange={(e) => setBody(e.target.value)}
          autoFocus
        />
      </div>
      <div className="flex gap-3">
        <Button type="submit" size="compact">
          Add note
        </Button>
        <Button variant="text" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
