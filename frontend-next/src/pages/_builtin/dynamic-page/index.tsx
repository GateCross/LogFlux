/**
 * Dynamic catch-all route handler.
 *
 * This component is registered as a `/*` child of the layout shell in config/routes.ts.
 * It catches all paths that don't match statically registered routes (e.g. /dashboard)
 * and dynamically resolves the correct page component from the route model.
 *
 * How it works:
 *  1. The route model (`useModel('route')`) holds auth routes fetched by `getInitialState`.
 *  2. On mount, this component finds the route matching the current URL.
 *  3. It resolves the page component via webpack `require.context` (same as app.tsx).
 *  4. Renders the resolved component, or a 404 if no match.
 */
import { useLocation, useModel } from '@umijs/max';
import { Spin, Result } from 'antd';
import React, { useState, useEffect, useMemo } from 'react';

/**
 * Webpack require.context: eagerly loads all page .tsx files at build time.
 * Used to resolve backend component identifiers to actual React components.
 */
const pageContext = require.context('@/pages', true, /\.tsx$/);

/**
 * Resolve a component path (e.g. "@/pages/caddy/config") to a React component.
 */
function resolvePageComponent(componentPath: string): React.ComponentType | null {
  const prefix = '@/pages/';
  if (!componentPath.startsWith(prefix)) return null;
  const relative = `.${componentPath.slice(prefix.length)}.tsx`;
  try {
    if (pageContext.keys().includes(relative)) {
      const mod = pageContext(relative);
      return mod.default || mod;
    }
  } catch (e) {
    console.warn('[DynamicPage] Failed to resolve component:', componentPath, e);
  }
  return null;
}

/**
 * Resolve backend component identifier (e.g. "view.caddy_config") to a page file path.
 */
function resolveComponentPath(component: string): string | undefined {
  if (component.includes('$view.')) {
    const viewPart = component.split('$view.')[1];
    if (viewPart) return `@/pages/${viewPart.replace(/_/g, '/')}`;
    return undefined;
  }
  if (component.startsWith('view.')) {
    return `@/pages/${component.slice(5).replace(/_/g, '/')}`;
  }
  if (component.startsWith('layout.')) return undefined;
  return `@/pages/${component.replace(/_/g, '/')}`;
}

/**
 * Recursively search the auth route tree for a route matching the given path.
 */
function findRouteByPath(
  routes: Api.Route.MenuRoute[],
  targetPath: string,
): Api.Route.MenuRoute | undefined {
  for (const route of routes) {
    if (route.path === targetPath) return route;
    if (route.children?.length) {
      const found = findRouteByPath(route.children, targetPath);
      if (found) return found;
    }
  }
  return undefined;
}

export default function DynamicPage() {
  const location = useLocation();
  const routeModel = useModel('route');

  const authRoutes = routeModel?.authRoutes ?? [];
  const constantRoutes = routeModel?.constantRoutes ?? [];
  const isInitAuthRoute = routeModel?.isInitAuthRoute ?? false;

  // Wait for route model to be initialized
  const isReady = isInitAuthRoute;

  // Find matching route and resolve component
  const ResolvedComponent = useMemo(() => {
    if (!isReady) return null;

    const path = location.pathname;
    const allRoutes = [...constantRoutes, ...authRoutes];
    const matchedRoute = findRouteByPath(allRoutes, path);

    if (!matchedRoute?.component) return null;

    const componentPath = resolveComponentPath(matchedRoute.component);
    if (!componentPath) return null;

    return resolvePageComponent(componentPath);
  }, [isReady, location.pathname, authRoutes, constantRoutes]);

  // Loading state while route model initializes
  if (!isReady) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 200 }}>
        <Spin size="large" />
      </div>
    );
  }

  // Component resolved — render it
  if (ResolvedComponent) {
    return <ResolvedComponent />;
  }

  // No matching route — 404
  return (
    <Result
      status="404"
      title="404"
      subTitle={`Page not found: ${location.pathname}`}
    />
  );
}
