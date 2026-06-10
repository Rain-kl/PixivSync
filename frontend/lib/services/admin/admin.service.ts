import {BaseService} from '@/lib/services';
import type {
  AdminUser,
  AuthSource,
  AuthSourceRequest,
  CreateScheduleRequest,
  CreateSystemConfigRequest,
  CreateTemplateRequest,
  CreateUserRequest,
  DatabaseInfo,
  DispatchTaskRequest,
  ListTaskExecutionsRequest,
  ListTaskExecutionsResponse,
  ListUsersRequest,
  ListUsersResponse,
  Schedule,
  SystemConfig,
  SystemStatus,
  TaskExecution,
  TaskMeta,
  TaskTypeResponse,
  Template,
  ToggleAuthSourceRequest,
  UpdateScheduleRequest,
  UpdateSystemConfigRequest,
  UpdateTemplateRequest,
  UpdateUserStatusRequest,
} from './types';

export type { AdminUser } from './types';

/**
 * 管理员服务
 *
 * @remarks
 * 所有接口都需要管理员权限
 */
export class AdminService extends BaseService {
  protected static readonly basePath = '/api/v1/admin';

  // ==================== 系统配置管理 ====================

  /**
   * 创建系统配置
   * @param request - 创建系统配置的请求参数
   * @returns void
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   * @throws {ValidationError} 当参数验证失败或配置键已存在时
   *
   * @example
   * ```typescript
   * await AdminService.createSystemConfig({
   *   key: 'app.version',
   *   value: '1.0.0',
   *   description: '应用版本号'
   * });
   * ```
   */
  static async createSystemConfig(
    request: CreateSystemConfigRequest,
  ): Promise<void> {
    return this.post<void>('/system-configs', request);
  }

  /**
   * 获取系统配置列表
   * @returns 系统配置列表
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   *
   * @example
   * ```typescript
   * const configs = await AdminService.listSystemConfigs();
   * console.log('系统配置数量:', configs.length);
   * ```
   */
  static async listSystemConfigs(type?: 'system' | 'business'): Promise<SystemConfig[]> {
    const query = type ? `?type=${type}` : '';
    return this.get<SystemConfig[]>(`/system-configs${query}`);
  }

  /**
   * 获取单个系统配置
   * @param key - 配置键
   * @returns 系统配置信息
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   * @throws {NotFoundError} 当配置不存在时
   *
   * @example
   * ```typescript
   * const config = await AdminService.getSystemConfig('app.version');
   * console.log('应用版本:', config.value);
   * ```
   */
  static async getSystemConfig(key: string): Promise<SystemConfig> {
    return this.get<SystemConfig>(`/system-configs/${ key }`);
  }

  /**
   * 更新系统配置
   * @param key - 配置键
   * @param request - 更新系统配置的请求参数
   * @returns void
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   * @throws {NotFoundError} 当配置不存在时
   * @throws {ValidationError} 当参数验证失败时
   *
   * @example
   * ```typescript
   * await AdminService.updateSystemConfig('app.version', {
   *   value: '1.1.0',
   *   description: '更新到新版本'
   * });
   * ```
   */
  static async updateSystemConfig(
    key: string,
    request: UpdateSystemConfigRequest,
  ): Promise<void> {
    return this.put<void>(`/system-configs/${ key }`, request);
  }

  static async testSMTP(request: {
    smtp_host: string;
    smtp_port: number;
    smtp_username: string;
    smtp_password: string;
    to: string;
  }): Promise<{ success: boolean; log: string; error: string }> {
    return this.post<{ success: boolean; log: string; error: string }>('/system-configs/smtp/test', request);
  }



  // ==================== 认证源管理 ====================

  static async listAuthSources(): Promise<AuthSource[]> {
    return this.get<AuthSource[]>('/auth-sources');
  }

  static async createAuthSource(request: AuthSourceRequest): Promise<AuthSource> {
    return this.post<AuthSource>('/auth-sources', request);
  }

  static async updateAuthSource(id: string, request: AuthSourceRequest): Promise<AuthSource> {
    return this.put<AuthSource>(`/auth-sources/${ id }`, request);
  }

  static async toggleAuthSource(id: string, request: ToggleAuthSourceRequest): Promise<void> {
    return this.put<void>(`/auth-sources/${ id }/toggle`, request);
  }

  static async deleteAuthSource(id: string): Promise<void> {
    return this.delete<void>(`/auth-sources/${ id }`);
  }



  // ==================== 任务管理 ====================

  /**
   * 获取支持的任务类型列表
   * @returns 任务类型列表
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   *
   * @example
   * ```typescript
   * const taskTypes = await AdminService.getTaskTypes();
   * console.log('可用任务类型:', taskTypes);
   * ```
   */
  static async getTaskTypes(): Promise<TaskMeta[]> {
    const response = await this.get<TaskTypeResponse[]>('/tasks/types');
    // Adapt backend PascalCase to frontend snake_case
    return response.map(item => ({
      type: item.Type || item.type || '',
      asynq_task: item.AsynqTask || item.asynq_task || '',
      name: item.Name || item.name || '',
      description: item.Description || item.description || '',
      supports_time: item.SupportsTime ?? item.supports_time ?? false,
      max_retry: item.MaxRetry ?? item.max_retry ?? 0,
      queue: item.Queue || item.queue || '',
      params: (item.Params || item.params || []).map(p => ({
        name: p.Name || p.name || '',
        label: p.Label || p.label || '',
        type: p.Type || p.type || '',
        required: p.Required ?? p.required ?? false,
        placeholder: p.Placeholder || p.placeholder || '',
        description: p.Description || p.description || '',
      })),
    }));
  }

  /**
   * 下发任务
   * @param request - 下发任务请求参数
   * @returns void
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   * @throws {ValidationError} 当参数验证失败时
   * @remarks
   * - 不同任务类型需要不同的参数
   * - order_sync 支持 start_time 和 end_time 参数
   * - user_gamification 需要 user_id 参数
   * - 其他任务无需额外参数
   */
  static async dispatchTask(request: DispatchTaskRequest): Promise<string> {
    return this.post<string>('/tasks/dispatch', request);
  }

  /**
   * 查询任务执行记录列表
   */
  static async listTaskExecutions(
    request: ListTaskExecutionsRequest = {},
  ): Promise<ListTaskExecutionsResponse> {
    return this.get<ListTaskExecutionsResponse>(
      '/tasks/executions',
      request as unknown as Record<string, unknown>,
    );
  }

  /**
   * 查询任务执行详情
   */
  static async getTaskExecution(id: string): Promise<TaskExecution> {
    return this.get<TaskExecution>(`/tasks/executions/${ id }`);
  }

  /**
   * 重试失败任务
   */
  static async retryTaskExecution(id: string): Promise<string> {
    return this.post<string>(`/tasks/executions/${ id }/retry`);
  }

  // ==================== 定时任务管理 ====================

  /**
   * 获取定时任务列表
   */
  static async listSchedules(): Promise<Schedule[]> {
    return this.get<Schedule[]>('/tasks/schedules');
  }

  /**
   * 创建定时任务
   */
  static async createSchedule(request: CreateScheduleRequest): Promise<Schedule> {
    return this.post<Schedule>('/tasks/schedules', request);
  }

  /**
   * 更新定时任务
   */
  static async updateSchedule(id: string, request: UpdateScheduleRequest): Promise<Schedule> {
    return this.put<Schedule>(`/tasks/schedules/${ id }`, request);
  }

  /**
   * 删除定时任务
   */
  static async deleteSchedule(id: string): Promise<void> {
    return this.delete<void>(`/tasks/schedules/${ id }`);
  }

  // ==================== 用户管理 ====================

  /**
   * 获取用户列表
   * @param request - 查询参数
   * @returns 用户列表及总数
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   * @throws {ValidationError} 当参数验证失败时
   *
   * @example
   * ```typescript
   * const result = await AdminService.listUsers({
   *   page: 1,
   *   page_size: 20,
   *   user_id: '10001',
   *   username: 'test'
   * });
   * console.log('用户总数:', result.total);
   * console.log('用户列表:', result.users);
   * ```
   *
   * @remarks
   * - page 从 1 开始
   * - page_size 范围 1-100
   * - user_id 按用户 ID 精确搜索
   * - username 按用户名做前缀搜索
   */
  static async listUsers(request: ListUsersRequest): Promise<ListUsersResponse> {
    return this.get<ListUsersResponse>('/users', request as unknown as Record<string, unknown>);
  }

  /**
   * 获取用户详情
   * @param id - 用户 ID
   * @returns 用户完整资料
   */
  static async getUser(id: string): Promise<AdminUser> {
    return this.get<AdminUser>(`/users/${ id }`);
  }

  /**
   * 更新用户状态
   * @param id - 用户 ID
   * @param request - 更新状态请求参数
   * @returns void
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限或禁用管理员用户时
   * @throws {NotFoundError} 当用户不存在时
   *
   * @example
   * ```typescript
   * // 禁用用户
   * await AdminService.updateUserStatus(123, { is_active: false });
   *
   * // 启用用户
   * await AdminService.updateUserStatus(123, { is_active: true });
   * ```
   *
   * @remarks
   * - 不能禁用管理员用户
   */
  static async updateUserStatus(
    id: string,
    request: UpdateUserStatusRequest
  ): Promise<void> {
    return this.put<void>(`/users/${ id }/status`, request);
  }

  /**
   * 创建用户
   * @param request - 创建用户请求参数
   * @returns 创建成功的用户信息
   */
  static async createUser(request: CreateUserRequest): Promise<AdminUser> {
    return this.post<AdminUser>('/users', request);
  }

  /**
   * 删除用户
   * @param id - 用户 ID
   */
  static async deleteUser(id: string): Promise<void> {
    return this.delete<void>(`/users/${ id }`);
  }

  /**
   * 获取系统状态
   * @returns 系统状态指标数据
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   */
  static async getSystemStatus(): Promise<SystemStatus> {
    return this.get<SystemStatus>('/status');
  }

  /**
   * 获取数据库信息
   * @returns 数据库类型、名称、版本信息
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   */
  static async getDatabaseInfo(): Promise<DatabaseInfo> {
    return this.get<DatabaseInfo>('/db-info');
  }

  /**
   * 导出数据库
   * SQLite 返回 .db 二进制，PostgreSQL 返回 pg_dump .sql 文本
   * @returns 包含文件 Blob 和建议文件名的对象
   * @throws {UnauthorizedError} 当未登录时
   * @throws {ForbiddenError} 当无管理员权限时
   */
  static async exportDatabase(): Promise<{ blob: Blob; filename: string }> {
    const response = await import('axios').then(({ default: axios }) =>
      axios.get<Blob>(this.getFullPath('/db-export'), {
        withCredentials: true,
        responseType: 'blob',
      })
    );
    // 从 Content-Disposition 提取文件名
    const disposition = response.headers['content-disposition'] as string | undefined;
    let filename = 'wavelet_export';
    if (disposition) {
      const match = disposition.match(/filename="?([^";]+)"?/);
      if (match) filename = match[1];
    }
    return { blob: response.data, filename };
  }

  // ==================== 系统日志 ====================

  /**
   * 获取系统历史日志
   * @param cursor - 日志游标，0=获取最新，>0=获取更早
   * @param limit - 每页条数，默认 200
   */
  static async getLogs(cursor: number = 0, limit: number = 200): Promise<{
    lines: Array<{ index: number; data: string }>;
    has_more: boolean;
    next_cursor: number;
  }> {
    return this.get('/logs', { cursor, limit });
  }

  /**
   * 获取 ClickHouse 访问日志列表
   */
  static async getAccessLogs(params: {
    page: number;
    page_size: number;
    username?: string;
    path?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<{
    total: number;
    list: Array<{
      id: string;
      user_id: string;
      username: string;
      nickname: string;
      path: string;
      method: string;
      ip: string;
      user_agent: string;
      headers: string;
      status: number;
      latency: number;
      created_at: string;
    }>;
  }> {
    return this.get('/logs/access', params as Record<string, unknown>);
  }

  /**
   * 获取 ClickHouse 访问日志图表聚合指标
   */
  static async getLogsAnalytics(): Promise<{
    trend: Array<{ date: string; count: number }>;
    browsers: Array<{ browser: string; count: number }>;
    top_users: Array<{
      user_id: string;
      username: string;
      nickname: string;
      count: number;
    }>;
  }> {
    return this.get('/logs/analytics');
  }

  // ==================== 模板管理 ====================

  /**
   * 获取模板列表
   */
  static async listTemplates(): Promise<Template[]> {
    return this.get<Template[]>('/templates');
  }

  /**
   * 获取单个模板
   */
  static async getTemplate(key: string): Promise<Template> {
    return this.get<Template>(`/templates/${ key }`);
  }

  /**
   * 创建模板
   */
  static async createTemplate(request: CreateTemplateRequest): Promise<Template> {
    return this.post<Template>('/templates', request);
  }

  /**
   * 更新模板
   */
  static async updateTemplate(key: string, request: UpdateTemplateRequest): Promise<Template> {
    return this.put<Template>(`/templates/${ key }`, request);
  }

  /**
   * 删除模板
   */
  static async deleteTemplate(key: string): Promise<void> {
    return this.delete<void>(`/templates/${ key }`);
  }
}
