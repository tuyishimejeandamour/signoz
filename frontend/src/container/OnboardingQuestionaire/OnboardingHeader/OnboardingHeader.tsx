import './OnboardingHeader.styles.scss';

export function OnboardingHeader(): JSX.Element {
	return (
		<div className="header-container">
			<div className="logo-container">
				<img src="/Logos/trinity-brand-logo.svg" alt="Trinity" />
				<span className="logo-text">Trinity</span>
			</div>
		</div>
	);
}
