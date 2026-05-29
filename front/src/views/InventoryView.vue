<script setup lang="ts">
import type { ApiError } from "@composables/useApi";
import {
	currencyFromCents,
	pageSizeOptions,
	paymentMethodOptions,
} from "@composables/useFormatters";
import {
	useInventory,
	type RestockResult,
	type InventoryItem,
	type InventoryItemDetails,
	type InventoryResponse,
} from "@composables/useInventory";
import {
	validateName,
	validateUnits,
	validatePositiveUnits,
	validatePriceInDollars,
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
	NModal,
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
import { type InventorySortColumn, useInventoryTableStore } from "@/stores/inventoryTable";
import { useOrdersSyncStore } from "@/stores/ordersSync";

const { getInventory, getItemDetails, createItem, updateItem, deleteItem, restockItem } =
	useInventory();
const tableState = useInventoryTableStore();
const ordersSyncStore = useOrdersSyncStore();
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
const isCreating = ref(false);
const isUpdating = ref(false);
const isDeleting = ref<string | null>(null);
const isRestocking = ref<string | null>(null);

const errorMessage = ref("");
const filterFormError = ref("");
const createFormError = ref("");
const editFormError = ref("");
const restockFormError = ref("");

const showEditModal = ref(false);
const editingItem = ref<InventoryItem | null>(null);

const showCreateModal = ref(false);
const createName = ref("");
const createUnits = ref<number | null>(0);
const createPrice = ref<number | null>(0);

const showRestockModal = ref(false);
const restockTargetItem = ref<InventoryItem | null>(null);
const restockUnits = ref<number | null>(1);
const restockPaymentMethod = ref<"credit_card" | "checking_account">("credit_card");

const showDetailsModal = ref(false);
const isDetailsLoading = ref(false);
const detailsError = ref("");
const itemDetails = ref<InventoryItemDetails | null>(null);

const editName = ref("");
const editUnits = ref<number | null>(null);
const editPrice = ref<number | null>(null);

const restockTotalCents = computed(() => {
	if (!restockTargetItem.value || restockUnits.value === null || restockUnits.value <= 0) {
		return 0;
	}

	return restockTargetItem.value.price * Math.floor(restockUnits.value);
});

function resetFeedback(): void {
	errorMessage.value = "";
}

function resetFormErrors(): void {
	filterFormError.value = "";
	createFormError.value = "";
	editFormError.value = "";
	restockFormError.value = "";
}

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

async function fetchInventory(after = currentAfter.value): Promise<void> {
	resetFeedback();
	isLoading.value = true;

	const priceMin = dollarsToCents(tableState.priceMinDollars);
	const priceMax = dollarsToCents(tableState.priceMaxDollars);

	try {
		let requestedAfter = after;
		let hasFallenBack = false;

		while (true) {
			const response = await getInventory({
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
		errorMessage.value = apiError.message || "Failed to load inventory.";
	} finally {
		isLoading.value = false;
	}
}

async function goNext(): Promise<void> {
	if (!inventory.value.hasNextPage || inventory.value.endCursor.trim() === "") {
		return;
	}

	cursorHistory.value.push(currentAfter.value);
	await fetchInventory(inventory.value.endCursor);
}

async function goPrevious(): Promise<void> {
	if (cursorHistory.value.length === 0) {
		return;
	}

	const previousAfter = cursorHistory.value.pop() ?? "";
	await fetchInventory(previousAfter);
}

async function changePageSize(value: number): Promise<void> {
	pageSize.value = value as 10 | 20 | 50;
	cursorHistory.value = [];
	await fetchInventory("");
}

async function applyFilters(): Promise<void> {
	resetFeedback();
	filterFormError.value = "";

	const validationError = validateFilters();
	if (validationError) {
		filterFormError.value = validationError;
		return;
	}

	cursorHistory.value = [];
	await fetchInventory("");
}

async function clearFilters(): Promise<void> {
	tableState.clearFilters();
	filterFormError.value = "";
	cursorHistory.value = [];
	await fetchInventory("");
}

function resolveSortColumn(columnKey: unknown): InventorySortColumn | null {
	if (columnKey === "name" || columnKey === "units" || columnKey === "price") {
		return columnKey;
	}

	return null;
}

function getColumnSortOrder(column: InventorySortColumn): "ascend" | "descend" | false {
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
	await fetchInventory("");
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

function openEditModal(item: InventoryItem): void {
	editingItem.value = item;
	editName.value = item.name;
	editUnits.value = item.units;
	editPrice.value = item.price / 100;
	showEditModal.value = true;
	resetFeedback();
	editFormError.value = "";
}

function closeEditModal(): void {
	showEditModal.value = false;
	editingItem.value = null;
	editName.value = "";
	editUnits.value = null;
	editPrice.value = null;
	editFormError.value = "";
}

function openCreateModal(): void {
	createName.value = "";
	createUnits.value = 0;
	createPrice.value = 0;
	showCreateModal.value = true;
	resetFeedback();
	createFormError.value = "";
}

function closeCreateModal(): void {
	showCreateModal.value = false;
	createName.value = "";
	createUnits.value = 0;
	createPrice.value = 0;
	createFormError.value = "";
}

function openRestockModal(item: InventoryItem): void {
	restockTargetItem.value = item;
	restockUnits.value = 1;
	restockPaymentMethod.value = "credit_card";
	restockFormError.value = "";
	showRestockModal.value = true;
}

function closeRestockModal(): void {
	showRestockModal.value = false;
	restockTargetItem.value = null;
	restockUnits.value = 1;
	restockPaymentMethod.value = "credit_card";
	restockFormError.value = "";
}

function scheduleRestockSync(result: RestockResult): void {
	const timeoutMs = result.deliverySeconds * 1000 + 1500;
	setTimeout(() => {
		void fetchInventory();
		ordersSyncStore.bumpRefresh();
	}, timeoutMs);
}

async function handleRestock(): Promise<void> {
	if (!restockTargetItem.value) {
		return;
	}

	restockFormError.value = "";
	const unitsError = validatePositiveUnits(restockUnits.value);
	if (unitsError) {
		restockFormError.value = unitsError;
		return;
	}

	isRestocking.value = restockTargetItem.value.id;
	try {
		const result = await restockItem(restockTargetItem.value.id, {
			units: restockUnits.value as number,
			paymentMethod: restockPaymentMethod.value,
		});

		const etaMinutes = Math.max(1, Math.round(result.deliverySeconds / 60));
		message.success(`Restock order placed. Delivery in about ${etaMinutes} minute(s).`);
		ordersSyncStore.bumpRefresh();
		scheduleRestockSync(result);
		closeRestockModal();
	} catch (error) {
		const apiError = error as ApiError;
		restockFormError.value = apiError.message || "Failed to place restock order.";
	} finally {
		isRestocking.value = null;
	}
}

async function handleCreate(): Promise<void> {
	resetFeedback();
	createFormError.value = "";

	const nameError = validateName(createName.value);
	if (nameError) {
		createFormError.value = nameError;
		return;
	}

	const unitsError = validateUnits(createUnits.value);
	if (unitsError) {
		createFormError.value = unitsError;
		return;
	}

	const priceError = validatePriceInDollars(createPrice.value);
	if (priceError) {
		createFormError.value = priceError;
		return;
	}

	const units = createUnits.value as number;
	const priceInDollars = createPrice.value as number;

	isCreating.value = true;
	try {
		await createItem({
			name: createName.value.trim(),
			units: Math.floor(units),
			price: Math.round(priceInDollars * 100),
		});

		message.success("Item created successfully.");
		closeCreateModal();
		cursorHistory.value = [];
		await fetchInventory("");
	} catch (error) {
		const apiError = error as ApiError;
		createFormError.value = apiError.message || "Failed to create item.";
	} finally {
		isCreating.value = false;
	}
}

async function handleEdit(): Promise<void> {
	if (!editingItem.value) {
		return;
	}

	resetFeedback();
	editFormError.value = "";

	const nameError = validateName(editName.value);
	if (nameError) {
		editFormError.value = nameError;
		return;
	}

	const unitsError = validateUnits(editUnits.value);
	if (unitsError) {
		editFormError.value = unitsError;
		return;
	}

	const priceError = validatePriceInDollars(editPrice.value);
	if (priceError) {
		editFormError.value = priceError;
		return;
	}

	const units = editUnits.value as number;
	const priceInDollars = editPrice.value as number;

	isUpdating.value = true;
	try {
		await updateItem(editingItem.value.id, {
			name: editName.value.trim(),
			units: Math.floor(units),
			price: Math.round(priceInDollars * 100),
		});

		message.success("Item updated successfully.");
		closeEditModal();
		await fetchInventory();
	} catch (error) {
		const apiError = error as ApiError;
		editFormError.value = apiError.message || "Failed to update item.";
	} finally {
		isUpdating.value = false;
	}
}

async function handleDelete(item: InventoryItem): Promise<void> {
	resetFeedback();
	resetFormErrors();
	isDeleting.value = item.id;

	try {
		await deleteItem(item.id);
		message.success("Item deleted successfully.");
		await fetchInventory();
	} catch (error) {
		const apiError = error as ApiError;
		errorMessage.value = apiError.message || "Failed to delete item.";
	} finally {
		isDeleting.value = null;
	}
}

function confirmDelete(item: InventoryItem): void {
	dialog.warning({
		title: "Delete product",
		content: `Are you sure you want to delete "${item.name}"? This action cannot be undone.`,
		positiveText: "Delete",
		negativeText: "Cancel",
		onPositiveClick: async () => {
			await handleDelete(item);
		},
	});
}

const columns = computed<DataTableColumns<InventoryItem>>(() => [
	{
		title: "Name",
		key: "name",
		sorter: true,
		sortOrder: getColumnSortOrder("name"),
	},
	{
		title: "Units",
		key: "units",
		width: 90,
		sorter: true,
		sortOrder: getColumnSortOrder("units"),
	},
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
				{ label: "Restock", key: "restock" },
				{ label: "Edit", key: "edit" },
				{ label: "Delete", key: "delete" },
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

						if (key === "edit") {
							openEditModal(row);
							return;
						}

						if (key === "restock") {
							openRestockModal(row);
							return;
						}

						if (key === "delete") {
							confirmDelete(row);
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
								loading: isDeleting.value === row.id,
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

const showEmptyState = computed(() => !isLoading.value && inventory.value.items.length === 0);

onMounted(async () => {
	await fetchInventory("");
});
</script>

<template>
	<main class="space-y-5">
		<header class="flex flex-wrap items-center justify-between gap-3">
			<div>
				<h1 class="text-2xl font-semibold text-slate-900">Inventory</h1>
				<p class="text-slate-600">Manage product stock and pricing.</p>
			</div>
			<n-space align="center">
				<n-button type="primary" @click="openCreateModal">Create product</n-button>
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
						? 'No items match your current filters.'
						: 'There are no items saved yet.'
				"
				:subtitle="
					hasActiveFilters
						? 'Adjust filters and try again.'
						: 'Create your first product to get started.'
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

		<n-modal
			v-model:show="showCreateModal"
			preset="card"
			title="Create product"
			class="w-full max-w-[520px] rounded-xl"
			:mask-closable="false"
		>
			<n-alert
				v-if="createFormError"
				type="error"
				:show-icon="false"
				closable
				@close="createFormError = ''"
				class="mb-4"
			>
				{{ createFormError }}
			</n-alert>
			<n-form @submit.prevent="handleCreate">
				<n-space vertical>
					<n-form-item label="Name">
						<n-input
							v-model:value="createName"
							placeholder="Product name"
							:maxlength="100"
							show-count
						/>
					</n-form-item>
					<n-form-item label="Units">
						<n-input-number v-model:value="createUnits" :min="0" :precision="0" class="w-full" />
					</n-form-item>
					<n-form-item label="Price (USD)">
						<n-input-number
							v-model:value="createPrice"
							:min="0"
							:precision="2"
							:step="0.01"
							class="w-full"
						/>
					</n-form-item>
					<n-space justify="end">
						<n-button @click="closeCreateModal">Cancel</n-button>
						<n-button type="primary" attr-type="submit" :loading="isCreating">Create</n-button>
					</n-space>
				</n-space>
			</n-form>
		</n-modal>

		<n-modal
			v-model:show="showRestockModal"
			preset="card"
			title="Restock item"
			class="w-full max-w-[520px] rounded-xl"
			:mask-closable="false"
		>
			<n-alert
				v-if="restockFormError"
				type="error"
				:show-icon="false"
				closable
				@close="restockFormError = ''"
				class="mb-4"
			>
				{{ restockFormError }}
			</n-alert>
			<n-form @submit.prevent="handleRestock">
				<n-space vertical>
					<n-form-item label="Item">
						<n-input :value="restockTargetItem?.name ?? ''" disabled />
					</n-form-item>
					<n-form-item label="Units to buy">
						<n-input-number v-model:value="restockUnits" :min="1" :precision="0" class="w-full" />
					</n-form-item>
					<n-form-item label="Payment method">
						<n-select v-model:value="restockPaymentMethod" :options="paymentMethodOptions" />
					</n-form-item>
					<n-form-item label="Total purchase cost">
						<n-input :value="currencyFromCents(restockTotalCents)" disabled />
					</n-form-item>
					<n-space justify="end">
						<n-button @click="closeRestockModal">Cancel</n-button>
						<n-button
							type="primary"
							attr-type="submit"
							:loading="isRestocking === restockTargetItem?.id"
							>Place order</n-button
						>
					</n-space>
				</n-space>
			</n-form>
		</n-modal>

		<n-modal
			v-model:show="showEditModal"
			preset="card"
			title="Edit item"
			class="w-full max-w-[520px] rounded-xl"
			:mask-closable="false"
		>
			<n-alert
				v-if="editFormError"
				type="error"
				:show-icon="false"
				closable
				@close="editFormError = ''"
				class="mb-4"
			>
				{{ editFormError }}
			</n-alert>
			<n-form @submit.prevent="handleEdit">
				<n-space vertical>
					<n-form-item label="Name">
						<n-input
							v-model:value="editName"
							placeholder="Product name"
							:maxlength="100"
							show-count
						/>
					</n-form-item>
					<n-form-item label="Units">
						<n-input-number v-model:value="editUnits" :min="0" :precision="0" class="w-full" />
					</n-form-item>
					<n-form-item label="Price (USD)">
						<n-input-number
							v-model:value="editPrice"
							:min="0"
							:precision="2"
							:step="0.01"
							class="w-full"
						/>
					</n-form-item>
					<n-space justify="end">
						<n-button @click="closeEditModal">Cancel</n-button>
						<n-button type="primary" attr-type="submit" :loading="isUpdating"
							>Save changes</n-button
						>
					</n-space>
				</n-space>
			</n-form>
		</n-modal>
	</main>
</template>