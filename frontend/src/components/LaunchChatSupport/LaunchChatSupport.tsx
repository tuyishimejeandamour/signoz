import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { Button, Tooltip } from 'antd';
import logEvent from 'api/common/logEvent';
import cx from 'classnames';
import { FeatureKeys } from 'constants/features';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import { defaultTo } from 'lodash-es';
import { HelpCircle } from 'lucide-react';
import { useAppContext } from 'providers/App/App';

import './LaunchChatSupport.styles.scss';

export interface LaunchChatSupportProps {
	eventName: string;
	attributes: Record<string, unknown>;
	message?: string;
	buttonText?: string;
	className?: string;
	onHoverText?: string;
	chatMessageDisabled?: boolean;
}

// eslint-disable-next-line sonarjs/cognitive-complexity
function LaunchChatSupport({
	attributes,
	eventName,
	message = '',
	buttonText = '',
	className = '',
	onHoverText = '',
	chatMessageDisabled = false,
}: LaunchChatSupportProps): JSX.Element | null {
	const { isCloudUser: isCloudUserVal } = useGetTenantLicense();
	const {
		featureFlags,
		isFetchingFeatureFlags,
		featureFlagsFetchError,
	} = useAppContext();
	const { pathname } = useLocation();

	const isChatSupportEnabled = useMemo(() => {
		if (!isFetchingFeatureFlags && (featureFlags || featureFlagsFetchError)) {
			let isChatSupportEnabled = false;

			if (featureFlags && featureFlags.length > 0) {
				isChatSupportEnabled =
					featureFlags.find((flag) => flag.name === FeatureKeys.CHAT_SUPPORT)
						?.active || false;
			}
			return isChatSupportEnabled;
		}
		return false;
	}, [featureFlags, featureFlagsFetchError, isFetchingFeatureFlags]);

	// In cloud-only mode, subscription is always converted, so no credit card modal needed
	const showAddCreditCardModal = false;

	const handleFacingIssuesClick = (): void => {
		logEvent(eventName, attributes);
		if (window.pylon && !chatMessageDisabled) {
			window.Pylon?.('showNewMessage', defaultTo(message, ''));
		}
	};

	return isCloudUserVal && isChatSupportEnabled ? ( // Note: we would need to move this condition to license based in future
		<div className="facing-issue-button">
			<Tooltip
				title={onHoverText}
				autoAdjustOverflow
				style={{ padding: 8 }}
				overlayClassName="tooltip-overlay"
			>
				<Button
					className={cx('periscope-btn', 'facing-issue-button', className)}
					onClick={handleFacingIssuesClick}
					icon={<HelpCircle size={14} />}
				>
					{buttonText || 'Facing issues?'}
				</Button>
			</Tooltip>

		</div>
	) : null;
}

LaunchChatSupport.defaultProps = {
	message: '',
	buttonText: '',
	className: '',
	onHoverText: '',
	chatMessageDisabled: false,
};

export default LaunchChatSupport;
