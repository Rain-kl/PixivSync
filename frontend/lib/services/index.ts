/**
 * 服务层统一入口
 * 提供所有业务服务的访问接口
 *
 * @example
 * ```typescript
 * // 推荐：使用统一的 services 对象
 * import services from '@/lib/services';
 *
 * const user = await services.auth.getUserInfo();
 * const systemConfig = await services.admin.listSystemConfigs();
 * ```
 *
 * @example
 * ```typescript
 * // 按需导入：直接导入特定服务
 * import { AuthService } from '@/lib/services';
 *
 * const user = await AuthService.getUserInfo();
 * ```
 */

import {AuthService} from './auth';
import {AdminService} from './admin';
import {UserService} from './user';
import {ConfigService} from './config';
import {UploadService} from './upload';

/**
 * 服务对象
 * 集中导出所有业务服务
 *
 * @description
 * 推荐使用此对象访问所有服务，保持代码风格统一
 */
const services = {
  /** 认证服务 */
  auth: AuthService,
  /** 管理员服务 */
  admin: AdminService,
  /** 用户服务 */
  user: UserService,
  /** 配置服务 */
  config: ConfigService,
  /** 上传服务 */
  upload: UploadService,
} as const;

export default services;

// ==================== 核心模块导出 ====================

export {
  apiClient,
  BaseService,
  apiConfig,
  cancelRequest,
  cancelAllRequests,
} from './core';

export {
  ApiErrorBase,
  NetworkError,
  TimeoutError,
  UnauthorizedError,
  ForbiddenError,
  NotFoundError,
  ServerError,
  ValidationError,
  isCancelError,
} from './core';

export type {
  ApiResponse,
  ApiError,
  PaginationParams,
  PaginationResponse,
  RequestConfig,
} from './core';

// ==================== 业务服务导出 ====================

// 认证服务
export { AuthService } from './auth';
export type { User, OAuthLoginUrlResponse, OAuthCallbackRequest, AuthSource, ExternalAccountBinding, ChangePasswordRequest, UpdateProfileRequest } from './auth';

// 配置服务
export { ConfigService } from './config';
export type { PublicConfigResponse } from './config';

// 管理员服务
export { AdminService } from './admin';
export type {
  SystemConfig,
  CreateSystemConfigRequest,
  CreateUserRequest,
  UpdateSystemConfigRequest,
  TaskMeta,
  TaskExecution,
  TaskExecutionStatus,
  ListTaskExecutionsRequest,
  ListTaskExecutionsResponse,
  DispatchTaskRequest,
  AdminUser,
  ListUsersRequest,
  ListUsersResponse,
  UpdateUserStatusRequest,
  SystemStatus,
  DatabaseInfo,
} from './admin';

// 用户服务
export { UserService } from './user';
export type { AccessToken, CreateTokenResponse } from './user';

// 上传服务
export { UploadService } from './upload';
export type { UploadImageResponse } from './upload';
