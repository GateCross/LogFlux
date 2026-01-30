declare namespace Api {
  /**
   * namespace Auth
   *
   * backend api module: "auth"
   */
  namespace Auth {
    interface LoginToken {
      token: string;
      refreshToken: string;
    }

    interface UserInfo {
      userId: string | number;
      username: string;
      roles: string[];
      buttons?: string[];
      preferences?: string;
    }
  }
}
