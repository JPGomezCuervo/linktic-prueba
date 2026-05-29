import { NTag } from "naive-ui";
import { h } from "vue";

export function currencyFromCents(cents: number): string {
	return new Intl.NumberFormat("en-US", {
		style: "currency",
		currency: "USD",
		minimumFractionDigits: 2,
		maximumFractionDigits: 2,
	}).format(cents / 100);
}

export function formatTimestamp(epochSeconds: number): string {
	return new Date(epochSeconds * 1000).toLocaleString();
}

export function formatPaymentMethod(value: "credit_card" | "checking_account"): string {
	if (value === "credit_card") {
		return "Linktic's credit card";
	}

	return "Linktic's checking account";
}

export function formatOrderStatus(status: "pending" | "completed") {
	if (status === "completed") {
		return h(NTag, { type: "success", size: "small", round: true }, { default: () => "Delivered" });
	}

	return h(NTag, { type: "warning", size: "small", round: true }, { default: () => "In transit" });
}

export function formatRemainingSeconds(value: number): string {
	if (value <= 0) {
		return "Delivered";
	}

	const minutes = Math.floor(value / 60);
	const seconds = value % 60;
	return `${minutes}m ${String(seconds).padStart(2, "0")}s`;
}

export const pageSizeOptions = [
	{ label: "10", value: 10 },
	{ label: "20", value: 20 },
	{ label: "50", value: 50 },
];

export const paymentMethodOptions = [
	{ label: "Linktic's credit card", value: "credit_card" },
	{ label: "Linktic's checking account", value: "checking_account" },
];