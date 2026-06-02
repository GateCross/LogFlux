/**
 * Service barrel export.
 *
 * Re-exports all service modules so consumers can do:
 *   import { fetchLogin, fetchCaddyServers, ... } from '@/services';
 */
export * from './auth';
export * from './route';
export * from './dashboard';
export * from './role';
export * from './system-log';
export * from './caddy';
export * from './caddy-source';
export * from './caddy-policy';
export * from './caddy-observe';
export * from './caddy-release-job';
export * from './caddy-simple-waf';
export * from './caddy-integration';
export * from './cron';
export * from './notification';
export * from './ip-region';
