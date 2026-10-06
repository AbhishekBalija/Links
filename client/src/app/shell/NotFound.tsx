import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { buttonStyles } from '../../features/announcements/buttons'

// NotFound is an address LINKS doesn't have (#212). It used to quietly land
// on Home, which hid a mistyped or old link; now it says so, with one way
// back. Posts that were taken down have their own "isn't available" pages.
export default function NotFound() {
  return (
    <section
      aria-labelledby="not-found-h"
      className="flex flex-col gap-2.5 rounded-xl border border-line bg-surface px-5 py-[22px] lg:mt-[72px] lg:w-[560px] lg:gap-3 lg:px-9 lg:py-8"
    >
      <p className="hidden font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase lg:block">Not found</p>
      <h1 id="not-found-h" className="font-serif text-[26px] leading-[1.15] font-medium lg:text-4xl lg:leading-[1.1] lg:tracking-[-0.6px]">
        This page isn't here
      </h1>
      <p className="text-[15px] leading-normal text-ink-2 lg:text-base">
        The link may be mistyped or old. Posts that were taken down say so on their own page.
      </p>
      <Link to="/" className={cn(buttonStyles.primary, 'mt-1.5 w-full hover:text-paper lg:mt-2 lg:w-auto lg:self-start')}>
        Go to Home
      </Link>
    </section>
  )
}
