/**
 * 管理员服务模块
 *
 * @description
 * 提供系统配置、用户与任务管理功能，包括：
 * - 系统配置管理（创建、查询、更新、删除）
 * - 异步任务配置及下发
 * - 用户账号状态管理
 *
 * @remarks
 * 所有接口都需要管理员权限
 *
 * @example
 * ```typescript
 * import { AdminService } from '@/lib/services';
 *
 * // 获取系统配置列表
 * const configs = await AdminService.listSystemConfigs();
 * ```
 */

export { AdminService } from './admin.service';
export type {
  SystemConfig,
  CreateSystemConfigRequest,
  CreateUserRequest,
  UpdateSystemConfigRequest,
  AuthSource,
  AuthSourceRequest,
  ToggleAuthSourceRequest,
  TaskMeta,
  TaskParam,
  TaskParamType,
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
  Schedule,
  CreateScheduleRequest,
  UpdateScheduleRequest,
} from './types';
