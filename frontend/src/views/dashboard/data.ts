export interface StatCard {
    id: string;
    title: string;
    value: number | string;
    unit?: string;
    icon?: string;
    color?: string;
    trend?: {
        value: number;
        dir: 'up' | 'down';
    };
}

export function getStatCards(): StatCard[] {
    return [
        { id: 'req', title: '请求次数', value: 183, icon: 'carbon:http', color: '#3b82f6' },
        { id: 'pv', title: '访问次数 (PV)', value: 41, icon: 'carbon:view', color: '#10b981' },
        { id: 'uv', title: '独立访客 (UV)', value: 23, icon: 'carbon:user', color: '#8b5cf6' },
        { id: 'ip', title: '独立 IP', value: 48, icon: 'carbon:nacl', color: '#f59e0b' },
        { id: 'block', title: '拦截次数', value: 8, icon: 'carbon:security', color: '#ef4444' },
        { id: 'attack', title: '攻击 IP', value: 6, icon: 'carbon:warning-alt', color: '#f97316' }
    ];
}

export function getErrorStats() {
    return [
        { title: '4xx 错误数', value: 20, rate: '10.93%', type: 'error' },
        { title: '4xx 拦截数', value: 8, rate: '4.37%', type: 'info' },
        { title: '5xx 错误数', value: 11, rate: '6.01%', type: 'error' }
    ];
}

export function getTrendData() {
    const times = [];
    const qps = [];
    const now = new Date();
    for (let i = 0; i < 60; i++) {
        times.push(new Date(now.getTime() - (60 - i) * 1000).toLocaleTimeString());
        qps.push(Math.floor(Math.random() * 50));
    }
    return { times, qps };
}

export function getMapData() {
    return [
        { name: '中国', value: 149 },
        { name: '美国', value: 22 },
        { name: '德国', value: 6 },
        { name: '英国', value: 3 }
    ];
}
