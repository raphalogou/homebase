import { type PointerEvent as ReactPointerEvent, useEffect, useRef, useState } from "react";
import type { Task } from "@/data/types";
import { cn } from "@/lib/utils";
import { GripIcon } from "./icons";
import { TaskRow } from "./task-row";

interface Drag {
  id: string;
  from: number;
  to: number;
  startY: number;
  dy: number;
  rowHeight: number;
}

interface YourThreeProps {
  tasks: Task[];
  onToggle: (id: string) => void;
  onMove: (id: string, to: number) => void;
  /** Desktop shows a grip handle; the phone starts a drag with a long press. */
  grip: boolean;
}

const LONG_PRESS_MS = 400;
const MOVE_TOLERANCE = 8;

// DESIGN.md "Your three": rank numbers, a grip on desktop, the dragged row
// lifted (field fill, shadow, radius 12). The grip also answers the arrow
// keys, and the phone's task sheet has up and down buttons.
export function YourThree({ tasks, onToggle, onMove, grip }: YourThreeProps) {
  const [drag, setDrag] = useState<Drag | null>(null);
  const [announce, setAnnounce] = useState("");
  const dragRef = useRef<Drag | null>(null);
  dragRef.current = drag;
  const press = useRef<{ timer: ReturnType<typeof setTimeout>; x: number; y: number } | null>(null);
  const listRef = useRef<HTMLOListElement>(null);
  // The pointer-up that ends a drag also fires a click; it must not open the task.
  const swallowClick = useRef(false);

  // Once a touch drag is live, stop the page from scrolling under it. This
  // needs a non-passive listener, which React does not offer.
  useEffect(() => {
    const el = listRef.current;
    if (!el) return;
    const block = (e: TouchEvent) => {
      if (dragRef.current) e.preventDefault();
    };
    el.addEventListener("touchmove", block, { passive: false });
    return () => el.removeEventListener("touchmove", block);
  }, []);

  function begin(id: string, index: number, y: number, row: HTMLElement) {
    setDrag({
      id,
      from: index,
      to: index,
      startY: y,
      dy: 0,
      rowHeight: row.getBoundingClientRect().height,
    });
  }

  function moveTo(y: number) {
    const d = dragRef.current;
    if (!d) return;
    const dy = y - d.startY;
    const to = Math.max(0, Math.min(tasks.length - 1, d.from + Math.round(dy / d.rowHeight)));
    setDrag({ ...d, dy, to });
  }

  function end() {
    const d = dragRef.current;
    if (d) swallowClick.current = true;
    setDrag(null);
    if (d && d.to !== d.from) {
      onMove(d.id, d.to);
      setAnnounce(`Moved to position ${d.to + 1}.`);
    }
  }

  function onGripDown(e: ReactPointerEvent<HTMLButtonElement>, id: string, index: number) {
    const row = e.currentTarget.closest("li");
    if (!row) return;
    e.currentTarget.setPointerCapture(e.pointerId);
    begin(id, index, e.clientY, row);
  }

  function onRowDown(e: ReactPointerEvent<HTMLLIElement>, id: string, index: number) {
    swallowClick.current = false;
    if (grip || e.pointerType === "mouse") return;
    const row = e.currentTarget;
    const { clientX: x, clientY: y, pointerId } = e;
    press.current = {
      x,
      y,
      timer: setTimeout(() => {
        press.current = null;
        row.setPointerCapture(pointerId);
        begin(id, index, y, row);
      }, LONG_PRESS_MS),
    };
  }

  function onRowMove(e: ReactPointerEvent) {
    const p = press.current;
    if (p && Math.hypot(e.clientX - p.x, e.clientY - p.y) > MOVE_TOLERANCE) {
      clearTimeout(p.timer);
      press.current = null;
    }
    moveTo(e.clientY);
  }

  function onRowUp() {
    if (press.current) {
      clearTimeout(press.current.timer);
      press.current = null;
    }
    end();
  }

  function shift(index: number): number {
    if (!drag) return 0;
    if (index === drag.from) return drag.dy;
    if (drag.from < drag.to && index > drag.from && index <= drag.to) return -drag.rowHeight;
    if (drag.from > drag.to && index < drag.from && index >= drag.to) return drag.rowHeight;
    return 0;
  }

  return (
    <>
      <ol ref={listRef} className="relative" aria-label="Your three">
        {tasks.map((t, i) => {
          const lifted = drag?.id === t.id;
          return (
            <li
              key={t.id}
              onPointerDown={(e) => onRowDown(e, t.id, i)}
              onPointerMove={onRowMove}
              onPointerUp={onRowUp}
              onPointerCancel={onRowUp}
              onClickCapture={(e) => {
                if (swallowClick.current) {
                  swallowClick.current = false;
                  e.preventDefault();
                  e.stopPropagation();
                }
              }}
              onContextMenu={(e) => {
                if (!grip) e.preventDefault();
              }}
              style={{ transform: `translateY(${shift(i)}px)` }}
              className={cn(
                "relative",
                drag && !lifted && "transition-transform duration-150",
                lifted && "z-10 rounded-[12px] bg-field shadow-lift [&>div]:border-transparent",
              )}
            >
              <TaskRow
                task={t}
                onToggle={() => onToggle(t.id)}
                className={cn(lifted && "px-3")}
                lead={
                  <>
                    {grip && (
                      <button
                        type="button"
                        aria-label={`Drag to reorder: ${t.title}`}
                        aria-describedby="reorder-hint"
                        onPointerDown={(e) => onGripDown(e, t.id, i)}
                        onPointerMove={(e) => moveTo(e.clientY)}
                        onPointerUp={end}
                        onPointerCancel={end}
                        onKeyDown={(e) => {
                          const to =
                            e.key === "ArrowUp" ? i - 1 : e.key === "ArrowDown" ? i + 1 : -1;
                          if (to < 0 || to >= tasks.length) return;
                          e.preventDefault();
                          onMove(t.id, to);
                          setAnnounce(`Moved to position ${to + 1}.`);
                        }}
                        className="-ml-2 grid size-11 shrink-0 cursor-grab touch-none place-items-center rounded-md text-muted-foreground active:cursor-grabbing"
                      >
                        <GripIcon size={20} />
                      </button>
                    )}
                    <span
                      className="w-4 shrink-0 text-center text-sm font-semibold text-muted-foreground"
                      aria-hidden="true"
                    >
                      {i + 1}
                    </span>
                  </>
                }
              />
            </li>
          );
        })}
      </ol>
      <p id="reorder-hint" className="sr-only">
        Use the arrow keys to move up or down.
      </p>
      <p className="sr-only" aria-live="polite">
        {announce}
      </p>
    </>
  );
}
