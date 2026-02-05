const local: App.I18n.Schema = {
  system: {
    title: 'LogFlux 管理系统',
    updateTitle: '系统版本更新通知',
    updateContent: '检测到系统有新版本发布，是否立即刷新页面？',
    updateConfirm: '立即刷新',
    updateCancel: '稍后再说'
  },
  common: {
    action: '操作',
    add: '新增',
    addSuccess: '添加成功',
    addFailed: '添加失败',
    backToHome: '返回首页',
    batchDelete: '批量删除',
    cancel: '取消',
    close: '关闭',
    check: '勾选',
    selectAll: '全选',
    expandColumn: '展开列',
    columnSetting: '列设置',
    config: '配置',
    confirm: '确认',
    delete: '删除',
    deleteSuccess: '删除成功',
    deleteFailed: '删除失败',
    confirmDelete: '确认删除吗？',
    edit: '编辑',
    warning: '警告',
    error: '错误',
    index: '序号',
    keywordSearch: '请输入关键词搜索',
    logout: '退出登录',
    logoutConfirm: '确认退出登录吗？',
    changePassword: '修改密码',
    oldPassword: '旧密码',
    newPassword: '新密码',
    confirmPassword: '确认密码',
    passwordNoMatch: '两次输入的密码不一致',
    changePasswordSuccess: '修改成功，请重新登录',
    lookForward: '敬请期待',
    modify: '修改',
    modifySuccess: '修改成功',
    noData: '无数据',
    operate: '操作',
    pleaseCheckValue: '请检查输入的值是否合法',
    refresh: '刷新',
    reset: '重置',
    search: '搜索',
    switch: '切换',
    tip: '提示',
    trigger: '触发',
    update: '更新',
    updateSuccess: '更新成功',
    updateFailed: '更新失败',
    userCenter: '个人中心',
    yesOrNo: {
      yes: '是',
      no: '否'
    }
  },
  request: {
    logout: '请求失败后登出用户',
    logoutMsg: '用户状态失效，请重新登录',
    logoutWithModal: '请求失败后弹出模态框再登出用户',
    logoutWithModalMsg: '用户状态失效，请重新登录',
    refreshToken: '请求的token已过期，刷新token',
    tokenExpired: 'token已过期'
  },
  theme: {
    themeDrawerTitle: '主题配置',
    tabs: {
      appearance: '外观',
      layout: '布局',
      general: '通用',
      preset: '预设'
    },
    appearance: {
      themeSchema: {
        title: '主题模式',
        light: '亮色模式',
        dark: '暗黑模式',
        auto: '跟随系统'
      },
      grayscale: '灰色模式',
      colourWeakness: '色弱模式',
      themeColor: {
        title: '主题颜色',
        primary: '主色',
        info: '信息色',
        success: '成功色',
        warning: '警告色',
        error: '错误色',
        followPrimary: '跟随主色'
      },
      themeRadius: {
        title: '主题圆角'
      },
      recommendColor: '应用推荐算法的颜色',
      recommendColorDesc: '推荐颜色的算法参照',
      preset: {
        title: '主题预设',
        apply: '应用',
        applySuccess: '预设应用成功',
        default: {
          name: '默认预设',
          desc: 'LogFlux 默认主题预设'
        },
        dark: {
          name: '暗色预设',
          desc: '适用于夜间使用的暗色主题预设'
        },
        compact: {
          name: '紧凑型',
          desc: '适用于小屏幕的紧凑布局预设'
        },
        azir: {
          name: 'Azir的预设',
          desc: '是 Azir 比较喜欢的莫兰迪色系冷淡风'
        }
      }
    },
    layout: {
      layoutMode: {
        title: '布局模式',
        vertical: '左侧菜单模式',
        'vertical-mix': '左侧菜单混合模式',
        'vertical-hybrid-header-first': '左侧混合-顶部优先',
        horizontal: '顶部菜单模式',
        'top-hybrid-sidebar-first': '顶部混合-侧边优先',
        'top-hybrid-header-first': '顶部混合-顶部优先',
        vertical_detail: '左侧菜单布局，菜单在左，内容在右。',
        'vertical-mix_detail': '左侧双菜单布局，一级菜单在左侧深色区域，二级菜单在左侧浅色区域。',
        'vertical-hybrid-header-first_detail':
          '左侧混合布局，一级菜单在顶部，二级菜单在左侧深色区域，三级菜单在左侧浅色区域。',
        horizontal_detail: '顶部菜单布局，菜单在顶部，内容在下方。',
        'top-hybrid-sidebar-first_detail': '顶部混合布局，一级菜单在左侧，二级菜单在顶部。',
        'top-hybrid-header-first_detail': '顶部混合布局，一级菜单在顶部，二级菜单在左侧。'
      },
      tab: {
        title: '标签栏设置',
        visible: '显示标签栏',
        cache: '标签栏信息缓存',
        cacheTip: '一键开启/关闭全局 keepalive',
        height: '标签栏高度',
        mode: {
          title: '标签栏风格',
          slider: '滑块风格',
          chrome: '谷歌风格',
          button: '按钮风格'
        },
        closeByMiddleClick: '鼠标中键关闭标签页',
        closeByMiddleClickTip: '启用后可以使用鼠标中键点击标签页进行关闭'
      },
      header: {
        title: '头部设置',
        height: '头部高度',
        breadcrumb: {
          visible: '显示面包屑',
          showIcon: '显示面包屑图标'
        }
      },
      sider: {
        title: '侧边栏设置',
        inverted: '深色侧边栏',
        width: '侧边栏宽度',
        collapsedWidth: '侧边栏折叠宽度',
        mixWidth: '混合布局侧边栏宽度',
        mixCollapsedWidth: '混合布局侧边栏折叠宽度',
        mixChildMenuWidth: '混合布局子菜单宽度',
        autoSelectFirstMenu: '自动选择第一个子菜单',
        autoSelectFirstMenuTip: '点击一级菜单时，自动选择并导航到第一个子菜单的最深层级'
      },
      footer: {
        title: '底部设置',
        visible: '显示底部',
        fixed: '固定底部',
        height: '底部高度',
        right: '底部居右'
      },
      content: {
        title: '内容区域设置',
        scrollMode: {
          title: '滚动模式',
          tip: '主题滚动仅 main 部分滚动，外层滚动可携带头部底部一起滚动',
          wrapper: '外层滚动',
          content: '主体滚动'
        },
        page: {
          animate: '页面切换动画',
          mode: {
            title: '页面切换动画类型',
            'fade-slide': '滑动',
            fade: '淡入淡出',
            'fade-bottom': '底部消退',
            'fade-scale': '缩放消退',
            'zoom-fade': '渐变',
            'zoom-out': '闪现',
            none: '无'
          }
        },
        fixedHeaderAndTab: '固定头部和标签栏'
      }
    },
    general: {
      title: '通用设置',
      watermark: {
        title: '水印设置',
        visible: '显示全屏水印',
        text: '自定义水印文本',
        enableUserName: '启用用户名水印',
        enableTime: '显示当前时间',
        timeFormat: '时间格式'
      },
      multilingual: {
        title: '多语言设置',
        visible: '显示多语言按钮'
      },
      globalSearch: {
        title: '全局搜索设置',
        visible: '显示全局搜索按钮'
      }
    },
    configOperation: {
      copyConfig: '复制配置',
      copySuccessMsg: '复制成功，请替换 src/theme/settings.ts 中的变量 themeSettings',
      resetConfig: '重置配置',
      resetSuccessMsg: '重置成功'
    }
  },
  route: {
    home: '首页',
    login: '登录',
    403: '无权限',
    404: '页面不存在',
    500: '服务器错误',
    'iframe-page': '外链页面',

    cron: '定时任务',
    dashboard: '仪表盘',
    caddy: 'Caddy管理',
    caddy_config: 'Caddy配置',
    caddy_log: 'Caddy日志',
    caddy_source: '日志源管理',
    manage: '系统管理',
    manage_user: '用户管理',
    manage_role: '角色管理',
    manage_menu: '菜单管理',
    notification: '通知管理',
    notification_channel: '渠道管理',
    notification_rule: '规则管理',
    notification_template: '模板管理',
    notification_log: '发送记录',
    user: '个人中心',
    user_center: '个人中心'
  },
  page: {
    login: {
      common: {
        loginOrRegister: '登录 / 注册',
        userNamePlaceholder: '请输入用户名',
        phonePlaceholder: '请输入手机号',
        codePlaceholder: '请输入验证码',
        passwordPlaceholder: '请输入密码',
        confirmPasswordPlaceholder: '请再次输入密码',
        codeLogin: '验证码登录',
        confirm: '确定',
        back: '返回',
        validateSuccess: '验证成功',
        loginSuccess: '登录成功',
        welcomeBack: '欢迎回来，{userName} ！'
      },
      pwdLogin: {
        title: '密码登录',
        rememberMe: '记住我',
        forgetPassword: '忘记密码？',
        register: '注册账号',
        otherAccountLogin: '其他账号登录',
        otherLoginMode: '其他登录方式',
        superAdmin: '超级管理员',
        admin: '管理员',
        user: '普通用户'
      },
      codeLogin: {
        title: '验证码登录',
        getCode: '获取验证码',
        reGetCode: '{time}秒后重新获取',
        sendCodeSuccess: '验证码发送成功',
        imageCodePlaceholder: '请输入图片验证码'
      },
      register: {
        title: '注册账号',
        agreement: '我已经仔细阅读并接受',
        protocol: '《用户协议》',
        policy: '《隐私权政策》'
      },
      resetPwd: {
        title: '重置密码'
      },
      bindWeChat: {
        title: '绑定微信'
      }
    },
    home: {
      branchDesc:
        '为了方便大家开发和更新合并，我们对main分支的代码进行了精简，只保留了首页菜单，其余内容已移至example分支进行维护。预览地址显示的内容即为example分支的内容。',
      greeting: '早安，{userName}, 今天又是充满活力的一天!',
      weatherDesc: '今日多云转晴，20℃ - 25℃!',
      projectCount: '项目数',
      todo: '待办',
      message: '消息',
      downloadCount: '下载量',
      registerCount: '注册量',
      schedule: '作息安排',
      study: '学习',
      work: '工作',
      rest: '休息',
      entertainment: '娱乐',
      visitCount: '访问量',
      turnover: '成交额',
      dealCount: '成交量',
      projectNews: {
        title: '项目动态',
        moreNews: '更多动态',
        desc1: 'LogFlux 在2026年1月创建了日志流量分析管理系统！',
        desc2: '团队成员向 LogFlux 提交了新的功能模块。',
        desc3: 'LogFlux 准备为系统发布做充分的准备工作！',
        desc4: 'LogFlux 正在忙于为系统写项目说明文档！',
        desc5: 'LogFlux 刚才把工作台页面优化完成了！'
      },
      creativity: '创意'
    },
    notification: {
      channel: {
        title: '通知渠道',
        add: '新增渠道',
        edit: '编辑渠道',
        name: '名称',
        type: '类型',
        status: '状态',
        config: '配置',
        events: '事件',
        description: '描述',
        enabled: '启用',
        disabled: '禁用',
        test: '测试',
        delete: '删除',
        deleteConfirmTitle: '确认删除',
        deleteConfirmContent: '确认删除渠道 "{name}" 吗?',
        testSuccess: '测试通知已发送',
        testFailed: '测试失败',
        placeholder: {
          name: '渠道名称',
          type: '选择类型',
          config: 'JSON 配置 (例如: { "webhook_url": "..." })',
          events: '["*"] 或 ["error", "caddy"]',
          description: '描述'
        }
      },
      rule: {
        title: '通知规则',
        add: '新增规则',
        edit: '编辑规则',
        name: '名称',
        ruleType: '规则类型',
        eventType: '事件类型',
        status: '状态',
        condition: '条件',
        channels: '通知渠道',
        template: '模板',
        silence: '静默时间 (秒)',
        description: '描述',
        enabled: '启用',
        disabled: '禁用',
        deleteConfirmTitle: '确认删除',
        deleteConfirmContent: '确认删除规则 "{name}" 吗?',
        placeholder: {
          name: '规则名称',
          type: '选择类型',
          eventType: '事件类型 (例如: error)',
          condition: 'JSON 条件 (例如: { "level": "error" })',
          channels: '选择渠道',
          template: '选择模板 (可选)',
          silence: '0',
          description: '描述'
        },
        types: {
          threshold: '阈值',
          frequency: '频率',
          pattern: '模式匹配'
        }
      },
      template: {
        title: '通知模板',
        add: '新增模板',
        edit: '编辑模板',
        name: '名称',
        format: '格式',
        type: '类型',
        content: '模板内容',
        preview: '预览',
        mockData: '模拟数据',
        refreshPreview: '刷新预览',
        deleteConfirmTitle: '确认删除',
        deleteConfirmContent: '确认删除模板 "{name}" 吗?',
        types: {
          user: '自定义',
          system: '系统'
        },
        formats: {
          html: 'HTML',
          text: '文本',
          markdown: 'Markdown',
          json: 'JSON'
        },
        placeholder: {
          name: '模板名称',
          format: '格式',
          type: '类型',
          mockData: '模拟数据 (JSON)'
        }
      },
      log: {
        title: '通知日志',
        status: '状态',
        channel: '渠道',
        refresh: '刷新',
        id: 'ID',
        eventTitle: '标题',
        eventType: '类型',
        level: '级别',
        sentAt: '发送时间',
        message: '消息内容',
        error: '错误信息',
        job: '队列状态',
        jobStatus: '队列状态',
        nextRunAt: '下次执行时间',
        statuses: {
          pending: '等待中',
          sending: '发送中',
          success: '成功',
          failed: '失败'
        },
        jobStatuses: {
          queued: '排队中',
          processing: '处理中',
          succeeded: '已完成',
          failed: '失败'
        },
	        actions: {
	          delete: '删除',
	          clear: '清空',
	          clearConfirm: '确认清空所有记录吗？',
	          deleteConfirm: '确认删除该记录吗？',
	          batchDeleteConfirm: '确认删除选中的 {count} 条记录吗？'
	        }
	      }
	    },
    userCenter: {
      profile: '个人资料',
      preferences: '偏好设置',
      notificationSettings: '通知设置',
      inAppNotificationLevel: '应用内通知级别',
      selectMinLevel: '选择最低级别',
      savePreferences: '保存偏好设置',
      note: '提示',
      noteContent: '只有级别等于或高于所选级别的通知才会显示在全局头部。',
      saveSuccess: '偏好设置保存成功',
      saveFailed: '偏好设置保存失败',
      levels: {
        info: '信息',
        warning: '警告',
        error: '错误',
        critical: '严重'
      },
      username: '用户名',
      roles: '角色'
    }
  },
  form: {
    required: '不能为空',
    userName: {
      required: '请输入用户名',
      invalid: '用户名格式不正确'
    },
    phone: {
      required: '请输入手机号',
      invalid: '手机号格式不正确'
    },
    pwd: {
      required: '请输入密码',
      invalid: '密码格式不正确，6-18位字符，包含字母、数字、下划线'
    },
    confirmPwd: {
      required: '请输入确认密码',
      invalid: '两次输入密码不一致'
    },
    code: {
      required: '请输入验证码',
      invalid: '验证码格式不正确'
    },
    email: {
      required: '请输入邮箱',
      invalid: '邮箱格式不正确'
    }
  },
  dropdown: {
    closeCurrent: '关闭',
    closeOther: '关闭其它',
    closeLeft: '关闭左侧',
    closeRight: '关闭右侧',
    closeAll: '关闭所有',
    pin: '固定标签',
    unpin: '取消固定'
  },
  icon: {
    themeConfig: '主题配置',
    themeSchema: '主题模式',
    lang: '切换语言',
    fullscreen: '全屏',
    fullscreenExit: '退出全屏',
    reload: '刷新页面',
    collapse: '折叠菜单',
    expand: '展开菜单',
    pin: '固定',
    unpin: '取消固定'
  },
  datatable: {
    itemCount: '共 {total} 条',
    fixed: {
      left: '左固定',
      right: '右固定',
      unFixed: '取消固定'
    }
  }
};

export default local;
