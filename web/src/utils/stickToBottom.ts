export interface ScrollMetrics {
  scrollTop: number
  scrollHeight: number
  clientHeight: number
}

/** True when the viewport is within `threshold` px of the end of the content. */
export function isNearBottom(m: ScrollMetrics, threshold: number): boolean {
  return m.scrollHeight - m.clientHeight - m.scrollTop <= threshold
}
