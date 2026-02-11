import { ReactChild, useCallback, useEffect, useMemo, useState } from 'react';
import { useQuery } from 'react-query';
import { matchPath, useLocation } from 'react-router-dom';
import getLocalStorageApi from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import getAll from 'api/v1/user/get';
import { FeatureKeys } from 'constants/features';
import { LOCALSTORAGE } from 'constants/localStorage';
import { ORG_PREFERENCES } from 'constants/orgPreferences';
import ROUTES from 'constants/routes';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import history from 'lib/history';
import { isEmpty } from 'lodash-es';
import { useAppContext } from 'providers/App/App';
import { SuccessResponseV2 } from 'types/api';
import APIError from 'types/api/error';
import { OrgPreference } from 'types/api/preferences/preference';
import { Organization } from 'types/api/user/getOrganization';
import { UserResponse } from 'types/api/user/getUser';
import { USER_ROLES } from 'types/roles';
import { routePermission } from 'utils/permission';

import routes, {
	LIST_LICENSES,
	oldNewRoutesMapping,
	oldRoutes,
	ROUTES_NOT_TO_BE_OVERRIDEN,
	SUPPORT_ROUTE,
} from './routes';

function PrivateRoute({ children }: PrivateRouteProps): JSX.Element {
	const location = useLocation();
	const { pathname } = location;
	const {
		org,
		orgPreferences,
		user,
		isLoggedIn: isLoggedInState,
		isFetchingOrgPreferences,
		featureFlags,
	} = useAppContext();

	const isAdmin = user.role === USER_ROLES.ADMIN;
	const mapRoutes = useMemo(
		() =>
			new Map(
				[...routes, LIST_LICENSES, SUPPORT_ROUTE].map((e) => {
					const currentPath = matchPath(pathname, {
						path: e.path,
					});
					return [currentPath === null ? null : 'current', e];
				}),
			),
		[pathname],
	);
	const isOldRoute = oldRoutes.indexOf(pathname) > -1;
	const currentRoute = mapRoutes.get('current');
	const { isCloudUser: isCloudUserVal } = useGetTenantLicense();

	const [orgData, setOrgData] = useState<Organization | undefined>(undefined);

	const { data: usersData, isFetching: isFetchingUsers } = useQuery<
		SuccessResponseV2<UserResponse[]> | undefined,
		APIError
	>({
		queryFn: () => {
			if (orgData && orgData.id !== undefined) {
				return getAll();
			}
			return undefined;
		},
		queryKey: ['getOrgUser'],
		enabled: !isEmpty(orgData) && user.role === 'ADMIN',
	});

	const checkFirstTimeUser = useCallback((): boolean => {
		const users = usersData?.data || [];

		const remainingUsers = (Array.isArray(users) ? users : []).filter(
			(user) => user.email !== 'admin@signoz.cloud',
		);

		return remainingUsers.length === 1;
	}, [usersData?.data]);

	useEffect(() => {
		if (
			isCloudUserVal &&
			!isFetchingOrgPreferences &&
			orgPreferences &&
			!isFetchingUsers &&
			usersData &&
			usersData.data
		) {
			const isOnboardingComplete = orgPreferences?.find(
				(preference: OrgPreference) =>
					preference.name === ORG_PREFERENCES.ORG_ONBOARDING,
			)?.value;

			const isFirstUser = checkFirstTimeUser();
			if (
				isFirstUser &&
				!isOnboardingComplete &&
				// if the current route is allowed to be overriden by org onboarding then only do the same
				!ROUTES_NOT_TO_BE_OVERRIDEN.includes(pathname)
			) {
				history.push(ROUTES.ONBOARDING);
			}
		}
	}, [
		checkFirstTimeUser,
		isCloudUserVal,
		isFetchingOrgPreferences,
		isFetchingUsers,
		orgPreferences,
		usersData,
		pathname,
	]);

	useEffect(() => {
		if (org && org.length > 0 && org[0].id !== undefined) {
			setOrgData(org[0]);
		}
	}, [org]);

	// if the feature flag is enabled and the current route is /get-started then redirect to /get-started-with-signoz-cloud
	useEffect(() => {
		if (
			currentRoute?.path === ROUTES.GET_STARTED &&
			featureFlags?.find((e) => e.name === FeatureKeys.ONBOARDING_V3)?.active
		) {
			history.push(ROUTES.GET_STARTED_WITH_CLOUD);
		}
	}, [currentRoute, featureFlags]);

	// eslint-disable-next-line sonarjs/cognitive-complexity
	useEffect(() => {
		// if it is an old route navigate to the new route
		if (isOldRoute) {
			const redirectUrl = oldNewRoutesMapping[pathname];

			const newLocation = {
				...location,
				pathname: redirectUrl,
			};
			history.replace(newLocation);
			return;
		}

		// if the current route is public dashboard then don't redirect to login
		const isPublicDashboard = currentRoute?.path === ROUTES.PUBLIC_DASHBOARD;

		if (isPublicDashboard) {
			return;
		}

		// if the current route
		if (currentRoute) {
			const { isPrivate, key } = currentRoute;
			if (isPrivate) {
				if (isLoggedInState) {
					const route = routePermission[key];
					if (route && route.find((e) => e === user.role) === undefined) {
						history.push(ROUTES.UN_AUTHORIZED);
					}
				} else {
					setLocalStorageApi(LOCALSTORAGE.UNAUTHENTICATED_ROUTE_HIT, pathname);
					history.push(ROUTES.LOGIN);
				}
			} else if (isLoggedInState) {
				const fromPathname = getLocalStorageApi(
					LOCALSTORAGE.UNAUTHENTICATED_ROUTE_HIT,
				);
				if (fromPathname) {
					history.push(fromPathname);
					setLocalStorageApi(LOCALSTORAGE.UNAUTHENTICATED_ROUTE_HIT, '');
				} else if (pathname !== ROUTES.SOMETHING_WENT_WRONG) {
					history.push(ROUTES.HOME);
				}
			} else {
				// do nothing as the unauthenticated routes are LOGIN and SIGNUP and the LOGIN container takes care of routing to signup if
				// setup is not completed
			}
		} else if (isLoggedInState) {
			const fromPathname = getLocalStorageApi(
				LOCALSTORAGE.UNAUTHENTICATED_ROUTE_HIT,
			);
			if (fromPathname) {
				history.push(fromPathname);
				setLocalStorageApi(LOCALSTORAGE.UNAUTHENTICATED_ROUTE_HIT, '');
			} else {
				history.push(ROUTES.HOME);
			}
		} else {
			setLocalStorageApi(LOCALSTORAGE.UNAUTHENTICATED_ROUTE_HIT, pathname);
			history.push(ROUTES.LOGIN);
		}
	}, [isLoggedInState, pathname, user, isOldRoute, currentRoute, location]);

	// NOTE: disabling this rule as there is no need to have div
	// eslint-disable-next-line react/jsx-no-useless-fragment
	return <>{children}</>;
}

interface PrivateRouteProps {
	children: ReactChild;
}

export default PrivateRoute;
