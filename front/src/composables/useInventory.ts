import { useApi } from "@composables/useApi";

export interface InventoryItem {
	id: string;
	name: string;
	units: number;
	price: number;
	deleted: boolean;
	createdAt: number;
	updatedAt: number;
}

export interface InventoryResponse {
	items: InventoryItem[];
	hasNextPage: boolean;
	hasPreviousPage: boolean;
	startCursor: string;
	endCursor: string;
}

export interface InventoryAuditFieldChange {
	from: unknown;
	to: unknown;
}

export interface InventoryAuditEntry {
	id: string;
	itemId: string;
	operation: "create" | "update" | "delete" | "restore" | "restock";
	changes: Record<string, InventoryAuditFieldChange>;
	actorAccountId: string;
	actorName: string;
	actorEmail: string;
	createdAt: number;
}

export interface InventoryItemDetails {
	item: InventoryItem;
	history: InventoryAuditEntry[];
}

interface GetInventoryInput {
	items: 10 | 20 | 50;
	after?: string;
	name?: string;
	unitsMin?: number;
	unitsMax?: number;
	priceMin?: number;
	priceMax?: number;
	sortBy?: "createdAt" | "name" | "units" | "price";
	sortOrder?: "asc" | "desc";
}

interface UpdateInventoryItemInput {
	name?: string;
	units?: number;
	price?: number;
}

interface CreateInventoryItemInput {
	name: string;
	units: number;
	price: number;
}

interface RestockItemInput {
	units: number;
	paymentMethod: "credit_card" | "checking_account";
}

export interface RestockResult {
	order: {
		id: string;
		deliveryAt: number;
	};
	deliverySeconds: number;
}

export function useInventory() {
	const { request } = useApi();

	async function getInventory(input: GetInventoryInput): Promise<InventoryResponse> {
		const query = new URLSearchParams();
		query.set("items", String(input.items));
		if (input.after && input.after.trim() !== "") {
			query.set("after", input.after);
		}
		if (input.name && input.name.trim() !== "") {
			query.set("name", input.name.trim());
		}
		if (typeof input.unitsMin === "number") {
			query.set("unitsMin", String(input.unitsMin));
		}
		if (typeof input.unitsMax === "number") {
			query.set("unitsMax", String(input.unitsMax));
		}
		if (typeof input.priceMin === "number") {
			query.set("priceMin", String(input.priceMin));
		}
		if (typeof input.priceMax === "number") {
			query.set("priceMax", String(input.priceMax));
		}
		if (input.sortBy) {
			query.set("sortBy", input.sortBy);
		}
		if (input.sortOrder) {
			query.set("sortOrder", input.sortOrder);
		}

		return request<InventoryResponse>(`/api/inventory?${query.toString()}`, {
			method: "GET",
		});
	}

	async function getDeletedInventory(input: GetInventoryInput): Promise<InventoryResponse> {
		const query = new URLSearchParams();
		query.set("items", String(input.items));
		if (input.after && input.after.trim() !== "") {
			query.set("after", input.after);
		}
		if (input.name && input.name.trim() !== "") {
			query.set("name", input.name.trim());
		}
		if (typeof input.unitsMin === "number") {
			query.set("unitsMin", String(input.unitsMin));
		}
		if (typeof input.unitsMax === "number") {
			query.set("unitsMax", String(input.unitsMax));
		}
		if (typeof input.priceMin === "number") {
			query.set("priceMin", String(input.priceMin));
		}
		if (typeof input.priceMax === "number") {
			query.set("priceMax", String(input.priceMax));
		}
		if (input.sortBy) {
			query.set("sortBy", input.sortBy);
		}
		if (input.sortOrder) {
			query.set("sortOrder", input.sortOrder);
		}

		return request<InventoryResponse>(`/api/inventory/deleted?${query.toString()}`, {
			method: "GET",
		});
	}

	async function updateItem(id: string, input: UpdateInventoryItemInput): Promise<InventoryItem> {
		return request<InventoryItem>(`/api/inventory/${id}`, {
			method: "PATCH",
			body: input,
		});
	}

	async function createItem(input: CreateInventoryItemInput): Promise<InventoryItem> {
		return request<InventoryItem>("/api/inventory", {
			method: "POST",
			body: input,
		});
	}

	async function getItemDetails(id: string): Promise<InventoryItemDetails> {
		return request<InventoryItemDetails>(`/api/inventory/${id}`, {
			method: "GET",
		});
	}

	async function deleteItem(id: string): Promise<void> {
		await request<void>(`/api/inventory/${id}`, {
			method: "DELETE",
		});
	}

	async function restoreItem(id: string): Promise<InventoryItem> {
		return request<InventoryItem>(`/api/inventory/${id}/restore`, {
			method: "PATCH",
		});
	}

	async function restockItem(id: string, input: RestockItemInput): Promise<RestockResult> {
		return request<RestockResult>(`/api/inventory/${id}/restock`, {
			method: "PATCH",
			body: input,
		});
	}

	return {
		getInventory,
		getDeletedInventory,
		getItemDetails,
		createItem,
		updateItem,
		deleteItem,
		restoreItem,
		restockItem,
	};
}