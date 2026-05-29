import { defineStore } from "pinia";

export type InventorySortBy = "createdAt" | "name" | "units" | "price";
export type InventorySortOrder = "asc" | "desc";
export type InventorySortColumn = "name" | "units" | "price";

interface InventoryTableState {
	name: string;
	unitsMin: number | null;
	unitsMax: number | null;
	priceMinDollars: number | null;
	priceMaxDollars: number | null;
	sortBy: InventorySortBy;
	sortOrder: InventorySortOrder;
}

export const useInventoryTableStore = defineStore("inventory-table", {
	state: (): InventoryTableState => ({
		name: "",
		unitsMin: null,
		unitsMax: null,
		priceMinDollars: null,
		priceMaxDollars: null,
		sortBy: "createdAt",
		sortOrder: "desc",
	}),
	actions: {
		setSort(columnKey: InventorySortColumn | null, order: "ascend" | "descend" | false): void {
			if (!columnKey || order === false) {
				this.sortBy = "createdAt";
				this.sortOrder = "desc";
				return;
			}

			this.sortBy = columnKey;
			this.sortOrder = order === "ascend" ? "asc" : "desc";
		},
		clearFilters(): void {
			this.name = "";
			this.unitsMin = null;
			this.unitsMax = null;
			this.priceMinDollars = null;
			this.priceMaxDollars = null;
		},
	},
});