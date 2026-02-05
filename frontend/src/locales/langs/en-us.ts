const local: App.I18n.Schema = {
  system: {
    title: 'LogFlux',
    updateTitle: 'System Version Update Notification',
    updateContent: 'A new version of the system has been detected. Do you want to refresh the page immediately?',
    updateConfirm: 'Refresh immediately',
    updateCancel: 'Later'
  },
  common: {
    action: 'Action',
    add: 'Add',
    addSuccess: 'Add Success',
    addFailed: 'Add Failed',
    backToHome: 'Back to home',
    batchDelete: 'Batch Delete',
    cancel: 'Cancel',
    close: 'Close',
    check: 'Check',
    selectAll: 'Select All',
    expandColumn: 'Expand Column',
    columnSetting: 'Column Setting',
    config: 'Config',
    confirm: 'Confirm',
    delete: 'Delete',
    deleteSuccess: 'Delete Success',
    deleteFailed: 'Delete Failed',
    confirmDelete: 'Are you sure you want to delete?',
    edit: 'Edit',
    warning: 'Warning',
    error: 'Error',
    index: 'Index',
    keywordSearch: 'Please enter keyword',
    logout: 'Logout',
    logoutConfirm: 'Are you sure you want to log out?',
    changePassword: 'Change Password',
    oldPassword: 'Old Password',
    newPassword: 'New Password',
    confirmPassword: 'Confirm Password',
    passwordNoMatch: 'The two passwords do not match',
    changePasswordSuccess: 'Password changed successfully, please log in again',
    lookForward: 'Coming soon',
    modify: 'Modify',
    modifySuccess: 'Modify Success',
    noData: 'No Data',
    operate: 'Operate',
    pleaseCheckValue: 'Please check whether the value is valid',
    refresh: 'Refresh',
    reset: 'Reset',
    search: 'Search',
    switch: 'Switch',
    tip: 'Tip',
    trigger: 'Trigger',
    update: 'Update',
    updateSuccess: 'Update Success',
    updateFailed: 'Update Failed',
    userCenter: 'User Center',
    yesOrNo: {
      yes: 'Yes',
      no: 'No'
    }
  },
  request: {
    logout: 'Logout user after request failed',
    logoutMsg: 'User status is invalid, please log in again',
    logoutWithModal: 'Pop up modal after request failed and then log out user',
    logoutWithModalMsg: 'User status is invalid, please log in again',
    refreshToken: 'The requested token has expired, refresh the token',
    tokenExpired: 'The requested token has expired'
  },
  theme: {
    themeDrawerTitle: 'Theme Configuration',
    tabs: {
      appearance: 'Appearance',
      layout: 'Layout',
      general: 'General',
      preset: 'Preset'
    },
    appearance: {
      themeSchema: {
        title: 'Theme Schema',
        light: 'Light',
        dark: 'Dark',
        auto: 'Follow System'
      },
      grayscale: 'Grayscale',
      colourWeakness: 'Colour Weakness',
      themeColor: {
        title: 'Theme Color',
        primary: 'Primary',
        info: 'Info',
        success: 'Success',
        warning: 'Warning',
        error: 'Error',
        followPrimary: 'Follow Primary'
      },
      themeRadius: {
        title: 'Theme Radius'
      },
      recommendColor: 'Apply Recommended Color Algorithm',
      recommendColorDesc: 'The recommended color algorithm refers to',
      preset: {
        title: 'Theme Presets',
        apply: 'Apply',
        applySuccess: 'Preset applied successfully',
        default: {
          name: 'Default Preset',
          desc: 'LogFlux default theme preset'
        },
        dark: {
          name: 'Dark Preset',
          desc: 'Dark theme preset for night time usage'
        },
        compact: {
          name: 'Compact Preset',
          desc: 'Compact layout preset for small screens'
        },
        azir: {
          name: "Azir's Preset",
          desc: 'It is a cold and elegant preset that Azir likes'
        }
      }
    },
    layout: {
      layoutMode: {
        title: 'Layout Mode',
        vertical: 'Vertical Mode',
        horizontal: 'Horizontal Mode',
        'vertical-mix': 'Vertical Mix Mode',
        'vertical-hybrid-header-first': 'Left Hybrid Header-First',
        'top-hybrid-sidebar-first': 'Top-Hybrid Sidebar-First',
        'top-hybrid-header-first': 'Top-Hybrid Header-First',
        vertical_detail: 'Vertical menu layout, with the menu on the left and content on the right.',
        'vertical-mix_detail':
          'Vertical mix-menu layout, with the primary menu on the dark left side and the secondary menu on the lighter left side.',
        'vertical-hybrid-header-first_detail':
          'Left hybrid layout, with the primary menu at the top, the secondary menu on the dark left side, and the tertiary menu on the lighter left side.',
        horizontal_detail: 'Horizontal menu layout, with the menu at the top and content below.',
        'top-hybrid-sidebar-first_detail':
          'Top hybrid layout, with the primary menu on the left and the secondary menu at the top.',
        'top-hybrid-header-first_detail':
          'Top hybrid layout, with the primary menu at the top and the secondary menu on the left.'
      },
      tab: {
        title: 'Tab Settings',
        visible: 'Tab Visible',
        cache: 'Tag Bar Info Cache',
        cacheTip: 'One-click to open/close global keepalive',
        height: 'Tab Height',
        mode: {
          title: 'Tab Mode',
          slider: 'Slider',
          chrome: 'Chrome',
          button: 'Button'
        },
        closeByMiddleClick: 'Close Tab by Middle Click',
        closeByMiddleClickTip: 'Enable closing tabs by clicking with the middle mouse button'
      },
      header: {
        title: 'Header Settings',
        height: 'Header Height',
        breadcrumb: {
          visible: 'Breadcrumb Visible',
          showIcon: 'Breadcrumb Icon Visible'
        }
      },
      sider: {
        title: 'Sider Settings',
        inverted: 'Dark Sider',
        width: 'Sider Width',
        collapsedWidth: 'Sider Collapsed Width',
        mixWidth: 'Mix Sider Width',
        mixCollapsedWidth: 'Mix Sider Collapse Width',
        mixChildMenuWidth: 'Mix Child Menu Width',
        autoSelectFirstMenu: 'Auto Select First Submenu',
        autoSelectFirstMenuTip:
          'When a first-level menu is clicked, the first submenu is automatically selected and navigated to the deepest level'
      },
      footer: {
        title: 'Footer Settings',
        visible: 'Footer Visible',
        fixed: 'Fixed Footer',
        height: 'Footer Height',
        right: 'Right Footer'
      },
      content: {
        title: 'Content Area Settings',
        scrollMode: {
          title: 'Scroll Mode',
          tip: 'The theme scroll only scrolls the main part, the outer scroll can carry the header and footer together',
          wrapper: 'Wrapper',
          content: 'Content'
        },
        page: {
          animate: 'Page Animate',
          mode: {
            title: 'Page Animate Mode',
            fade: 'Fade',
            'fade-slide': 'Slide',
            'fade-bottom': 'Fade Zoom',
            'fade-scale': 'Fade Scale',
            'zoom-fade': 'Zoom Fade',
            'zoom-out': 'Zoom Out',
            none: 'None'
          }
        },
        fixedHeaderAndTab: 'Fixed Header And Tab'
      }
    },
    general: {
      title: 'General Settings',
      watermark: {
        title: 'Watermark Settings',
        visible: 'Watermark Full Screen Visible',
        text: 'Custom Watermark Text',
        enableUserName: 'Enable User Name Watermark',
        enableTime: 'Show Current Time',
        timeFormat: 'Time Format'
      },
      multilingual: {
        title: 'Multilingual Settings',
        visible: 'Display multilingual button'
      },
      globalSearch: {
        title: 'Global Search Settings',
        visible: 'Display GlobalSearch button'
      }
    },
    configOperation: {
      copyConfig: 'Copy Config',
      copySuccessMsg: 'Copy Success, Please replace the variable "themeSettings" in "src/theme/settings.ts"',
      resetConfig: 'Reset Config',
      resetSuccessMsg: 'Reset Success'
    }
  },
  route: {
    home: 'Home',
    login: 'Login',
    403: 'No Permission',
    404: 'Page Not Found',
    500: 'Server Error',
    'iframe-page': 'Iframe',

    cron: 'Scheduled Tasks',
    dashboard: 'Dashboard',
    caddy: 'Caddy Management',
    caddy_config: 'Caddy Config',
    caddy_log: 'Caddy Logs',
    caddy_source: 'Log Sources',
    manage: 'System Manage',
    manage_user: 'User Management',
    manage_role: 'Role Management',
    manage_menu: 'Menu Management',
    notification: 'Notification',
    notification_channel: 'Channel Management',
    notification_rule: 'Rule Management',
    notification_template: 'Template Management',
    notification_log: 'Notification Log',
    user: 'User Center',
    user_center: 'User Center'
  },
  page: {
    login: {
      common: {
        loginOrRegister: 'Login / Register',
        userNamePlaceholder: 'Please enter user name',
        phonePlaceholder: 'Please enter phone number',
        codePlaceholder: 'Please enter verification code',
        passwordPlaceholder: 'Please enter password',
        confirmPasswordPlaceholder: 'Please enter password again',
        codeLogin: 'Verification code login',
        confirm: 'Confirm',
        back: 'Back',
        validateSuccess: 'Verification passed',
        loginSuccess: 'Login successfully',
        welcomeBack: 'Welcome back, {userName} !'
      },
      pwdLogin: {
        title: 'Password Login',
        rememberMe: 'Remember me',
        forgetPassword: 'Forget password?',
        register: 'Register',
        otherAccountLogin: 'Other Account Login',
        otherLoginMode: 'Other Login Mode',
        superAdmin: 'Super Admin',
        admin: 'Admin',
        user: 'User'
      },
      codeLogin: {
        title: 'Verification Code Login',
        getCode: 'Get verification code',
        reGetCode: 'Reacquire after {time}s',
        sendCodeSuccess: 'Verification code sent successfully',
        imageCodePlaceholder: 'Please enter image verification code'
      },
      register: {
        title: 'Register',
        agreement: 'I have read and agree to',
        protocol: '《User Agreement》',
        policy: '《Privacy Policy》'
      },
      resetPwd: {
        title: 'Reset Password'
      },
      bindWeChat: {
        title: 'Bind WeChat'
      }
    },
    home: {
      branchDesc:
        'For the convenience of everyone in developing and updating the merge, we have streamlined the code of the main branch, only retaining the homepage menu, and the rest of the content has been moved to the example branch for maintenance. The preview address displays the content of the example branch.',
      greeting: 'Good morning, {userName}, today is another day full of vitality!',
      weatherDesc: 'Today is cloudy to clear, 20℃ - 25℃!',
      projectCount: 'Project Count',
      todo: 'Todo',
      message: 'Message',
      downloadCount: 'Download Count',
      registerCount: 'Register Count',
      schedule: 'Work and rest Schedule',
      study: 'Study',
      work: 'Work',
      rest: 'Rest',
      entertainment: 'Entertainment',
      visitCount: 'Visit Count',
      turnover: 'Turnover',
      dealCount: 'Deal Count',
      projectNews: {
        title: 'Project News',
        moreNews: 'More News',
        desc1: 'LogFlux created the log flow analysis management system in January 2026!',
        desc2: 'Team members submitted new feature modules to LogFlux.',
        desc3: 'LogFlux is preparing for the system release!',
        desc4: 'LogFlux is busy writing project documentation!',
        desc5: 'LogFlux just optimized the workbench page!'
      },
      creativity: 'Creativity'
    },
    notification: {
      channel: {
        title: 'Notification Channels',
        add: 'Add Channel',
        edit: 'Edit Channel',
        name: 'Name',
        type: 'Type',
        status: 'Status',
        config: 'Config',
        events: 'Events',
        description: 'Description',
        enabled: 'Enabled',
        disabled: 'Disabled',
        test: 'Test',
        delete: 'Delete',
        deleteConfirmTitle: 'Confirm Delete',
        deleteConfirmContent: 'Are you sure to delete channel "{name}"?',
        testSuccess: 'Test notification sent',
        testFailed: 'Test failed',
        placeholder: {
          name: 'Channel Name',
          type: 'Select Type',
          config: 'JSON Configuration (e.g., { "webhook_url": "..." })',
          events: '["*"] or ["error", "caddy"]',
          description: 'Description'
        }
      },
      rule: {
        title: 'Notification Rules',
        add: 'Add Rule',
        edit: 'Edit Rule',
        name: 'Name',
        ruleType: 'Rule Type',
        eventType: 'Event Type',
        status: 'Status',
        condition: 'Condition',
        channels: 'Channels',
        template: 'Template',
        silence: 'Silence (sec)',
        description: 'Description',
        enabled: 'Enabled',
        disabled: 'Disabled',
        deleteConfirmTitle: 'Confirm Delete',
        deleteConfirmContent: 'Are you sure to delete rule "{name}"?',
        placeholder: {
          name: 'Rule Name',
          type: 'Select Type',
          eventType: 'Event Type (e.g., error)',
          condition: 'JSON Condition (e.g., { "level": "error" })',
          channels: 'Select Channels',
          template: 'Select Template (Optional)',
          silence: '0',
          description: 'Description'
        },
        types: {
          threshold: 'Threshold',
          frequency: 'Frequency',
          pattern: 'Pattern'
        }
      },
      template: {
        title: 'Notification Templates',
        add: 'Add Template',
        edit: 'Edit Template',
        name: 'Name',
        format: 'Format',
        type: 'Type',
        content: 'Content Template',
        preview: 'Preview',
        mockData: 'Mock Data',
        refreshPreview: 'Refresh Preview',
        deleteConfirmTitle: 'Confirm Delete',
        deleteConfirmContent: 'Are you sure to delete template "{name}"?',
        types: {
          user: 'User',
          system: 'System'
        },
        formats: {
          html: 'HTML',
          text: 'Text',
          markdown: 'Markdown',
          json: 'JSON'
        },
        placeholder: {
          name: 'Template Name',
          format: 'Format',
          type: 'Type',
          mockData: 'Mock Data (JSON)'
        }
      },
      log: {
        title: 'Notification Logs',
        status: 'Status',
        channel: 'Channel',
        refresh: 'Refresh',
        id: 'ID',
        eventTitle: 'Title',
        eventType: 'Type',
        level: 'Level',
        sentAt: 'Sent At',
        message: 'Message',
        error: 'Error',
        job: 'Job Status',
        jobStatus: 'Job Status',
        nextRunAt: 'Next Run At',
        statuses: {
          pending: 'Pending',
          sending: 'Sending',
          success: 'Success',
          failed: 'Failed'
        },
        jobStatuses: {
          queued: 'Queued',
          processing: 'Processing',
          succeeded: 'Succeeded',
          failed: 'Failed'
        },
	        actions: {
	          delete: 'Delete',
	          clear: 'Clear',
	          clearConfirm: 'Are you sure you want to clear all records?',
	          deleteConfirm: 'Are you sure you want to delete this record?',
	          batchDeleteConfirm: 'Are you sure you want to delete {count} selected records?'
	        }
	      }
	    },
    userCenter: {
      profile: 'Profile',
      preferences: 'Preferences',
      notificationSettings: 'Notification Settings',
      inAppNotificationLevel: 'In-App Notification Level',
      selectMinLevel: 'Select Minimum Level',
      savePreferences: 'Save Preferences',
      note: 'Note',
      noteContent: 'Only notifications with a level equal to or higher than the selected level will be shown in the global header.',
      saveSuccess: 'Preferences saved successfully',
      saveFailed: 'Failed to save preferences',
      levels: {
        info: 'Info',
        warning: 'Warning',
        error: 'Error',
        critical: 'Critical'
      },
      username: 'Username',
      roles: 'Roles'
    }
  },
  form: {
    required: 'Cannot be empty',
    userName: {
      required: 'Please enter user name',
      invalid: 'User name format is incorrect'
    },
    phone: {
      required: 'Please enter phone number',
      invalid: 'Phone number format is incorrect'
    },
    pwd: {
      required: 'Please enter password',
      invalid: '6-18 characters, including letters, numbers, and underscores'
    },
    confirmPwd: {
      required: 'Please enter password again',
      invalid: 'The two passwords are inconsistent'
    },
    code: {
      required: 'Please enter verification code',
      invalid: 'Verification code format is incorrect'
    },
    email: {
      required: 'Please enter email',
      invalid: 'Email format is incorrect'
    }
  },
  dropdown: {
    closeCurrent: 'Close Current',
    closeOther: 'Close Other',
    closeLeft: 'Close Left',
    closeRight: 'Close Right',
    closeAll: 'Close All',
    pin: 'Pin Tab',
    unpin: 'Unpin Tab'
  },
  icon: {
    themeConfig: 'Theme Configuration',
    themeSchema: 'Theme Schema',
    lang: 'Switch Language',
    fullscreen: 'Fullscreen',
    fullscreenExit: 'Exit Fullscreen',
    reload: 'Reload Page',
    collapse: 'Collapse Menu',
    expand: 'Expand Menu',
    pin: 'Pin',
    unpin: 'Unpin'
  },
  datatable: {
    itemCount: 'Total {total} items',
    fixed: {
      left: 'Left Fixed',
      right: 'Right Fixed',
      unFixed: 'Unfixed'
    }
  }
};

export default local;
