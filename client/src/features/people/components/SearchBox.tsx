import { Search, X } from 'lucide-react'

type Props = { value: string; onChange: (value: string) => void }

// SearchBox is the first thing on People: most visits are looking for one
// person, so it comes before the filters.
export function SearchBox({ value, onChange }: Props) {
  return (
    <label className="flex min-h-[52px] w-full items-center gap-3 rounded-xl border border-[#d6ccbb] bg-surface pr-2 pl-4 text-ink-2 focus-within:border-ink">
      <Search aria-hidden="true" className="size-5 shrink-0" />
      <span className="sr-only">Search people</span>
      <input
        type="search"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="Search by name or what they do"
        autoComplete="off"
        enterKeyHint="search"
        className="min-w-0 flex-1 bg-transparent text-[17px] text-ink outline-none placeholder:text-ink-3 [&::-webkit-search-cancel-button]:hidden"
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange('')}
          aria-label="Clear search"
          className="flex size-11 shrink-0 items-center justify-center rounded-lg text-ink-3 hover:text-ink"
        >
          <X aria-hidden="true" className="size-5" />
        </button>
      )}
    </label>
  )
}
