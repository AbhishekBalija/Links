import type { ReactNode } from 'react'
import { ChevronLeft } from 'lucide-react'

const sections = [
  { n: '01', title: 'Notices', text: 'Official word from the principal, your HOD and the placement office.' },
  { n: '02', title: 'Events', text: "What's on across campus, and a seat if you want one." },
  { n: '03', title: 'People', text: 'Classmates and staff, by name or department.' },
  { n: '04', title: 'Placement', text: 'Open drives and where your applications stand.' },
]

const today = new Intl.DateTimeFormat('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }).format(new Date())

// SignInLayout frames every sign-in screen: a front page beside the form on
// desktop, the masthead (or a back link) above it on phones.
export function SignInLayout({ children, back }: { children: ReactNode; back?: { label: string; onClick: () => void } }) {
  return (
    <div className="min-h-dvh bg-paper text-ink lg:flex">
      <aside className="hidden w-1/2 max-w-[720px] flex-col gap-7 bg-rail px-16 py-14 lg:flex">
        <p className="font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase">{today}</p>
        <div className="flex flex-col gap-2.5">
          <p className="font-serif text-[128px] leading-[0.9] font-semibold tracking-[-3px]">Links</p>
          <p className="font-serif text-[30px] leading-tight text-ink-2 italic">Your campus, on one page.</p>
        </div>
        <ul className="mt-6 grid max-w-[560px] grid-cols-2 gap-x-10 gap-y-7">
          {sections.map((s) => (
            <li key={s.n} className="flex flex-col gap-1">
              <span className="font-mono text-xs text-rust">{s.n}</span>
              <span className="font-serif text-[22px] font-medium">{s.title}</span>
              <span className="text-sm leading-normal text-ink-2">{s.text}</span>
            </li>
          ))}
        </ul>
        <p className="mt-auto text-[13px] text-ink-3">No passwords to remember. Your college's class list decides who gets in.</p>
      </aside>
      <main className="flex flex-1 flex-col lg:items-center lg:justify-center">
        {back ? (
          <div className="px-3 pt-3.5 lg:hidden">
            <button type="button" onClick={back.onClick} aria-label={`Back to ${back.label.toLowerCase()}`} className="inline-flex min-h-11 items-center gap-1 px-1.5 text-sm font-semibold text-rust">
              <ChevronLeft aria-hidden="true" className="size-4" />
              {back.label}
            </button>
          </div>
        ) : (
          <header className="flex flex-col gap-1 px-6 pt-7 lg:hidden">
            <p className="font-serif text-[44px] leading-none font-semibold tracking-[-1px]">Links</p>
            <p className="font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase">Campus hub</p>
          </header>
        )}
        <div className="flex w-full max-w-[468px] flex-col gap-4 px-6 pt-5 pb-8 lg:px-6 lg:py-12">
          {back && (
            <button type="button" onClick={back.onClick} aria-label={`Back to ${back.label.toLowerCase()}`} className="hidden min-h-11 items-center gap-1 self-start text-sm font-semibold text-rust lg:inline-flex">
              <ChevronLeft aria-hidden="true" className="size-4" />
              {back.label}
            </button>
          )}
          {children}
        </div>
      </main>
    </div>
  )
}

export function SignInHeading({ children, label }: { children: ReactNode; label?: string }) {
  return (
    <div className="flex flex-col gap-2">
      {label && <p className="font-mono text-[11px] tracking-[1.2px] text-ink-3 uppercase">{label}</p>}
      <h1 className="font-serif text-[30px] leading-[1.1] font-medium tracking-[-0.6px] lg:text-4xl">{children}</h1>
    </div>
  )
}
