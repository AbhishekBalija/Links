import { describe, expect, it } from 'vitest'
import { serverErrors } from './useEventForm'

describe('serverErrors', () => {
  it("puts the server's field messages on the form's fields, as sentences", () => {
    expect(serverErrors({ starts_at: 'must be in the future', capacity: '45 people are already going', department_id: 'no department has this ID' })).toEqual({
      starts: 'Must be in the future.',
      capacity: '45 people are already going.',
    })
  })
})
