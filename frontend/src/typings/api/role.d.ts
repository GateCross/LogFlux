declare namespace Api {
    namespace Role {
        interface RoleItem {
            id: number;
            name: string; // 唯一标识，如 "admin"
            displayName: string; // 显示名称，如 "管理员"
            description: string;
            permissions: string[]; // 权限列表
            createdAt: string;
        }

        interface RoleListResp {
            list: RoleItem[];
        }

        interface UpdateRolePermissionsReq {
            id: number;
            permissions: string[];
        }
    }
}
