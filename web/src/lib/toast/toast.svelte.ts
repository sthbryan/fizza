export type ToastKind = "ok" | "error";

type ToastState = {
  message: string;
  kind: ToastKind;
} | null;

let current = $state<ToastState>(null);
let timer: number | undefined;

export function getToast(): ToastState {
  return current;
}

export function showToast(message: string, kind: ToastKind = "ok") {
  current = { message, kind };
  if (timer !== undefined) window.clearTimeout(timer);
  timer = window.setTimeout(() => {
    if (current?.message === message) current = null;
  }, 3200);
}

export function clearToast() {
  current = null;
}
