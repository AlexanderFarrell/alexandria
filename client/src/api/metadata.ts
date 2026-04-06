import client from './client'
import type { MetadataSearchResponse } from '@/types'

export interface MetadataSearchParams {
  q?: string
  title?: string
  author?: string
  isbn?: string
}

export async function searchMetadata(params: MetadataSearchParams): Promise<MetadataSearchResponse> {
  return client.get('/metadata/search', { params: params as Record<string, string | number | boolean | null | undefined> })
}
