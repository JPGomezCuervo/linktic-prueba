import { useApi } from "@composables/useApi";

export interface OrderItem {
	id: string;
	accountId: string;
	itemId: string;
	itemName: string;
	units: number;
	unitPrice: number;
	totalPrice: number;
	paymentMethod: "credit_card" | "checking_account";
	status: "pending" | "completed";
	deliveryAt: number;
	completedAt?: number;
	createdAt: number;
	updatedAt: number;
	remainingSeconds: number;
}

export interface OrdersResponse {
	orders: OrderItem[];
	hasNextPage: boolean;
	hasPreviousPage: boolean;
	startCursor: string;
	endCursor: string;
}

interface GetOrdersInput {
	items: 10 | 20 | 50;
	after?: string;
}

export function useOrders() {
	const { request } = useApi();

	async function getOrders(input: GetOrdersInput): Promise<OrdersResponse> {
		const query = new URLSearchParams();
		query.set("items", String(input.items));
		if (input.after && input.after.trim() !== "") {
			query.set("after", input.after);
		}

		return request<OrdersResponse>(`/api/orders?${query.toString()}`, {
			method: "GET",
		});
	}

	return {
		getOrders,
	};
}