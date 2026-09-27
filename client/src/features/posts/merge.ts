type Dated = { at: string }

export type Source<T extends Dated> = {
  // Loaded so far, newest first.
  items: T[]
  // True once the list has no more pages.
  complete: boolean
}

// mergeNewestFirst interleaves several newest-first lists that load a page at
// a time. A list with more pages could still hold something newer than the
// oldest item already loaded from another list, so anything older than its
// last loaded item waits until that list loads further.
export function mergeNewestFirst<T extends Dated>(sources: Source<T>[]): { items: T[]; hasMore: boolean } {
  let cutoff = -Infinity
  for (const source of sources) {
    const last = source.items[source.items.length - 1]
    if (!source.complete && last) cutoff = Math.max(cutoff, new Date(last.at).getTime())
  }
  const items = sources
    .flatMap((source) => source.items)
    .filter((item) => new Date(item.at).getTime() >= cutoff)
    .sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime())
  return { items, hasMore: sources.some((source) => !source.complete) }
}
