import { Check, Copy, Link2 } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Avatar } from '../../../app/shell/Avatar'
import { roleLabel } from '../../../app/shell/nav'
import { Skeleton } from '../../../shared/ui/states'
import { useDepartmentOverview } from '../api'
import type { PublicProfile } from '../types'

type Props = {
  profile: PublicProfile
  // The signed-in member's own profile: it explains what others see and
  // leaves contact details to Edit profile.
  own?: boolean
  // Sits above the name on desktop, such as the own profile's Edit button.
  action?: ReactNode
}

// ProfileView is a member's page: who they are first, then what they wrote,
// with contact details and their Department to the side on desktop.
export function ProfileView({ profile, own = false, action }: Props) {
  const links = profileLinks(profile)
  const firstName = profile.full_name.trim().split(/\s+/)[0]
  const sparse = !own && !profile.bio && links.length === 0 && !profile.email && !profile.phone

  return (
    <div className="grid items-start gap-3.5 lg:max-w-[1120px] lg:grid-cols-[minmax(0,1fr)_340px] lg:gap-7">
      <article className="flex flex-col gap-3.5 lg:gap-[22px] lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-10 lg:pt-[30px] lg:pb-[38px]">
        {action && <div className="hidden items-center justify-between gap-4 lg:flex">{action}</div>}
        <header className="flex flex-col items-center gap-2 rounded-xl border border-line bg-surface px-[18px] py-[22px] text-center lg:flex-row lg:gap-[22px] lg:border-0 lg:p-0 lg:text-left">
          <Avatar name={profile.full_name} className="size-[76px] text-2xl lg:size-[88px] lg:text-[28px]" />
          <div className="flex flex-col gap-1.5">
            <h1 className="mt-1 font-serif text-[28px] font-medium lg:mt-0 lg:text-[40px] lg:leading-[1.05] lg:tracking-[-0.6px]">{profile.full_name}</h1>
            {profile.headline && <p className="text-[15px] text-ink-2 lg:text-base">{profile.headline}</p>}
            <WhoLine profile={profile} />
          </div>
        </header>

        {sparse && <p className="mx-1 text-sm text-ink-3 lg:mx-0">{firstName} hasn't added a bio, links or contact details yet.</p>}

        {profile.bio && (
          <Section title="About">
            <p className="font-serif text-[17px] leading-[1.55] whitespace-pre-line text-prose lg:max-w-[640px] lg:text-lg lg:leading-[1.6]">{profile.bio}</p>
          </Section>
        )}

        {links.length > 0 && (
          <Section title="Links">
            <ul className="flex flex-col">
              {links.map((link) => (
                <li key={link.label}>
                  <a href={link.url} target="_blank" rel="noopener noreferrer" className="flex min-h-11 items-center gap-2.5 text-[15px] font-semibold">
                    <Link2 aria-hidden="true" className="size-4 shrink-0" />
                    <span>{link.label}</span>
                    <span className="truncate text-[13px] font-normal text-ink-3">{displayURL(link.url)}</span>
                  </a>
                </li>
              ))}
            </ul>
          </Section>
        )}
      </article>

      {!sparse && (
        <aside className="flex flex-col gap-3.5 lg:gap-4">
          <ContactPanel profile={profile} own={own} />
          {!own && profile.department && <DepartmentPanel code={profile.department.code} name={profile.department.name} />}
        </aside>
      )}
    </div>
  )
}

// WhoLine reads "HOD · Computer Science" or "Student · Computer Science ·
// batch 2023", with the Department opening its page.
function WhoLine({ profile }: { profile: PublicProfile }) {
  const top = profile.roles?.[0]
  if (!top) return null
  return (
    <p className="text-[13px] text-ink-3 lg:text-sm">
      {roleLabel(top)}
      {profile.department && (
        <>
          {' · '}
          <Link to={`/departments/${profile.department.code}`} className="font-semibold">
            {profile.department.name}
          </Link>
        </>
      )}
      {profile.batch_year !== undefined && ` · batch ${profile.batch_year}`}
    </p>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-1.5 rounded-xl border border-line bg-surface px-[18px] py-4 lg:gap-2 lg:border-0 lg:bg-transparent lg:p-0">
      <h2 className="text-[13px] font-semibold text-ink-3">{title}</h2>
      {children}
    </section>
  )
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3 rounded-xl border border-line bg-surface px-[18px] py-4 lg:p-[22px]">
      <h2 className="text-[13px] font-semibold text-ink-3 lg:font-serif lg:text-xl lg:font-medium lg:text-ink">{title}</h2>
      {children}
    </section>
  )
}

function ContactPanel({ profile, own }: { profile: PublicProfile; own: boolean }) {
  if (own) {
    const shared = [profile.show_email && 'email', profile.show_phone && 'phone'].filter(Boolean)
    return (
      <Panel title="Contact">
        <p className="text-sm leading-normal text-ink-2">
          {shared.length === 0
            ? 'Your email and phone are hidden from others. You can share them in Edit profile.'
            : `Signed-in members can see your ${shared.join(' and ')}. Change this in Edit profile.`}
        </p>
      </Panel>
    )
  }
  return (
    <Panel title="Contact">
      {profile.email && <ContactRow label="Email" value={profile.email} href={`mailto:${profile.email}`} />}
      {profile.phone && <ContactRow label="Phone" value={profile.phone} href={`tel:${profile.phone}`} />}
      {!profile.email && !profile.phone ? (
        <p className="text-sm text-ink-3">Email and phone not shared.</p>
      ) : (
        (!profile.email || !profile.phone) && <p className="text-[13px] text-ink-3">{profile.email ? 'Phone' : 'Email'} not shared.</p>
      )}
    </Panel>
  )
}

// ContactRow shows an email or phone number with a copy button, since most
// people paste it into another app.
function ContactRow({ label, value, href }: { label: string; value: string; href: string }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Some browsers block the clipboard; the value is still there to select.
    }
  }

  return (
    <div className="flex items-center justify-between gap-2">
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="text-xs text-ink-3">{label}</span>
        <a href={href} className="truncate text-[15px] font-semibold">
          {value}
        </a>
      </span>
      <button
        type="button"
        onClick={copy}
        aria-label={copied ? `${label} copied` : `Copy ${label.toLowerCase()}`}
        className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-well text-ink hover:bg-[#e0d7c7]"
      >
        {copied ? <Check aria-hidden="true" className="size-[18px]" /> : <Copy aria-hidden="true" className="size-[18px]" />}
      </button>
      <span role="status" className="sr-only">
        {copied ? `${label} copied` : ''}
      </span>
    </div>
  )
}

function DepartmentPanel({ code, name }: { code: string; name: string }) {
  const overview = useDepartmentOverview(code)
  const counts = overview.data?.counts
  return (
    <Panel title="Department">
      <Link to={`/departments/${code}`} className="text-[15px] font-semibold">
        {name} →
      </Link>
      {overview.isPending ? (
        <Skeleton className="h-3.5 w-40" />
      ) : (
        counts && (
          <span className="text-[13px] text-ink-3">
            {counts.faculty} faculty · {counts.students} students
          </span>
        )
      )}
    </Panel>
  )
}

function profileLinks(profile: PublicProfile) {
  return [
    { label: 'LinkedIn', url: profile.linkedin_url },
    { label: 'GitHub', url: profile.github_url },
    { label: 'Portfolio', url: profile.portfolio_url },
  ].filter((link): link is { label: string; url: string } => Boolean(link.url))
}

// displayURL drops the scheme and "www." so a link reads "github.com/priyak".
function displayURL(url: string) {
  return url.replace(/^https?:\/\//, '').replace(/^www\./, '').replace(/\/$/, '')
}

export function ProfileSkeleton() {
  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-line bg-surface px-5 py-8 lg:max-w-[1120px] lg:flex-row lg:gap-6 lg:px-10">
      <Skeleton className="size-[76px] rounded-full lg:size-[88px]" />
      <div className="flex w-full flex-col items-center gap-2.5 lg:items-start">
        <Skeleton className="h-7 w-48" />
        <Skeleton className="h-4 w-64 max-w-full" />
        <Skeleton className="h-3.5 w-40" />
      </div>
    </div>
  )
}
