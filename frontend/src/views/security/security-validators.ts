import type { FormItemRule } from 'naive-ui';

const dateTimePattern = /^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}$/;

export function createDateTimeValidator(message = '截止时间格式应为 YYYY-MM-DD HH:mm:ss') {
  return {
    validator(_rule: FormItemRule, value: string) {
      const text = String(value || '').trim();
      if (!text) return true;
      if (!dateTimePattern.test(text)) return new Error(message);
      return true;
    },
    trigger: ['blur', 'input'] as const
  };
}

export function createStatusCodeValidator(message = '状态码必须在 100-599 之间') {
  return {
    validator(_rule: FormItemRule, value: number) {
      const num = Number(value);
      if (!Number.isFinite(num) || num < 100 || num > 599) return new Error(message);
      return true;
    },
    trigger: ['blur', 'change'] as const
  };
}

export function createMethodValidator(
  allowedMethods: string[],
  message = 'Method 不合法'
) {
  return {
    validator(_rule: FormItemRule, value: string) {
      const normalized = String(value || '').trim().toUpperCase();
      if (!normalized) return true;
      if (!allowedMethods.some(item => item === normalized)) return new Error(message);
      return true;
    },
    trigger: ['blur', 'change'] as const
  };
}
