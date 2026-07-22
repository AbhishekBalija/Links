import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '../../../components/ui/button'
import { useAuthStore } from '../../auth/store'
import { updateMyProfile } from '../api'
import { ApiRequestError } from '../../../shared/api/types'

export default function EditProfile() {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const refreshUser = useAuthStore((s) => s.refreshUser)

  const p = user?.profile
  const [headline, setHeadline] = useState(p?.headline ?? '')
  const [bio, setBio] = useState(p?.bio ?? '')
  const [linkedinUrl, setLinkedinUrl] = useState(p?.linkedin_url ?? '')
  const [githubUrl, setGithubUrl] = useState(p?.github_url ?? '')
  const [portfolioUrl, setPortfolioUrl] = useState(p?.portfolio_url ?? '')
  const [showEmail, setShowEmail] = useState(p?.show_email ?? false)
  const [showPhone, setShowPhone] = useState(p?.show_phone ?? false)
  const [serverError, setServerError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setServerError(null)
    setLoading(true)

    try {
      await updateMyProfile({
        headline: headline || null,
        bio: bio || null,
        linkedin_url: linkedinUrl || null,
        github_url: githubUrl || null,
        portfolio_url: portfolioUrl || null,
        show_email: showEmail,
        show_phone: showPhone,
      })
      await refreshUser()
      navigate('/', { replace: true })
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setServerError(err.message)
      } else {
        setServerError('Something went wrong. Please try again.')
      }
    } finally {
      setLoading(false)
    }
  }

  const inputClass = 'block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20'

  return (
    <div className="mx-auto max-w-lg px-4 py-8">
      <h1 className="text-xl font-semibold text-foreground mb-1">Edit Profile</h1>
      <p className="text-sm text-muted-foreground mb-6">
        {user?.profile.full_name ?? user?.email}
      </p>

      <form onSubmit={handleSubmit} noValidate className="space-y-4">
        <div>
          <label htmlFor="headline" className="block text-sm font-medium text-foreground mb-1.5">
            Headline
          </label>
          <input id="headline" value={headline} onChange={(e) => setHeadline(e.target.value)} className={inputClass} placeholder="e.g. Computer Science Student" />
        </div>

        <div>
          <label htmlFor="bio" className="block text-sm font-medium text-foreground mb-1.5">
            Bio
          </label>
          <textarea id="bio" value={bio} onChange={(e) => setBio(e.target.value)} rows={4} className={inputClass + ' resize-y min-h-[80px]'} placeholder="A short description about yourself" />
        </div>

        <fieldset className="space-y-3 rounded-md border border-border p-4">
          <legend className="text-sm font-medium text-foreground px-1">Social Links</legend>

          <div>
            <label htmlFor="linkedin" className="block text-sm font-medium text-foreground mb-1.5">LinkedIn</label>
            <input id="linkedin" value={linkedinUrl} onChange={(e) => setLinkedinUrl(e.target.value)} className={inputClass} placeholder="https://linkedin.com/in/..." />
          </div>

          <div>
            <label htmlFor="github" className="block text-sm font-medium text-foreground mb-1.5">GitHub</label>
            <input id="github" value={githubUrl} onChange={(e) => setGithubUrl(e.target.value)} className={inputClass} placeholder="https://github.com/..." />
          </div>

          <div>
            <label htmlFor="portfolio" className="block text-sm font-medium text-foreground mb-1.5">Portfolio</label>
            <input id="portfolio" value={portfolioUrl} onChange={(e) => setPortfolioUrl(e.target.value)} className={inputClass} placeholder="https://..." />
          </div>
        </fieldset>

        <fieldset className="space-y-3 rounded-md border border-border p-4">
          <legend className="text-sm font-medium text-foreground px-1">Privacy</legend>

          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={showEmail} onChange={(e) => setShowEmail(e.target.checked)} className="rounded border-input" />
            <span className="text-sm text-foreground">Show email on public profile</span>
          </label>

          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={showPhone} onChange={(e) => setShowPhone(e.target.checked)} className="rounded border-input" />
            <span className="text-sm text-foreground">Show phone on public profile</span>
          </label>
        </fieldset>

        {serverError && (
          <div className="rounded-md bg-destructive/10 border border-destructive/20 px-3 py-2 text-sm text-destructive">
            {serverError}
          </div>
        )}

        <div className="flex gap-3">
          <Button type="submit" disabled={loading}>
            {loading ? 'Saving…' : 'Save'}
          </Button>
          <Button type="button" variant="outline" onClick={() => navigate('/')}>
            Cancel
          </Button>
        </div>
      </form>
    </div>
  )
}
