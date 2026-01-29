import type { CustomRoute, ElegantRoute } from '@elegant-router/types';
import { generatedRoutes } from '../elegant/routes';
import { layouts, views } from '../elegant/imports';
import { getRoutePath, transformElegantRoutesToVueRoutes } from '../elegant/transform';

export const ROOT_ROUTE: CustomRoute = {
  name: 'root',
  path: '/',
  redirect: getRoutePath(import.meta.env.VITE_ROUTE_HOME) || '/home',
  meta: {
    title: 'root',
    constant: true
  }
};

const NOT_FOUND_ROUTE: CustomRoute = {
  name: 'not-found',
  path: '/:pathMatch(.*)*',
  component: 'layout.blank$view.404',
  meta: {
    title: 'not-found',
    constant: true
  }
};

/** get constant routes from generated routes */
function getConstantRoutes() {
  const constantRoutes = generatedRoutes.filter(route => route.meta?.constant);

  return constantRoutes.map(route => {
    if (route.name === 'login') {
      route.component = 'layout.blank$view.login' as any;
    }
    if (['403', '404', '500'].includes(route.name)) {
      route.component = `layout.blank$view.${route.name}` as any;
    }
    return route as unknown as ElegantRoute;
  });
}

/** builtin routes, it must be constant and setup in vue-router */
const builtinRoutes: ElegantRoute[] = [ROOT_ROUTE, ...getConstantRoutes(), NOT_FOUND_ROUTE];

/** create builtin vue routes */
export function createBuiltinVueRoutes() {
  return transformElegantRoutesToVueRoutes(builtinRoutes, layouts, views);
}
