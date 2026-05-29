import { defineStore } from "pinia";

export type DeletedSortBy = "createdAt" | "name" | "units" | "price";
export type DeletedSortOrder = "asc" | "desc";
export type DeletedSortColumn = "name" | "units" | "price";

interface DeletedTableState {
	name: string;
	unitsMin: number | null;
	unitsMax: number | null;
	priceMinDollars: number | null;
	priceMaxDollars: number | null;
	sortBy: DeletedSortBy;
	sortOrder: DeletedSortOrder;
}

export const useDeletedTableStore = defineStore("deleted-table", {
	state: (): DeletedTableState => ({
		name: "",
		unitsMin: null,
		unitsMax: null,
		priceMinDollars: null,
		priceMaxDollars: null,
		sortBy: "createdAt",
		sortOrder: "desc",
	}),
	actions: {
		setSort(columnKey: DeletedSortColumn | null, order: "ascend" | "descend" | false): void {
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