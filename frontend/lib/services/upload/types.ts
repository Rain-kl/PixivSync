/**
 * 文件上传元数据
 */
export interface UploadMetadata {
  width?: number
  height?: number
  duration?: number
  original_mime?: string
  user_agent?: string
  client_ip?: string
  bucket?: string
  extra?: Record<string, unknown>
}

/**
 * 上传记录
 */
export interface Upload {
  id: string
  user_id: string
  file_name: string
  file_path: string
  file_size: number
  mime_type: string
  extension: string
  hash: string
  storage_driver: string
  type: string
  status: string
  metadata: UploadMetadata
  created_at: string
  updated_at: string
}

/**
 * 上传接口响应（原 UploadImageResponse 兼容）
 */
export interface UploadImageResponse {
  /** 上传记录 ID */
  id: string
}

/**
 * 文件列表查询参数
 */
export interface ListUploadsQuery {
  page?: number
  page_size?: number
  type?: string
  extension?: string
  keyword?: string
}

/**
 * 文件列表分页响应
 */
export interface ListUploadsResponse {
  total: number
  page: number
  page_size: number
  items: Upload[]
}