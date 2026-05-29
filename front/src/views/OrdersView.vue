<script setup lang="ts">
import type { ApiError } from "@composables/useApi";
import {
	currencyFromCents,
	formatTimestamp,
	formatPaymentMethod,
	formatOrderStatus,
	formatRemainingSeconds,
	pageSizeOptions,
} from "@composables/useFormatters";
import { useOrders, type OrderItem, type OrdersResponse } from "@composables/useOrders";
import {
	NAlert,
	NButton,
	NCard,
	NDataTable,
	NSelect,
	NSpace,
	type DataTableColumns,
} from "naive-ui";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

import EmptyState from "@/components/shared/EmptyState.vue";
import { useOrdersSyncStore } from "@/stores/ordersSync";

const { getOrders } = useOrders();
const ordersSyncStore = useOrdersSyncStore();

const ordersResponse = ref<OrdersResponse>({
	orders: [],
	hasNextPage: false,
	hasPreviousPage: false,
	startCursor: "",
	endCursor: "",
});

const pageSize = ref<10 | 20 | 50>(10);
const currentAfter = ref("");
const cursorHistory = ref<string[]>([]);

const isLoading = ref(false);
const errorMessage = ref("");

let countdownInterval: ReturnType<typeof setInterval> | null = null;
const pendingRefreshTimeouts = new Map<string, ReturnType<typeof setTimeout>>();

function clearRefreshTimeouts(): void {
	for (const timeoutId of pendingRefreshTimeouts.values()) {
		clearTimeout(timeoutId);
	}
	pendingRefreshTimeouts.clear();
}

function schedulePendingRefreshes(): void {
	clearRefreshTimeouts();

	for (const order of ordersResponse.value.orders) {
		if (order.status !== "pending" || order.remainingSeconds <= 0) {
			continue;
		}

		const timeoutMs = order.remainingSeconds * 1000 + 1500;
		const timeoutId = setTimeout(() => {
			pendingRefreshTimeouts.delete(order.id);
			void fetchOrders();
		}, timeoutMs);

		pendingRefreshTimeouts.set(order.id, timeoutId);
	}
}

function startCountdownTicker(): void {
	if (countdownInterval) {
		clearInterval(countdownInterval);
	}

	countdownInterval = setInterval(() => {
		for (const order of ordersResponse.value.orders) {
			if (order.status !== "pending") {
				continue;
			}

			if (order.remainingSeconds > 0) {
				order.remainingSeconds -= 1;
			}

			if (order.remainingSeconds <= 0) {
				order.remainingSeconds = 0;
				order.status = "completed";
			}
		}
	}, 1000);
}

async function fetchOrders(after = currentAfter.value): Promise<void> {
	errorMessage.value = "";
	isLoading.value = true;

	try {
		const response = await getOrders({
			items: pageSize.value,
			after,
		});

		if (!response || response.orders == null) {
			ordersResponse.value = {
				orders: [],
				hasNextPage: false,
				hasPreviousPage: false,
				startCursor: "",
				endCursor: "",
			};
			currentAfter.value = after;
			return;
		}

		ordersResponse.value = response;
		currentAfter.value = after;
		schedulePendingRefreshes();
	} catch (error) {
		const apiError = error as ApiError;
		errorMessage.value = apiError.message || "Failed to load orders.";
	} finally {
		isLoading.value = false;
	}
}

async function goNext(): Promise<void> {
	if (!ordersResponse.value.hasNextPage || ordersResponse.value.endCursor.trim() === "") {
		return;
	}

	cursorHistory.value.push(currentAfter.value);
	await fetchOrders(ordersResponse.value.endCursor);
}

async function goPrevious(): Promise<void> {
	if (cursorHistory.value.length === 0) {
		return;
	}

	const previousAfter = cursorHistory.value.pop() ?? "";
	await fetchOrders(previousAfter);
}

async function changePageSize(value: number): Promise<void> {
	pageSize.value = value as 10 | 20 | 50;
	cursorHistory.value = [];
	await fetchOrders("");
}

const columns = computed<DataTableColumns<OrderItem>>(() => [
	{ title: "Item", key: "itemName" },
	{ title: "Units", key: "units", width: 90 },
	{ title: "Total", key: "totalPrice", render: (row) => currencyFromCents(row.totalPrice) },
	{
		title: "Payment",
		key: "paymentMethod",
		render: (row) => formatPaymentMethod(row.paymentMethod),
	},
	{ title: "Status", key: "status", render: (row) => formatOrderStatus(row.status) },
	{
		title: "Remaining",
		key: "remainingSeconds",
		render: (row) => formatRemainingSeconds(row.remainingSeconds),
	},
	{ title: "Created", key: "createdAt", render: (row) => formatTimestamp(row.createdAt) },
]);

const showEmptyState = computed(() => !isLoading.value && ordersResponse.value.orders.length === 0);

watch(
	() => ordersSyncStore.refreshNonce,
	async () => {
		await fetchOrders();
	},
);

onMounted(async () => {
	startCountdownTicker();
	await fetchOrders("");
});

onBeforeUnmount(() => {
	if (countdownInterval) {
		clearInterval(countdownInterval);
		countdownInterval = null;
	}
	clearRefreshTimeouts();
});
</script>

<template>
	<main class="space-y-5">
		<header class="flex flex-wrap items-center justify-between gap-3">
			<div>
				<h1 class="text-2xl font-semibold text-slate-900">Orders</h1>
				<p class="text-slate-600">Track restock orders and delivery progress.</p>
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
			<empty-state
				v-if="showEmptyState"
				title="No orders yet."
				subtitle="Restock orders will appear here once placed."
			/>
			<n-data-table
				v-else
				:columns="columns"
				:data="ordersResponse.orders"
				:loading="isLoading"
				:bordered="false"
				:single-line="false"
			/>

			<template #footer>
				<div class="flex items-center justify-between gap-3">
					<n-button :disabled="cursorHistory.length === 0 || isLoading" @click="goPrevious"
						>Previous</n-button
					>
					<n-space align="center">
						<span class="text-sm text-slate-600"
							>Showing {{ ordersResponse.orders.length }} orders</span
						>
						<n-button
							type="primary"
							:disabled="!ordersResponse.hasNextPage || isLoading"
							@click="goNext"
							>Next</n-button
						>
					</n-space>
				</div>
			</template>
		</n-card>
	</main>
</template>