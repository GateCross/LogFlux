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
