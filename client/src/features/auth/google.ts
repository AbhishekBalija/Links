// The Google OAuth client ID (public). Google sign-in is hidden while it's
// empty, as on Vercel preview URLs, which Google can't list as origins.
export const googleClientId: string = import.meta.env.VITE_GOOGLE_CLIENT_ID ?? ''
