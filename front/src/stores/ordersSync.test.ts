import { describe, it, expect, beforeEach } from "vitest";
import { setActivePinia } from "pinia";
import { createTestingPinia } from "@pinia/testing";

import { useOrdersSyncStore } from "@/stores/ordersSync";

describe("useOrdersSyncStore", () => {
	beforeEach(() => {
		setActivePinia(
			createTestingPinia({
				stubActions: false,
			}),
		);
	});

	it("initializes with refreshNonce at 0", () => {
		const store = useOrdersSyncStore();
		expect(store.refreshNonce).toBe(0);
	});

	it("bumpRefresh increments nonce by 1", () => {
		const store = useOrdersSyncStore();
		store.bumpRefresh();
		expect(store.refreshNonce).toBe(1);
	});

	it("bumpRefresh can be called multiple times", () => {
		const store = useOrdersSyncStore();
		store.bumpRefresh();
		store.bumpRefresh();
		store.bumpRefresh();
		expect(store.refreshNonce).toBe(3);
	});
});
