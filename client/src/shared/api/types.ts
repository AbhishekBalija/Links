export type ApiSuccess<T, M = Record<string, unknown>> = {
  data: T
  meta?: M
}

export type ApiErrorPayload = {
  code: string
  message: string
  details?: Record<string, unknown>
}

export type ApiError = {
  error: ApiErrorPayload
}

export class ApiRequestError extends Error {
  code: string
  status: number
  details?: Record<string, unknown>

  constructor(status: number, payload: ApiErrorPayload) {
    super(payload.message)
    this.name = 'ApiRequestError'
    this.code = payload.code
    this.status = status
    this.details = payload.details
  }
}
