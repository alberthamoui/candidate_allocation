function formatCompactCount(value: string, significantDigitLimit: number) {
	try {
		const count = BigInt(value);
		const zero = BigInt(0);
		const sign = count < zero ? "-" : "";
		const digits = (count < zero ? -count : count).toString();
		if (digits.length <= significantDigitLimit) return count.toLocaleString("pt-BR");

		const significantDigits = digits.slice(0, significantDigitLimit);
		const fraction = significantDigits.slice(1).replace(/0+$/, "");
		const mantissa = fraction
			? `${sign}${significantDigits[0]},${fraction}`
			: `${sign}${significantDigits[0]}`;
		return `${mantissa} * 10**${digits.length - 1}`;
	} catch {
		return value;
	}
}

export function formatPossibilityCount(value: string) {
	return formatCompactCount(value, 7);
}

export function remainingPossibilityCount(total: string, resolved: string) {
	try {
		const remaining = BigInt(total) - BigInt(resolved);
		return (remaining > BigInt(0) ? remaining : BigInt(0)).toString();
	} catch {
		return "0";
	}
}

export function formatRemainingPossibilityCount(total: string, resolved: string) {
	return formatCompactCount(remainingPossibilityCount(total, resolved), 5);
}

export function clampProgressPercent(value: number) {
	if (!Number.isFinite(value)) return 0;
	return Math.min(100, Math.max(0, value));
}
