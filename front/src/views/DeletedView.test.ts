import { describe, it, expect, beforeEach, vi } from "vitest";
import { setActivePinia } from "pinia";
import { createTestingPinia } from "@pinia/testing";
import { mount } from "@vue/test-utils";
import { NMessageProvider, NDialogProvider, NConfigProvider } from "naive-ui";
import { nextTick, h } from "vue";

import DeletedView from "@views/DeletedView.vue";
import { mockFetch, mockFetchResponse, mockFetchError, resetFetchMock } from "@test/setup";

const emptyDeletedResponse = {
	items: [],
	hasNextPage: false,
	hasPreviousPage: false,
	startCursor: "",
	endCursor: "",
};

const sampleDeletedItems = [
	{
		id: "deleted-1",
		name: "Old Widget",
		units: 3,
		price: 999,
		deleted: true,
		createdAt: 1000,
		updatedAt: 2000,
	},
];

const sampleDeletedResponse = {
	items: sampleDeletedItems,
	hasNextPage: false,
	hasPreviousPage: false,
	startCursor: "",
	endCursor: "",
};

function mountDeletedView() {
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
									default: () => h(DeletedView),
								}),
						}),
				}),
		},
	);
}

describe("DeletedView", () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	it("renders the Deleted header", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyDeletedResponse)));
		const wrapper = mountDeletedView();
		await nextTick();

		expect(wrapper.text()).toContain("Deleted");
		expect(wrapper.text()).toContain("Restore products that were previously deleted.");
	});

	it("shows empty state when no deleted items exist", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyDeletedResponse)));
		const wrapper = mountDeletedView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("There are no deleted items.");
		expect(wrapper.text()).toContain("Deleted items will appear here.");
	});

	it("shows item names when deleted items exist", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleDeletedResponse)));
		const wrapper = mountDeletedView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Old Widget");
		expect(wrapper.text()).toContain("$9.99");
	});

	it("shows error alert on fetch failure", async () => {
		mockFetch(() => Promise.resolve(mockFetchError(500, "Failed to load deleted items.")));
		const wrapper = mountDeletedView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Failed to load deleted items.");
	});

	it("shows filter form with expected fields", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyDeletedResponse)));
		const wrapper = mountDeletedView();
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

	it("shows Actions button when items exist", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleDeletedResponse)));
		const wrapper = mountDeletedView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Actions");
	});

	it("disables Previous button when at start", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleDeletedResponse)));
		const wrapper = mountDeletedView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		const prevBtn = wrapper.findAll("button").find((b) => b.text() === "Previous");
		expect(prevBtn?.attributes("disabled")).toBeDefined();
	});

	it("disables Next button when no next page", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleDeletedResponse)));
		const wrapper = mountDeletedView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		const nextBtn = wrapper.findAll("button").find((b) => b.text() === "Next");
		expect(nextBtn?.attributes("disabled")).toBeDefined();
	});

	it("shows pagination info with item count", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, sampleDeletedResponse)));
		const wrapper = mountDeletedView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Showing 1 items");
	});

	it("shows page size selector", async () => {
		mockFetch(() => Promise.resolve(mockFetchResponse(200, emptyDeletedResponse)));
		const wrapper = mountDeletedView();
		await nextTick();
		await vi.runAllTimersAsync();
		await nextTick();

		expect(wrapper.text()).toContain("Items per page");
		expect(wrapper.text()).toContain("10");
	});
});
