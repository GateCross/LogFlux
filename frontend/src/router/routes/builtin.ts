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
  return generatedRoutes.map(route => {
    if (route.name === 'login') {
      route.component = 'layout.blank$view.login';
    }
    return route;
  }).filter(route => route.meta?.constant);
}

/** builtin routes, it must be constant and setup in vue-router */
const builtinRoutes: ElegantRoute[] = [ROOT_ROUTE, ...getConstantRoutes(), NOT_FOUND_ROUTE];

/** create builtin vue routes */
export function createBuiltinVueRoutes() {
  return transformElegantRoutesToVueRoutes(builtinRoutes, layouts, views);
}
