import {BaseService} from '../core/base.service';

export interface AccessToken {
  id: number;
  user_id: number;
  name: string;
  masked_token: string;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTokenResponse {
  token: string;
  record: AccessToken;
}

/**
 * 用户服务
 * 处理用户个人设置相关的 API 请求
 */
export class UserService extends BaseService {
  protected static readonly basePath = '/api/v1/user';

  /**
   * 获取当前用户的 AccessToken 列表
   */
  static async getAccessTokens(): Promise<AccessToken[]> {
    return this.get<AccessToken[]>('/access-tokens');
  }

  /**
   * 创建一个新的 AccessToken
   * @param name - 令牌名称
   */
  static async createAccessToken(name: string): Promise<CreateTokenResponse> {
    return this.post<CreateTokenResponse>('/access-tokens', { name });
  }

  /**
   * 删除一个 AccessToken
   * @param id - 令牌 ID
   */
  static async deleteAccessToken(id: number): Promise<string> {
    return this.delete<string>(`/access-tokens/${id}`);
  }

  /**
   * 轮换一个 AccessToken 密钥
   * @param id - 令牌 ID
   */
  static async rotateAccessToken(id: number): Promise<CreateTokenResponse> {
    return this.post<CreateTokenResponse>(`/access-tokens/${id}/rotate`);
  }
}
