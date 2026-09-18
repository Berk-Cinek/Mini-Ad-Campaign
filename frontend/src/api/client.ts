import type { ApiErrorBody } from '../types';

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

type RequestOptions = Omit<RequestInit, 'body'> & { body?: unknown };

export async function request<T>(path: string, options?: RequestOptions): Promise<T> {
  const { body, ...rest } = options ?? {};
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...rest,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new ApiError(res.status, body?.error ?? res.statusText);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}
