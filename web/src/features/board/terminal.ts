export function isTerminalColumn(name: string): boolean {
  const n = name.toLowerCase().trim();
  return n === "done" || n === "completed" || n === "closed";
}
