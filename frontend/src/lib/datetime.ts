// datetime-local inputs need "YYYY-MM-DDTHH:mm" in the viewer's local time.
// Using the Date object's local getters (not the UTC ones) renders the
// browser's local wall-clock view of a given instant — the exact inverse of
// new Date(localString).toISOString().
export function toDatetimeLocalValue(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}
