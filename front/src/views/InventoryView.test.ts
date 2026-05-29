import { describe, it, expect, beforeEach, vi } from "vitest";
import { setActivePinia } from "pinia";
import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";
import { NMessageProvider, NDialogProvider, NConfigProvider } from "naive-ui";
import { nextTick, h } from "vue";

import InventoryView from "@views/InventoryView.vue";
import { mockFetch, mockFetchResponse, mockFetchError, resetFetchMock } from "@test/setup";

const emptyInventoryResponse = {
	items: [],
	hasNextPage: false,
	hasPreviousPage: false,
	startCursor: "",
	endCursor: "",
};

const sampleItems = [
	{
		id: "item-1",
		name: "Widget A",
		units: 10,
		price: 1500,
		deleted: false,
		createdAt: 1000,
		updatedAt: 1000,
	},
	{
		id: "item-2",
		name: "Widget B",
		units: 5,
		price: 2500,
		deleted: false,
		createdAt: 2000,
		updatedAt: 2000,
	},
];

const sampleInventoryResponse = {
	items: sampleItems,
	hasNextPage: true,
	hasPreviousPage: false,
	startCursor: "cursor-start",
	endCursor: "cursor-end",
};

function mountInventoryView() {
	setActivePinia(
		createTestingPinia({
			stubActions: false,
		}),
	);
	resetFetchMock();

	return mount(
		{
			render: () =>
				h(NConfigProvider, null, {
					default: () =>
						h(NMessageProvider, null, {
							default: () =>
								h(NDialogProvider, null, {
									default: () => h(InventoryView),
								}),
						}),
				}),
		},
	);
}

describe("InventoryView", () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	it("renders the Inventory header", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();

		expect(wrapper.text()).toContain("Inventory");
		expect(wrapper.text()).toContain("Manage product stock and pricing.");
	});

	it("shows empty state when no items exist", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("There are no items saved yet.");
		expect(wrapper.text()).toContain("Create your first product to get started.");
	});

	it("shows item names when items exist", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Widget A");
		expect(wrapper.text()).toContain("Widget B");
		expect(wrapper.text()).toContain("$15.00");
		expect(wrapper.text()).toContain("$25.00");
	});

	it("shows error alert on fetch failure", async () => {
		mockFetch(() => Promise.resolve(mockFetchError(500, "Server error")));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Server error");
	});

	it("shows filter form with expected fields", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Name contains");
		expect(wrapper.text()).toContain("Units min");
		expect(wrapper.text()).toContain("Units max");
		expect(wrapper.text()).toContain("Price min (USD)");
		expect(wrapper.text()).toContain("Price max (USD)");
		expect(wrapper.text()).toContain("Apply filters");
		expect(wrapper.text()).toContain("Clear filters");
	});

	it("shows Create product button", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Create product");
	});

	it("shows Actions buttons for each item", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		const actionsButtons = wrapper.findAll("button").filter((b) => b.text() === "Actions");
		expect(actionsButtons.length).toBe(2);
	});

	it("disables Previous button when at start", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		const prevBtn = wrapper.findAll("button").find((b) => b.text() === "Previous");
		expect(prevBtn?.attributes("disabled")).toBeDefined();
	});

	it("shows pagination info with item count", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Showing 2 items");
	});

	it("shows page size selector", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyInventoryResponse)));
		const wrapper = mountInventoryView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Items per page");
		expect(wrapper.text()).toContain("10");
	});
});
