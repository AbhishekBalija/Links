type Dated = { at: string }

export type Source<T extends Dated> = {
  // Loaded so far, in the list's order.
  items: T[]
  // True once the list has no more pages.
  complete: boolean
}

// merge interleaves several sorted lists that load a page at a time. A list
// with more pages could still hold something that belongs before the last
// item already loaded from another list, so anything past its last loaded
// item waits until that list loads further. `direction` is 1 for oldest
// first and -1 for newest first.
function merge<T extends Dated>(sources: Source<T>[], direction: 1 | -1): { items: T[]; hasMore: boolean } {
  const time = (item: T) => direction * new Date(item.at).getTime()
  let cutoff = Infinity
  for (const source of sources) {
    const last = source.items[source.items.length - 1]
    if (!source.complete && last) cutoff = Math.min(cutoff, time(last))
  }
  const items = sources
    .flatMap((source) => source.items)
    .filter((item) => time(item) <= cutoff)
    .sort((a, b) => time(a) - time(b))
  return { items, hasMore: sources.some((source) => !source.complete) }
}

// mergeNewestFirst interleaves newest-first lists, such as My posts.
export function mergeNewestFirst<T extends Dated>(sources: Source<T>[]) {
  return merge(sources, -1)
}

// mergeOldestFirst interleaves oldest-first lists, such as the approval queue.
export function mergeOldestFirst<T extends Dated>(sources: Source<T>[]) {
  return merge(sources, 1)
}
