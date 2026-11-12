/** Whether the pending top-up count rose since the last poll — the trigger for
 * the "new top-up request" alert. The first observation (prev undefined) only
 * seeds the baseline, so opening the console never alerts on old requests. */
export function isNewTopUp(prev: number | undefined, next: number | undefined): boolean {
  return prev !== undefined && next !== undefined && next > prev
}

/** Document title with a "(N) " pending prefix (N = 0 → plain title). */
export function titleWithCount(base: string, pending: number): string {
  const clean = base.replace(/^\(\d+\)\s*/, '')
  return pending > 0 ? `(${pending}) ${clean}` : clean
}

/** A short two-tone chime via Web Audio (no asset to ship). Best-effort:
 * browsers block audio until the admin has interacted with the page once. */
export function playChime(): void {
  try {
    const Ctx = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!Ctx) return
    const ctx = new Ctx()
    const tone = (freq: number, start: number) => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.type = 'sine'
      osc.frequency.value = freq
      gain.gain.setValueAtTime(0.0001, ctx.currentTime + start)
      gain.gain.exponentialRampToValueAtTime(0.25, ctx.currentTime + start + 0.02)
      gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + start + 0.35)
      osc.connect(gain).connect(ctx.destination)
      osc.start(ctx.currentTime + start)
      osc.stop(ctx.currentTime + start + 0.4)
    }
    tone(880, 0)
    tone(1320, 0.18)
    setTimeout(() => void ctx.close(), 1000)
  } catch {
    /* audio is a nicety — never break the console over it */
  }
}
