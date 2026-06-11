import {BaseService} from '../core/base.service'
import type {ListUploadsResponse, Upload, UploadImageResponse} from './types'
import type {InternalAxiosRequestConfig} from 'axios'

export type ImageQuality = 'low' | 'medium' | 'high' | 'origin'

/**
 * 根据上传ID构造文件访问URL
 * @param id - 上传记录ID
 * @param quality - 图片质量
 * @returns 文件访问URL
 */
export function getFileUrl(
  id: string | number | null | undefined,
  quality: ImageQuality = 'origin'
): string | null {
  if (!id) return null
  if (quality === 'origin') return `/f/${id}`
  return `/f/${id}?quality=${quality}`
}

/**
 * 格式化文件大小
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

/**
 * 上传服务
 * 处理文件上传相关的 API 请求
 */
export class UploadService extends BaseService {
  protected static readonly basePath = '/api/v1/upload'

  /**
   * 通用文件上传
   * @param file - 文件对象
   * @param type - 业务分类（如 avatar、attachment、generic）
   * @param metadata - 可选额外 JSON 元数据
   */
  static async uploadFile(
    file: File,
    type: string = 'generic',
    metadata?: Record<string, unknown>
  ): Promise<Upload> {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('type', type)
    if (metadata) {
      formData.append('metadata', JSON.stringify(metadata))
    }

    return this.post<Upload>('', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    } as InternalAxiosRequestConfig)
  }

  /**
   * 获取我的文件列表
   * @param page - 页码（1-based）
   * @param pageSize - 每页数量
   * @param keyword - 搜索关键词（文件名模糊）
   * @param type - 业务分类过滤
   * @param extension - 扩展名过滤
   */
  static async listMyFiles(
    page = 1,
    pageSize = 20,
    keyword?: string,
    type?: string,
    extension?: string
  ): Promise<ListUploadsResponse> {
    const params: Record<string, string | number> = { page, page_size: pageSize }
    if (keyword) params.keyword = keyword
    if (type) params.type = type
    if (extension) params.extension = extension
    return this.get<ListUploadsResponse>('/my', { params })
  }

  /**
   * 删除文件
   */
  static async deleteFile(id: string): Promise<void> {
    return this.delete<void>(`/${id}`)
  }

  /**
   * 获取单文件下载 URL（触发 attachment 下载）
   */
  static getDownloadUrl(id: string): string {
    return `/api/v1/upload/download/${id}`
  }

  /**
   * 批量 ZIP 打包下载
   * @param ids - 文件 ID 数组
   */
  static async batchDownload(ids: string[]): Promise<Blob> {
    const response = await this.post<Blob>('/download/batch', { ids }, {
      responseType: 'blob',
    } as InternalAxiosRequestConfig)
    return response
  }

  /**
   * 将 base64 图片转换为 Blob 并上传（兼容旧接口）
   */
  static async uploadBase64Image(
    base64: string,
    type: string = 'generic',
    filename: string = 'image.png'
  ): Promise<UploadImageResponse> {
    const response = await fetch(base64)
    const blob = await response.blob()
    const mimeType = base64.match(/data:([^;]+);/)?.[1] || 'image/png'
    const file = new File([blob], filename, { type: mimeType })
    const result = await this.uploadFile(file, type)
    return { id: result.id }
  }
}
