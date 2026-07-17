export type StatusKind = "ok" | "error";

type StatusState = {
  message: string;
  kind: StatusKind;
} | null;

let current = $state<StatusState>(null);
let timer: number | undefined;

export function getStatus(): StatusState {
  return current;
}

export function showStatus(message: string, kind: StatusKind = "ok") {
  current = { message, kind };
  if (timer !== undefined) window.clearTimeout(timer);
  timer = window.setTimeout(() => {
    if (current?.message === message) current = null;
  }, 3200);
}

export function clearStatus() {
  current = null;
}
