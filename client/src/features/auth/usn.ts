// The USN's shape only: which Department codes exist comes from the server,
// so a Department an admin adds works without a client change (#18).
const usnFormat = /^4MN\d{2}[A-Z]{2}\d{3}$/i

export function isUSNFormat(usn: string): boolean {
  return usnFormat.test(usn)
}
