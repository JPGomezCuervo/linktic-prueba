import { describe, it, expect, beforeEach } from "vitest";
import { setActivePinia } from "pinia";
import { createTestingPinia } from "@pinia/testing";

import { useInventoryTableStore } from "@/stores/inventoryTable";

describe("useInventoryTableStore", () => {
	beforeEach(() => {
		setActivePinia(
			createTestingPinia({
				stubActions: false,
			}),
		);
	});

	it("initializes with correct default state", () => {
		const store = useInventoryTableStore();
		expect(store.name).toBe("");
		expect(store.unitsMin).toBeNull();
		expect(store.unitsMax).toBeNull();
		expect(store.priceMinDollars).toBeNull();
		expect(store.priceMaxDollars).toBeNull();
		expect(store.sortBy).toBe("createdAt");
		expect(store.sortOrder).toBe("desc");
	});

	it("setSort updates sortBy and sortOrder for ascend", () => {
		const store = useInventoryTableStore();
		store.setSort("name", "ascend");
		expect(store.sortBy).toBe("name");
		expect(store.sortOrder).toBe("asc");
	});

	it("setSort updates sortBy and sortOrder for descend", () => {
		const store = useInventoryTableStore();
		store.setSort("price", "descend");
		expect(store.sortBy).toBe("price");
		expect(store.sortOrder).toBe("desc");
	});

	it("setSort resets to defaults when columnKey is null", () => {
		const store = useInventoryTableStore();
		store.sortBy = "name";
		store.sortOrder = "asc";
		store.setSort(null, false);
		expect(store.sortBy).toBe("createdAt");
		expect(store.sortOrder).toBe("desc");
	});

	it("setSort resets to defaults when order is false", () => {
		const store = useInventoryTableStore();
		store.sortBy = "units";
		store.sortOrder = "asc";
		store.setSort("units", false);
		expect(store.sortBy).toBe("createdAt");
		expect(store.sortOrder).toBe("desc");
	});

	it("clearFilters resets all filter fields but preserves sort", () => {
		const store = useInventoryTableStore();
		store.name = "test";
		store.unitsMin = 5;
		store.unitsMax = 10;
		store.priceMinDollars = 1.0;
		store.priceMaxDollars = 5.0;
		store.sortBy = "price";
		store.sortOrder = "asc";

		store.clearFilters();

		expect(store.name).toBe("");
		expect(store.unitsMin).toBeNull();
		expect(store.unitsMax).toBeNull();
		expect(store.priceMinDollars).toBeNull();
		expect(store.priceMaxDollars).toBeNull();
		expect(store.sortBy).toBe("price");
		expect(store.sortOrder).toBe("asc");
	});
});
