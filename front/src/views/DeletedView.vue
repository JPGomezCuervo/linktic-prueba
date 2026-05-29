<script setup lang="ts">
import type { ApiError } from "@composables/useApi";
import { currencyFromCents, pageSizeOptions } from "@composables/useFormatters";
import {
	useInventory,
	type InventoryItem,
	type InventoryItemDetails,
	type InventoryResponse,
} from "@composables/useInventory";
import {
	validateOptionalWholeNumber,
	validateOptionalPrice,
	dollarsToCents,
} from "@composables/useValidation";
import {
	NAlert,
	NButton,
	NCard,
	NDataTable,
	NDropdown,
	NForm,
	NFormItem,
	NInput,
	NInputNumber,
	NSelect,
	NSpace,
	type DataTableColumns,
	type DataTableSortState,
	type DropdownOption,
	useDialog,
	useMessage,
} from "naive-ui";
import { computed, h, onMounted, ref } from "vue";

import ItemDetailsModal from "@/components/inventory/ItemDetailsModal.vue";
import EmptyState from "@/components/shared/EmptyState.vue";
import { type DeletedSortColumn, useDeletedTableStore } from "@/stores/deletedTable";

const { getDeletedInventory, getItemDetails, restoreItem } = useInventory();
const tableState = useDeletedTableStore();
const message = useMessage();
const dialog = useDialog();

const inventory = ref<InventoryResponse>({
	items: [],
	hasNextPage: false,
	hasPreviousPage: false,
	startCursor: "",
	endCursor: "",
});

const pageSize = ref<10 | 20 | 50>(10);
const currentAfter = ref("");
const cursorHistory = ref<string[]>([]);

const isLoading = ref(false);
const isRestoring = ref<string | null>(null);

const errorMessage = ref("");
const filterFormError = ref("");

const showDetailsModal = ref(false);
const isDetailsLoading = ref(false);
const detailsError = ref("");
const itemDetails = ref<InventoryItemDetails | null>(null);

function validateFilters(): string | null {
	const unitsMinError = validateOptionalWholeNumber(tableState.unitsMin, "Units min");
	if (unitsMinError) {
		return unitsMinError;
	}

	const unitsMaxError = validateOptionalWholeNumber(tableState.unitsMax, "Units max");
	if (unitsMaxError) {
		return unitsMaxError;
	}

	if (
		tableState.unitsMin !== null &&
		tableState.unitsMax !== null &&
		tableState.unitsMin > tableState.unitsMax
	) {
		return "Units min cannot be greater than units max.";
	}

	const priceMinError = validateOptionalPrice(tableState.priceMinDollars, "Price min");
	if (priceMinError) {
		return priceMinError;
	}

	const priceMaxError = validateOptionalPrice(tableState.priceMaxDollars, "Price max");
	if (priceMaxError) {
		return priceMaxError;
	}

	if (
		tableState.priceMinDollars !== null &&
		tableState.priceMaxDollars !== null &&
		tableState.priceMinDollars > tableState.priceMaxDollars
	) {
		return "Price min cannot be greater than price max.";
	}

	return null;
}

async function fetchDeletedInventory(after = currentAfter.value): Promise<void> {
	errorMessage.value = "";
	isLoading.value = true;

	const priceMin = dollarsToCents(tableState.priceMinDollars);
	const priceMax = dollarsToCents(tableState.priceMaxDollars);

	try {
		let requestedAfter = after;
		let hasFallenBack = false;

		while (true) {
			const response = await getDeletedInventory({
				items: pageSize.value,
				after: requestedAfter,
				name: tableState.name,
				unitsMin: tableState.unitsMin === null ? undefined : tableState.unitsMin,
				unitsMax: tableState.unitsMax === null ? undefined : tableState.unitsMax,
				priceMin,
				priceMax,
				sortBy: tableState.sortBy,
				sortOrder: tableState.sortOrder,
			});

			if (!response || response.items == null) {
				inventory.value = {
					items: [],
					hasNextPage: false,
					hasPreviousPage: false,
					startCursor: "",
					endCursor: "",
				};
				currentAfter.value = requestedAfter;
				break;
			}

			if (
				response.items.length === 0 &&
				requestedAfter !== "" &&
				cursorHistory.value.length > 0 &&
				!hasFallenBack
			) {
				requestedAfter = cursorHistory.value.pop() ?? "";
				hasFallenBack = true;
				continue;
			}

			inventory.value = response;
			currentAfter.value = requestedAfter;
			break;
		}
	} catch (error) {
		const apiError = error as ApiError;
		errorMessage.value = apiError.message || "Failed to load deleted items.";
	} finally {
		isLoading.value = false;
	}
}

async function openDetailsModal(itemID: string): Promise<void> {
	showDetailsModal.value = true;
	isDetailsLoading.value = true;
	detailsError.value = "";
	itemDetails.value = null;

	try {
		itemDetails.value = await getItemDetails(itemID);
	} catch (error) {
		const apiError = error as ApiError;
		detailsError.value = apiError.message || "Failed to load item details.";
	} finally {
		isDetailsLoading.value = false;
	}
}

function closeDetailsModal(): void {
	showDetailsModal.value = false;
	itemDetails.value = null;
	detailsError.value = "";
}

function getRowProps(row: InventoryItem): { style: string; onClick: () => void } {
	return {
		style: "cursor: pointer;",
		onClick: () => {
			void openDetailsModal(row.id);
		},
	};
}

async function handleRestore(item: InventoryItem): Promise<void> {
	errorMessage.value = "";
	isRestoring.value = item.id;

	try {
		await restoreItem(item.id);
		message.success("Item restored successfully.");
		await fetchDeletedInventory();
	} catch (error) {
		const apiError = error as ApiError;
		errorMessage.value = apiError.message || "Failed to restore item.";
	} finally {
		isRestoring.value = null;
	}
}

function confirmRestore(item: InventoryItem): void {
	dialog.success({
		title: "Restore product",
		content: `Restore "${item.name}" to active inventory?`,
		positiveText: "Restore",
		negativeText: "Cancel",
		onPositiveClick: async () => {
			await handleRestore(item);
		},
	});
}

async function goNext(): Promise<void> {
	if (!inventory.value.hasNextPage || inventory.value.endCursor.trim() === "") {
		return;
	}

	cursorHistory.value.push(currentAfter.value);
	await fetchDeletedInventory(inventory.value.endCursor);
}

async function goPrevious(): Promise<void> {
	if (cursorHistory.value.length === 0) {
		return;
	}

	const previousAfter = cursorHistory.value.pop() ?? "";
	await fetchDeletedInventory(previousAfter);
}

async function changePageSize(value: number): Promise<void> {
	pageSize.value = value as 10 | 20 | 50;
	cursorHistory.value = [];
	await fetchDeletedInventory("");
}

async function applyFilters(): Promise<void> {
	filterFormError.value = "";
	const validationError = validateFilters();
	if (validationError) {
		filterFormError.value = validationError;
		return;
	}

	cursorHistory.value = [];
	await fetchDeletedInventory("");
}

async function clearFilters(): Promise<void> {
	tableState.clearFilters();
	filterFormError.value = "";
	cursorHistory.value = [];
	await fetchDeletedInventory("");
}

function resolveSortColumn(columnKey: unknown): DeletedSortColumn | null {
	if (columnKey === "name" || columnKey === "units" || columnKey === "price") {
		return columnKey;
	}

	return null;
}

function getColumnSortOrder(column: DeletedSortColumn): "ascend" | "descend" | false {
	if (tableState.sortBy !== column) {
		return false;
	}

	return tableState.sortOrder === "asc" ? "ascend" : "descend";
}

async function handleSorterUpdate(
	sorter: DataTableSortState | DataTableSortState[] | null,
): Promise<void> {
	const sortState = Array.isArray(sorter) ? (sorter[0] ?? null) : sorter;
	tableState.setSort(resolveSortColumn(sortState?.columnKey), sortState?.order ?? false);

	cursorHistory.value = [];
	await fetchDeletedInventory("");
}

const columns = computed<DataTableColumns<InventoryItem>>(() => [
	{ title: "Name", key: "name", sorter: true, sortOrder: getColumnSortOrder("name") },
	{ title: "Units", key: "units", width: 90, sorter: true, sortOrder: getColumnSortOrder("units") },
	{
		title: "Price",
		key: "price",
		sorter: true,
		sortOrder: getColumnSortOrder("price"),
		render: (row) => currencyFromCents(row.price),
	},
	{
		title: "Actions",
		key: "actions",
		width: 140,
		render: (row) => {
			const options: DropdownOption[] = [
				{ label: "Details", key: "details" },
				{ label: "Restore", key: "restore" },
			];

			return h(
				NDropdown,
				{
					options,
					onSelect: (key: string) => {
						if (key === "details") {
							void openDetailsModal(row.id);
							return;
						}

						if (key === "restore") {
							confirmRestore(row);
						}
					},
				},
				{
					default: () =>
						h(
							NButton,
							{
								size: "small",
								tertiary: true,
								loading: isRestoring.value === row.id,
								onClick: (event: MouseEvent) => {
									event.stopPropagation();
								},
							},
							{ default: () => "Actions" },
						),
				},
			);
		},
	},
]);

const hasActiveFilters = computed(
	() =>
		tableState.name.trim() !== "" ||
		tableState.unitsMin !== null ||
		tableState.unitsMax !== null ||
		tableState.priceMinDollars !== null ||
		tableState.priceMaxDollars !== null,
);

const showEmptyState = computed(() => {
	return !isLoading.value && inventory.value.items.length === 0;
});

onMounted(async () => {
	await fetchDeletedInventory("");
});
</script>

<template>
	<main class="space-y-5">
		<header class="flex flex-wrap items-center justify-between gap-3">
			<div>
				<h1 class="text-2xl font-semibold text-slate-900">Deleted</h1>
				<p class="text-slate-600">Restore products that were previously deleted.</p>
			</div>
			<n-space align="center">
				<span class="text-sm text-slate-600">Items per page</span>
				<n-select
					:value="pageSize"
					:options="pageSizeOptions"
					style="width: 120px"
					@update:value="changePageSize"
				/>
			</n-space>
		</header>

		<n-alert
			v-if="errorMessage"
			type="error"
			closable
			:show-icon="false"
			@close="errorMessage = ''"
		>
			{{ errorMessage }}
		</n-alert>

		<n-card :bordered="false" class="rounded-2xl shadow-lg shadow-slate-900/8">
			<n-alert
				v-if="filterFormError"
				type="error"
				:show-icon="false"
				closable
				@close="filterFormError = ''"
				class="mb-4"
			>
				{{ filterFormError }}
			</n-alert>
			<n-form @submit.prevent="applyFilters">
				<div class="grid gap-3 md:grid-cols-2 lg:grid-cols-5">
					<n-form-item label="Name contains">
						<n-input v-model:value="tableState.name" placeholder="Search by name" clearable />
					</n-form-item>
					<n-form-item label="Units min">
						<n-input-number
							v-model:value="tableState.unitsMin"
							:min="0"
							:precision="0"
							class="w-full"
							clearable
						/>
					</n-form-item>
					<n-form-item label="Units max">
						<n-input-number
							v-model:value="tableState.unitsMax"
							:min="0"
							:precision="0"
							class="w-full"
							clearable
						/>
					</n-form-item>
					<n-form-item label="Price min (USD)">
						<n-input-number
							v-model:value="tableState.priceMinDollars"
							:min="0"
							:precision="2"
							:step="0.01"
							class="w-full"
							clearable
						/>
					</n-form-item>
					<n-form-item label="Price max (USD)">
						<n-input-number
							v-model:value="tableState.priceMaxDollars"
							:min="0"
							:precision="2"
							:step="0.01"
							class="w-full"
							clearable
						/>
					</n-form-item>
				</div>
				<n-space justify="end">
					<n-button @click="clearFilters">Clear filters</n-button>
					<n-button type="primary" attr-type="submit" :loading="isLoading">Apply filters</n-button>
				</n-space>
			</n-form>
		</n-card>

		<n-card :bordered="false" class="rounded-2xl shadow-lg shadow-slate-900/8">
			<empty-state
				v-if="showEmptyState"
				:title="
					hasActiveFilters
						? 'No deleted items match your current filters.'
						: 'There are no deleted items.'
				"
				:subtitle="
					hasActiveFilters ? 'Adjust filters and try again.' : 'Deleted items will appear here.'
				"
			/>
			<n-data-table
				v-else
				:columns="columns"
				:data="inventory.items"
				:loading="isLoading"
				:remote="true"
				:row-props="getRowProps"
				:bordered="false"
				:single-line="false"
				@update:sorter="handleSorterUpdate"
			/>

			<template #footer>
				<div class="flex items-center justify-between gap-3">
					<n-button :disabled="cursorHistory.length === 0 || isLoading" @click="goPrevious"
						>Previous</n-button
					>
					<n-space align="center">
						<span class="text-sm text-slate-600">Showing {{ inventory.items.length }} items</span>
						<n-button type="primary" :disabled="!inventory.hasNextPage || isLoading" @click="goNext"
							>Next</n-button
						>
					</n-space>
				</div>
			</template>
		</n-card>

		<item-details-modal
			:show="showDetailsModal"
			:is-loading="isDetailsLoading"
			:error-message="detailsError"
			:details="itemDetails"
			@update:show="showDetailsModal = $event"
			@close="closeDetailsModal"
		/>
	</main>
</template>
