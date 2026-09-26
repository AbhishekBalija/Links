import { CategoryTag } from '../../../notices/components/CategoryTag'
import type { Draft } from '../../types'

// PreviewArticle shows the notice the way readers will see it.
export function PreviewArticle({ draft, audience }: { draft: Draft; audience: string }) {
  const paragraphs = draft.body.trim() ? draft.body.split(/\n\s*\n/) : []
  return (
    <section aria-label="Preview" className="flex flex-col gap-2">
      <p className="px-1 text-xs text-ink-3">This is how {audience} will see it.</p>
      <article className="flex flex-col gap-4 rounded-xl border border-line bg-surface px-5 py-5 lg:px-10 lg:py-9">
        <div className="flex items-center gap-2.5 text-xs">
          <CategoryTag category={draft.category} className="lg:text-xs" />
          <span className="text-ink-3">Posted when it goes live</span>
        </div>
        <h2 className="font-serif text-[26px] leading-tight font-medium tracking-[-0.4px] lg:text-[36px]">
          {draft.title.trim() || <span className="text-ink-3">Your title</span>}
        </h2>
        <div className="flex max-w-[680px] flex-col gap-3 font-serif text-[17px] leading-relaxed text-prose lg:text-[19px]">
          {paragraphs.length === 0 ? (
            <p className="text-ink-3">Your notice will appear here.</p>
          ) : (
            paragraphs.map((text, i) => (
              <p key={i} className="whitespace-pre-line">
                {text}
              </p>
            ))
          )}
        </div>
      </article>
    </section>
  )
}
