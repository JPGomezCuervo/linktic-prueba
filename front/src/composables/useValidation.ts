const EMAIL_REGEX = /^[\w._%+-]+@[\w.-]+\.[A-Za-z]{2,}$/;
const MAX_NAME_LENGTH = 100;

export function isEmailValid(value: string): boolean {
	return EMAIL_REGEX.test(value);
}

export function validateName(value: string): string | null {
	const trimmed = value.trim();
	if (trimmed === "") {
		return "Name is required.";
	}

	if (trimmed.length > MAX_NAME_LENGTH) {
		return `Name must be ${MAX_NAME_LENGTH} characters or fewer.`;
	}

	return null;
}

export function validateUnits(value: number | null): string | null {
	if (value === null || Number.isNaN(value)) {
		return "Units is required.";
	}

	if (!Number.isFinite(value)) {
		return "Units must be a valid number.";
	}

	if (!Number.isInteger(value)) {
		return "Units must be a whole number.";
	}

	if (value < 0) {
		return "Units must be 0 or greater.";
	}

	return null;
}

export function validatePositiveUnits(value: number | null): string | null {
	if (value === null || Number.isNaN(value)) {
		return "Units is required.";
	}

	if (!Number.isFinite(value)) {
		return "Units must be a valid number.";
	}

	if (!Number.isInteger(value)) {
		return "Units must be a whole number.";
	}

	if (value <= 0) {
		return "Units must be greater than 0.";
	}

	return null;
}

export function validatePriceInDollars(value: number | null): string | null {
	if (value === null || Number.isNaN(value)) {
		return "Price is required.";
	}

	if (!Number.isFinite(value)) {
		return "Price must be a valid number in dollars.";
	}

	if (value < 0) {
		return "Price must be 0 or greater.";
	}

	if (Math.abs(value * 100 - Math.round(value * 100)) > 1e-8) {
		return "Price can have up to 2 decimal places.";
	}

	return null;
}

export function validateOptionalWholeNumber(
	value: number | null,
	fieldName: string,
): string | null {
	if (value === null) {
		return null;
	}

	if (!Number.isFinite(value) || Number.isNaN(value)) {
		return `${fieldName} must be a valid number.`;
	}

	if (!Number.isInteger(value)) {
		return `${fieldName} must be a whole number.`;
	}

	if (value < 0) {
		return `${fieldName} must be 0 or greater.`;
	}

	return null;
}

export function validateOptionalPrice(value: number | null, fieldName: string): string | null {
	if (value === null) {
		return null;
	}

	if (!Number.isFinite(value) || Number.isNaN(value)) {
		return `${fieldName} must be a valid number in dollars.`;
	}

	if (value < 0) {
		return `${fieldName} must be 0 or greater.`;
	}

	if (Math.abs(value * 100 - Math.round(value * 100)) > 1e-8) {
		return `${fieldName} can have up to 2 decimal places.`;
	}

	return null;
}

export function validatePassword(value: string): string | null {
	if (value.trim().length < 8) {
		return "Password must be at least 8 characters.";
	}

	return null;
}

export function dollarsToCents(value: number | null): number | undefined {
	if (value === null) {
		return undefined;
	}

	return Math.round(value * 100);
}